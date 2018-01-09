package main

import (
	"fmt"

	"bitbucket.org/jgrelet/go/go-testings/channels/acq-dev/devices"
	_ "bitbucket.org/jgrelet/go/go-testings/channels/acq-dev/simul"
)

// main entry
func main() {

	// simul.GpsChan = make(chan interface{})
	// simul.EchoSounderChan = make(chan interface{})
	devices.SerialGetInfo()
	// ttyytt
	gps := devices.New("myGPS", "COM9")
	err := gps.Connect()
	defer gps.Disconnect()
	if err != nil {
		fmt.Println(err)
	}
	for {
		sentence, err := gps.Read()
		if err != nil {
			break
		}
		fmt.Println(sentence)
	}
	/*
		// read GPS on serial port or simulation
		go simul.Gps()

		// main loop
		for {
			data := <-simul.GpsChan
			switch v := data.(type) {
			case string:
				fmt.Println(v)
			case []byte:
				fmt.Println(v)
			}
		}
	*/
}
