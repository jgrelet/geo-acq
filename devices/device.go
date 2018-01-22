package devices

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"github.com/jgrelet/geo-acq/config"
	"github.com/jgrelet/geo-acq/util"
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
	openSP   func(port string) (io.ReadWriteCloser, error)
	openEth  func(port string) (io.ReadWriteCloser, error)
	simul    bool
	logger   *log.Logger
	verbose  bool
	Data     chan string
}

// New creates a new Device object and connects to the specified serial port.
func New(name string, args ...interface{}) *Device {
	var cfg config.Config
	// Create new device
	dev := &Device{
		name: name,
		port: "",
		conn: nil,
		openSP: func(port string) (io.ReadWriteCloser, error) {
			p, err := serial.Open(port, &serial.Mode{
				BaudRate: cfg.Serials[name].Baud,
				DataBits: cfg.Serials[name].Databit,
				Parity:   serial.NoParity,
				StopBits: serial.OneStopBit,
			})
			if err != nil {
				err = fmt.Errorf("Can't open serial port %s -> %s", port, err)
			}
			return p, err
		},
		openEth: func(port string) (io.ReadWriteCloser, error) {
			saddr, err := net.ResolveUDPAddr("udp", cfg.UDP[name].Port)
			if err != nil {
				err = fmt.Errorf("Can't open ethernet port %s -> %s", port, err)
			}

			/* Now listen at selected port */
			scon, err := net.ListenUDP("udp", saddr)
			if err != nil {
				err = fmt.Errorf("Can't open ethernet port %s -> %s", port, err)
			}
			defer scon.Close()
			return scon, err
		},
		logger:  log.New(os.Stdout, fmt.Sprintf("[%s] ", name), log.Ltime),
		verbose: true,
		Data:    make(chan string),
	}

	// Parse variadic args
	for _, arg := range args {
		switch arg.(type) {
		case config.Config:
			cfg = arg.(config.Config)
			// dev.name is gps, echo-sounder or radar
			dev.typePort = cfg.Devices[dev.name].Device
			switch dev.typePort {
			case "serial":
				dev.port = cfg.Serials[dev.name].Port
			case "udp":
				dev.port = cfg.UDP[dev.name].Port
			}
		case io.ReadWriteCloser:
			dev.conn = arg.(io.ReadWriteCloser)
			//case chan:
			//	dev.data = make(chan interface{})
		}
	}
	fmt.Println("Port:", dev.port)
	return dev
}

// Connect starts a connection to the firmata board.
func (dev *Device) Connect() error {

	// serial or ethernet
	switch dev.typePort {
	case "serial":
		if dev.conn == nil {
			// enumerate avalaible serial port
			SerialGetInfo()
			// Try to connect to serial port
			sp, err := dev.openSP(dev.Port())
			if err != nil {
				return err
			}
			// Serial connection was successful
			dev.conn = sp
		}
	case "udp":
		fmt.Println("ethernet....")
		eth, err := dev.openEth(dev.Port())
		if err != nil {
			return err
		}
		// Serial connection was successful
		dev.conn = eth
	}
	go func() {
		for {
			sentence, err := dev.Read()
			if err != nil {
				close(dev.Data)
				break
			}
			dev.Data <- sentence
		}
	}()
	return nil
}

// Disconnect closes the io connection to the firmata board
func (dev *Device) Disconnect() (err error) {

	return nil
}

// Port returns the  FirmataAdaptors port
func (dev *Device) Port() string { return dev.port }

// Name returns the  FirmataAdaptors name
func (dev *Device) Name() string { return dev.name }

// Write over serial port
func (dev *Device) Write(sentence string) (err error) {
	_, err = dev.conn.Write([]byte(sentence))
	return err
}

// Read over serial port
func (dev *Device) Read() (response string, err error) {
	var n int
	buff := make([]byte, 80)
	var stringbuff string
	var state int
	var endOfSentence bool
	defer func() {
		if e := recover(); e != nil {
			fmt.Println(fmt.Errorf("serial port %s is disconnected -> %s Please check device connection",
				dev.Port(), err))
		}
	}()
	for {
		n, err = dev.conn.Read(buff)
		if err != nil {
			log.Panic("Log panic here !")
			break
		}
		if n == 0 {
			fmt.Println(err)
			log.Fatal("\nEOF")
			return "", err
		}
		//fmt.Printf("%v\n", string(buff[:n]))
		switch string(buff[:n]) {
		case util.CR:
			state = 1
			//fmt.Println("found carriage return")
		case util.LF:
			endOfSentence = true
			if state == 1 {
				//fmt.Println("found newline")
				state = 2
			} else {
				state = 0
				log.Printf("Invalid end of line, CR is missing: \n%s", stringbuff)
				stringbuff = ""
			}
		default:
			if state == 1 {
				state = 0
				log.Printf("Invalid end of line, LF is missing: \n%s", stringbuff)
				stringbuff = string(buff[:n])
			} else {
				stringbuff = fmt.Sprintf("%s%s", stringbuff, string(buff[:n]))
			}
		}
		if endOfSentence && state == 2 {
			//fmt.Printf("result: %s\n", stringbuff)
			break
		}
	}
	return stringbuff, nil
}

// SerialGetInfo retrieve the port list
func SerialGetInfo() {
	ports, err := serial.GetPortsList()
	if err != nil {
		log.Fatal(err)
	}
	if len(ports) == 0 {
		log.Fatal("No serial ports found!")
	}

	// Print the list of detected ports
	for _, port := range ports {
		fmt.Printf("Found port: %v\n", port)
	}
}
