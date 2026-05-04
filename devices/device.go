package devices

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"

	"github.com/jgrelet/geo-acq/config"
	"go.bug.st/serial.v1"
)

// Devices is a type that provides access to instrument data,
// as GPS, Echo_sounder, Radar, etc...
type Devices interface {
	Connect(io.ReadWriteCloser) error
	Disconnect() error
	Read()
	Write()
}

// Device represents instrument.
type Device struct {
	name     string
	port     string
	typePort string
	conn     io.ReadWriteCloser
	reader   *bufio.Reader
	mode     *serial.Mode
	openSP   func(port string) (io.ReadWriteCloser, error)
	openUDP  func() (io.ReadWriteCloser, error)
	simul    bool
	logger   *log.Logger
	verbose  bool
	Data     chan string
	Errors   chan error
	initErr  error
}

// New creates a new Device object and connects to the specified transport.
func New(name string, args ...interface{}) *Device {
	dev := &Device{
		name: name,
		openSP: func(port string) (io.ReadWriteCloser, error) {
			return nil, fmt.Errorf("device %q is not configured", port)
		},
		openUDP: func() (io.ReadWriteCloser, error) {
			return nil, fmt.Errorf("device %q is not configured", name)
		},
		logger:  log.New(os.Stdout, fmt.Sprintf("[%s] ", name), log.Ltime),
		verbose: true,
		Data:    make(chan string),
		Errors:  make(chan error, 1),
	}

	for _, arg := range args {
		switch value := arg.(type) {
		case config.Config:
			dev.applyConfig(value)
		case io.ReadWriteCloser:
			dev.conn = value
			dev.reader = bufio.NewReader(value)
		}
	}

	return dev
}

// Connect starts a connection to the configured device.
func (dev *Device) Connect() error {
	if dev.initErr != nil {
		return dev.initErr
	}

	switch dev.typePort {
	case "serial":
		if dev.conn == nil {
			sp, err := dev.openSP(dev.Port())
			if err != nil {
				return err
			}
			dev.conn = sp
			dev.reader = bufio.NewReader(sp)
		}
		if ports, err := SerialGetInfo(); err == nil && len(ports) > 0 {
			dev.logger.Printf("available serial ports: %s", strings.Join(ports, ", "))
		}
	case "udp":
		if dev.conn == nil {
			udpConn, err := dev.openUDP()
			if err != nil {
				return err
			}
			dev.conn = udpConn
			dev.reader = bufio.NewReader(udpConn)
		}
	default:
		return fmt.Errorf("unsupported device transport %q for %s", dev.typePort, dev.name)
	}

	go func() {
		defer close(dev.Data)
		defer close(dev.Errors)
		if dev.typePort == "serial" {
			dev.streamSerial()
			return
		}
		for {
			sentence, err := dev.Read()
			if err != nil {
				if isTransientReadError(err) {
					continue
				}
				select {
				case dev.Errors <- err:
				default:
				}
				return
			}
			if sentence == "" {
				continue
			}
			dev.Data <- sentence
		}
	}()

	return nil
}

func (dev *Device) streamSerial() {
	if dev.conn == nil {
		select {
		case dev.Errors <- errors.New("device is not connected"):
		default:
		}
		return
	}

	buf := make([]byte, 256)
	pending := make([]byte, 0, 1024)

	for {
		n, err := dev.conn.Read(buf)
		if err != nil {
			if isTransientReadError(err) {
				continue
			}
			select {
			case dev.Errors <- err:
			default:
			}
			return
		}
		if n <= 0 {
			continue
		}

		pending = append(pending, buf[:n]...)
		sentences, rest := extractNMEASentences(pending)
		pending = rest
		for _, sentence := range sentences {
			if sentence == "" {
				continue
			}
			dev.Data <- sentence
		}
	}
}

// Disconnect closes the io connection to the device.
func (dev *Device) Disconnect() error {
	if dev.conn == nil {
		return nil
	}
	return dev.conn.Close()
}

// Port returns the device port.
func (dev *Device) Port() string { return dev.port }

// Name returns the device name.
func (dev *Device) Name() string { return dev.name }

// Write writes a sentence to the configured transport.
func (dev *Device) Write(sentence string) error {
	if dev.conn == nil {
		return errors.New("device is not connected")
	}
	_, err := dev.conn.Write([]byte(sentence))
	return err
}

