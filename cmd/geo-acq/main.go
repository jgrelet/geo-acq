package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/jgrelet/geo-acq/config"
	"github.com/jgrelet/geo-acq/decoder"
	"github.com/jgrelet/geo-acq/devices"
	"github.com/jgrelet/geo-acq/storage"
)

type deviceMessage struct {
	receivedAt   time.Time
	deviceName   string
	transport    string
	port         string
	payload      string
	sentenceType string
	decodedJSON  string
}

// main entry
func main() {
	configPath := flag.String("config", config.DefaultFile(), "configuration TOML file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	store, err := storage.OpenSQLite(cfg.Acq.File, cfg.Mission, *configPath)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	deviceNames := enabledDeviceNames(cfg)
	if len(deviceNames) == 0 {
		log.Fatal("no enabled devices found in configuration")
	}

	messageCh := make(chan deviceMessage)
	managedDevices := make([]*devices.Device, 0, len(deviceNames))

	for _, deviceName := range deviceNames {
		dev := devices.New(deviceName, cfg)
		if err := dev.Connect(); err != nil {
			log.Fatalf("connect %s: %v", deviceName, err)
		}
		log.Printf("connected %s on %s %s", deviceName, cfg.Devices[deviceName].Device, dev.Port())
		managedDevices = append(managedDevices, dev)

		go func(name string, transport string, d *devices.Device) {
			for msg := range d.Data {
				decoded, err := decoder.DecodeNMEA(msg)
				if err != nil && cfg.Global.Debug {
					log.Printf("decode %s frame: %v", name, err)
				}

				sentenceType := ""
				decodedJSON := ""
				if err == nil {
					sentenceType = decoded.SentenceType
					decodedJSON = decoded.JSON
				}

				messageCh <- deviceMessage{
					receivedAt:   time.Now().UTC(),
					deviceName:   name,
					transport:    transport,
					port:         d.Port(),
					payload:      msg,
					sentenceType: sentenceType,
					decodedJSON:  decodedJSON,
				}
			}
		}(deviceName, cfg.Devices[deviceName].Device, dev)
	}
	defer func() {
		for _, dev := range managedDevices {
			_ = dev.Disconnect()
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	for {
		select {
		case msg := <-messageCh:
			if cfg.Global.Echo {
				fmt.Println(formatTerminalFrame(msg))
			}
			if err := store.SaveRawFrame(storage.RawFrame{
				ReceivedAt:   msg.receivedAt,
				DeviceName:   msg.deviceName,
				Transport:    msg.transport,
				Payload:      msg.payload,
				SentenceType: msg.sentenceType,
				DecodedJSON:  msg.decodedJSON,
			}); err != nil {
				log.Fatal(err)
			}
		case sig := <-sigCh:
			fmt.Printf("Exit on signal %s...\n", sig)
			return
		}
	}
}

func formatTerminalFrame(msg deviceMessage) string {
	sentenceType := msg.sentenceType
	if sentenceType == "" {
		sentenceType = "RAW"
	}

	return fmt.Sprintf(
		"%s | %-12s | %-6s | %-8s | %-5s | %s",
		msg.receivedAt.UTC().Format(time.RFC3339),
		msg.deviceName,
		msg.transport,
		msg.port,
		sentenceType,
		msg.payload,
	)
}

func enabledDeviceNames(cfg config.Config) []string {
	names := make([]string, 0, len(cfg.Devices))
	for name, device := range cfg.Devices {
		if device.Use {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}
