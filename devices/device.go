package devices

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"

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
		for {
			sentence, err := dev.Read()
			if err != nil {
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

// Read reads a NMEA sentence terminated by LF and trims CRLF.
func (dev *Device) Read() (string, error) {
	if dev.reader == nil {
		return "", errors.New("device is not connected")
	}

	line, err := dev.reader.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) && len(line) > 0 {
			return strings.TrimRight(line, "\r\n"), nil
		}
		return "", err
	}

	return strings.TrimRight(line, "\r\n"), nil
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
	if err != nil {
		return nil, err
	}
	if len(ports) == 0 {
		return nil, nil
	}
	return ports, nil
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