// Read reads a NMEA sentence terminated by CR, LF, or CRLF.
func (dev *Device) Read() (string, error) {
	if dev.reader == nil {
		return "", errors.New("device is not connected")
	}

	var buf []byte
	for {
		b, err := dev.reader.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) && len(buf) > 0 {
				return normalizeSerialLine(string(buf)), nil
			}
			return "", err
		}

		switch b {
		case '\r', '\n':
			if len(buf) == 0 {
				continue
			}
			return normalizeSerialLine(string(buf)), nil
		default:
			buf = append(buf, b)
		}
	}
}

func isTransientReadError(err error) bool {
	return errors.Is(err, syscall.EINTR)
}

func extractNMEASentences(data []byte) ([]string, []byte) {
	var out []string

	for {
		start := bytes.IndexByte(data, '$')
		if start < 0 {
			return out, nil
		}
		data = data[start:]

		end := bytes.IndexAny(data, "\r\n")
		if end < 0 {
			return out, append([]byte(nil), data...)
		}

		line := normalizeSerialLine(string(data[:end]))
		if line != "" {
			out = append(out, line)
		}

		next := end
		for next < len(data) && (data[next] == '\r' || data[next] == '\n') {
			next++
		}
		data = data[next:]
	}
}

func normalizeSerialLine(line string) string {
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return ""
	}

	// When the serial port is opened mid-stream, the first read can start in the
	// middle of a sentence. If we can find a NMEA sentence start marker, resync on it.
	if idx := strings.IndexByte(line, '$'); idx > 0 {
		return line[idx:]
	}

	return line
}

func (dev *Device) applyConfig(cfg config.Config) {
	deviceCfg, ok := cfg.Devices[dev.name]
	if !ok {
		dev.initErr = fmt.Errorf("device %q not found in configuration", dev.name)
		return
	}

	dev.typePort = deviceCfg.Device
	switch dev.typePort {
	case "serial":
		dev.applySerialConfig(cfg)
	case "udp":
		dev.applyUDPConfig(cfg)
	default:
		dev.initErr = fmt.Errorf("unsupported device transport %q", dev.typePort)
	}
}

func (dev *Device) applySerialConfig(cfg config.Config) {
	serialCfg, ok := cfg.Serials[dev.name]
	if !ok {
		dev.initErr = fmt.Errorf("serial config for %q not found", dev.name)
		return
	}
	mode, err := newSerialMode(serialCfg)
	if err != nil {
		dev.initErr = fmt.Errorf("invalid serial config for %q: %w", dev.name, err)
		return
	}
	dev.port = serialCfg.Port
	dev.mode = mode
	dev.openSP = func(port string) (io.ReadWriteCloser, error) {
		p, openErr := serial.Open(port, mode)
		if openErr != nil {
			return nil, fmt.Errorf("can't open serial port %s -> %w", port, openErr)
		}
		return p, nil
	}
}

func (dev *Device) applyUDPConfig(cfg config.Config) {
	udpCfg, ok := cfg.UDP[dev.name]
	if !ok {
		dev.initErr = fmt.Errorf("udp config for %q not found", dev.name)
		return
	}

	conn, err := newUDPConn(udpCfg)
	if err != nil {
		dev.initErr = fmt.Errorf("invalid udp config for %q: %w", dev.name, err)
		return
	}

	dev.port = udpCfg.Port
	dev.openUDP = func() (io.ReadWriteCloser, error) {
		return conn.clone()
	}
}

func newSerialMode(cfg config.SerialPort) (*serial.Mode, error) {
	parity, err := serialParity(cfg.Parity)
	if err != nil {
		return nil, err
	}

	stopBits, err := serialStopBits(cfg.Stopbit)
	if err != nil {
		return nil, err
	}

	if cfg.Baud <= 0 {
		return nil, fmt.Errorf("baud must be > 0")
	}
	if cfg.Databit < 5 || cfg.Databit > 8 {
		return nil, fmt.Errorf("databit must be between 5 and 8")
	}

	return &serial.Mode{
		BaudRate: cfg.Baud,
		DataBits: cfg.Databit,
		Parity:   parity,
		StopBits: stopBits,
	}, nil
}

