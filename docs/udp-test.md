# UDP test

## Receiver

Start the acquisition app in UDP listener mode:

```bash
task build
./bin/geo-acq.exe -config examples/udp-listener.toml
```

On Linux/macOS:

```bash
task build
./bin/geo-acq -config examples/udp-listener.toml
```

## Sender

In a second terminal, start the GPS simulator in UDP sender mode:

```bash
task build-sim
./bin/simul-gps.exe -config examples/udp-sender.toml
```

On Linux/macOS:

```bash
task build-sim
./bin/simul-gps -config examples/udp-sender.toml
```

In a third terminal, start the echosounder simulator:

```bash
task build-sim-sounder
./bin/simul-echosounder.exe -config examples/udp-sender.toml
```

On Linux/macOS:

```bash
task build-sim-sounder
./bin/simul-echosounder -config examples/udp-sender.toml
```

## Notes

- `examples/udp-listener.toml` listens on `127.0.0.1:10183`
- `examples/udp-sender.toml` sends GPS NMEA frames to `127.0.0.1:10183`
- The echosounder simulator sends DBT frames to `127.0.0.1:10184`
- To test on another machine, replace `127.0.0.1` in `examples/udp-sender.toml` with the receiver IP
