package devices

import (
	"fmt"
	"io"
	"log"
	"os"

	//"github.com/tarm/serial"
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
}

// Device represents instrument.
type Device struct {
	name    string
	port    string
	conn    io.ReadWriteCloser
	openSP  func(port string) (io.ReadWriteCloser, error)
	simul   bool
	logger  *log.Logger
	verbose bool
}

// New creates a new Device object and connects to the specified serial port.
func New(name string, args ...interface{}) *Device {
	var cfg config.Config
	// Create new Godudev client
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
		logger:  log.New(os.Stdout, fmt.Sprintf("[%s] ", name), log.Ltime),
		verbose: true,
	}

	// Parse variadic args
	for _, arg := range args {
		switch arg.(type) {
		case config.Config:
			cfg = arg.(config.Config)
			dev.port = cfg.Serials[dev.name].Port
		case io.ReadWriteCloser:
			dev.conn = arg.(io.ReadWriteCloser)
		case bool:
			//dev.conn = make(chan interface{})
		}
	}
	return dev
}

// Connect starts a connection to the firmata board.
func (dev *Device) Connect() error {
	if dev.conn == nil {
		// Try to connect to serial port
		sp, err := dev.openSP(dev.Port())
		if err != nil {
			return err
		}
		// Serial connection was successful
		dev.conn = sp
	}
	// Firmata connection
	//return dev.board.Connect(dev.conn)
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

// Read over serial port
func (dev *Device) Read() (response string, err error) {
	var n int
	buff := make([]byte, 80)
	var stringbuff string
	var state int
	var endOfSentence bool

	defer func() {
		if e := recover(); e != nil {
			fmt.Println(fmt.Errorf("serial port %T is disconnected -> %s, please check RS232 or USB connection",
				dev.Port(), err))
		}
	}()

	for {
		n, err = dev.conn.Read(buff)
		if err != nil {
			fmt.Println(err)
			log.Panic(err)
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
