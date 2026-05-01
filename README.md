# geo-acq

`geo-acq` is a Go application for acquiring NMEA data from marine instruments such as:

- GPS
- Echosounder
- Radar

The project currently supports:

- acquisition from `serial` or `udp` transports
- GPS simulation
- echosounder simulation
- local build/test workflows with `make` and `task`

## Repository layout

- `cmd/geo-acq`: main acquisition binary
- `cmd/simul/gps`: GPS simulator binary
- `cmd/simul/echosounder`: echosounder simulator binary
- `config/`: configuration loading
- `devices/`: serial and UDP transport layer
- `simul/`: simulation logic
- `examples/`: ready-to-use sample configurations

## Requirements

- Go 1.19 or newer
- GNU Make if you want to use the `Makefile`
- [Task](https://taskfile.dev/) if you want to use `Taskfile.yml`

On Windows, Git Bash works well with the current `Makefile` and `Taskfile`.

## Build and test

### Task

```bash
task help
task test
task build
task build-sim
task build-sim-sounder
```

### Make

```bash
make help
make test
make build
make build-sim
make cross-build
```

Build outputs are written to `bin/` and release artifacts to `dist/`.

## Configuration

The runtime selects a default configuration file from the OS:

- `windows.toml` on Windows
- `linux.toml` on Linux and macOS

You can always override it with `-config`:

```bash
./bin/geo-acq.exe -config windows.toml
./bin/geo-acq -config linux.toml
```

Each device is configured with:

- a logical section in `[devices]`
- a transport type: `serial` or `udp`
- a matching section in `[serials]` or `[udp]`

For UDP:

- `host = ""` means listener mode
- `host = "127.0.0.1"` or another IP means sender mode

## Acquisition mode

The acquisition binary reads incoming NMEA sentences and prints them to stdout.

### Serial acquisition

On Windows:

```bash
task build
./bin/geo-acq.exe -config windows.toml
```

On Linux:

```bash
task build
./bin/geo-acq -config linux.toml
```

### UDP acquisition

Use the sample listener configuration:

```bash
task build
./bin/geo-acq.exe -config examples/udp-listener.toml
```

On Linux/macOS:

```bash
task build
./bin/geo-acq -config examples/udp-listener.toml
```

The listener example enables:

- GPS on UDP port `10183`
- echosounder on UDP port `10184`

## Simulation mode

The simulators generate NMEA sentences and send them to the configured transport.

### GPS simulation

```bash
task build-sim
./bin/simul-gps.exe -config examples/udp-sender.toml
```

On Linux/macOS:

```bash
task build-sim
./bin/simul-gps -config examples/udp-sender.toml
```

The GPS simulator currently emits:

- `GPGGA`
- `GPVTG`

### Echosounder simulation

```bash
task build-sim-sounder
./bin/simul-echosounder.exe -config examples/udp-sender.toml
```

On Linux/macOS:

```bash
task build-sim-sounder
./bin/simul-echosounder -config examples/udp-sender.toml
```

Optional flags:

- `-interval 600ms`: emission interval
- `-depth 12.0`: initial depth in meters

The echosounder simulator emits `GPDBT`.

## End-to-end UDP example

Terminal 1, start the acquisition listener:

```bash
task build
./bin/geo-acq.exe -config examples/udp-listener.toml
```

Terminal 2, start the GPS simulator:

```bash
task build-sim
./bin/simul-gps.exe -config examples/udp-sender.toml
```

Terminal 3, start the echosounder simulator:

```bash
task build-sim-sounder
./bin/simul-echosounder.exe -config examples/udp-sender.toml
```

For a multi-machine test, replace `127.0.0.1` in `examples/udp-sender.toml` with the IP address of the receiver host.

## Example files

- `examples/udp-listener.toml`: UDP receiver config for `geo-acq`
- `examples/udp-sender.toml`: UDP sender config for simulators
- `docs/udp-test.md`: short UDP test memo

## Notes

- Local Go caches are redirected to `.gocache/` and `.gomodcache/` by the `Taskfile`
- The serial reader now consumes complete NMEA lines terminated by `CRLF`
- UDP is implemented for both acquisition and simulation workflows