func serialParity(value string) (serial.Parity, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "none":
		return serial.NoParity, nil
	case "odd":
		return serial.OddParity, nil
	case "even":
		return serial.EvenParity, nil
	case "mark":
		return serial.MarkParity, nil
	case "space":
		return serial.SpaceParity, nil
	default:
		return serial.NoParity, fmt.Errorf("unsupported parity %q", value)
	}
}

func serialStopBits(value int) (serial.StopBits, error) {
	switch value {
	case 0, 1:
		return serial.OneStopBit, nil
	case 2:
		return serial.TwoStopBits, nil
	default:
		return serial.OneStopBit, fmt.Errorf("unsupported stopbit %d", value)
	}
}

// SerialGetInfo retrieves the port list.
func SerialGetInfo() ([]string, error) {
	ports, err := serial.GetPortsList()

	fallback := discoverSerialPortsFallback()
	if err != nil {
		if len(fallback) > 0 {
			return fallback, nil
		}
		return nil, err
	}

	merged := mergePortLists(ports, fallback)
	if len(merged) == 0 {
		return nil, nil
	}
	return merged, nil
}

func discoverSerialPortsFallback() []string {
	if runtime.GOOS != "linux" {
		return nil
	}

	patterns := []string{
		"/dev/ttyUSB*",
		"/dev/ttyACM*",
		"/dev/ttyAMA*",
		"/dev/ttyS*",
		"/dev/rfcomm*",
	}

	var ports []string
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		ports = append(ports, matches...)
	}

	sort.Strings(ports)
	return dedupeStrings(ports)
}

func mergePortLists(primary []string, secondary []string) []string {
	merged := append([]string(nil), primary...)
	merged = append(merged, secondary...)
	sort.Strings(merged)
	return dedupeStrings(merged)
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	out := values[:0]
	var last string
	for i, value := range values {
		if value == "" {
			continue
		}
		if i == 0 || value != last {
			out = append(out, value)
			last = value
		}
	}
	return out
}

type udpConn struct {
	mode       string
	localAddr  string
	remoteAddr string
	conn       *net.UDPConn
	lastRemote *net.UDPAddr
}

func newUDPConn(cfg config.UDP) (*udpConn, error) {
	if strings.TrimSpace(cfg.Port) == "" {
		return nil, fmt.Errorf("port is required")
	}

	mode := "listen"
	localAddr := ":" + cfg.Port
	remoteAddr := ""
	if strings.TrimSpace(cfg.Host) != "" {
		mode = "dial"
		remoteAddr = net.JoinHostPort(cfg.Host, cfg.Port)
	}

	return &udpConn{
		mode:       mode,
		localAddr:  localAddr,
		remoteAddr: remoteAddr,
	}, nil
}

func (c *udpConn) clone() (io.ReadWriteCloser, error) {
	switch c.mode {
	case "dial":
		addr, err := net.ResolveUDPAddr("udp", c.remoteAddr)
		if err != nil {
			return nil, err
		}
		conn, err := net.DialUDP("udp", nil, addr)
		if err != nil {
			return nil, err
		}
		return &udpConn{mode: c.mode, remoteAddr: c.remoteAddr, conn: conn}, nil
	case "listen":
		addr, err := net.ResolveUDPAddr("udp", c.localAddr)
		if err != nil {
			return nil, err
		}
		conn, err := net.ListenUDP("udp", addr)
		if err != nil {
			return nil, err
		}
		return &udpConn{mode: c.mode, localAddr: c.localAddr, conn: conn}, nil
	default:
		return nil, fmt.Errorf("unsupported udp mode %q", c.mode)
	}
}

func (c *udpConn) Read(p []byte) (int, error) {
	if c.conn == nil {
		return 0, errors.New("udp connection is not open")
	}
	if c.mode == "dial" {
		return c.conn.Read(p)
	}
	n, addr, err := c.conn.ReadFromUDP(p)
	if addr != nil {
		c.lastRemote = addr
	}
	return n, err
}

func (c *udpConn) Write(p []byte) (int, error) {
	if c.conn == nil {
		return 0, errors.New("udp connection is not open")
	}
	if c.mode == "dial" {
		return c.conn.Write(p)
	}
	if c.lastRemote == nil {
		return 0, errors.New("udp listener has no remote peer yet")
	}
	return c.conn.WriteToUDP(p, c.lastRemote)
}

func (c *udpConn) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}
