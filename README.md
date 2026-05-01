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
- `cmd/export`: offline export binary
- `cmd/simul/gps`: GPS simulator binary
- `cmd/simul/echosounder`: echosounder simulator binary
- `config/`: configuration loading
- `devices/`: serial and UDP transport layer
- `storage/`: SQLite raw acquisition persistence
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
task build-export
```

### Make

```bash
make help
make test
make build
make build-sim
make build-export
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

Mission metadata is configured in the `[mission]` section:

- `name`: mission or campaign name
- `pi`: principal investigator
- `organization`: lab, institute, or operator organization

The acquisition database path is configured in `[acq].file`.

Offline export parameters are configured in the `[export]` section:

- `database`: SQLite source database
- `output`: text output file
- `mode`: `slowest_device` or `fixed_interval`
- `interval`: required for `fixed_interval`
- `mission`: optional mission filter
- `session_id`: optional session selector

## Data flow and storage

The current processing pipeline is intentionally simple:

1. A device is created from the TOML configuration.
2. The acquisition runtime opens every enabled device from `[devices]`.
3. Each device opens either a serial port or a UDP socket.
4. Incoming bytes are read line by line until `LF`.
5. Trailing `CRLF` is removed.
6. Each complete NMEA sentence is pushed to the device `Data` channel.
7. `geo-acq` timestamps the sentence at reception time.
8. The raw sentence is stored in SQLite with mission and session metadata.
9. If `global.echo = true`, the sentence is also printed to stdout.

In practice, the data path is:

- transport setup in `devices/`
- sentence framing in `devices.Device.Read()`
- dispatch through `devices.Device.Data`
- persistence in `storage/`
- optional display in `cmd/geo-acq`

### What is processed today

At the moment, `geo-acq`:

- reads raw NMEA sentences from all enabled configured devices
- keeps sentence boundaries intact
- timestamps frames on reception
- stores them in SQLite as append-only raw records
- optionally prints received sentences to standard output

There is not yet a higher-level processing stage that:

- parses incoming sentences in the acquisition binary
- enriches or merges GPS and echosounder data
- computes scientific products directly during acquisition

### What is stored today

The runtime now persists acquisition data in a SQLite database defined by `[acq].file`.

The storage model is append-only and centered on raw frames:

- `missions`: mission metadata from the TOML file
- `acquisition_sessions`: one row per `geo-acq` run
- `raw_frames`: one row per received NMEA sentence

Each raw frame stores:

- mission reference
- acquisition session reference
- UTC reception timestamp
- device name
- transport type
- raw NMEA payload

The `log` field is still present in the configuration, but the current runtime mainly writes operational messages to stdout rather than managing a dedicated log file.

### Current implication

If you run `geo-acq` today:

- incoming NMEA sentences are stored in SQLite
- incoming NMEA sentences are visible in the terminal only if `global.echo = true`
- transport errors stop the process
- one acquisition session is created for each program start
- mission metadata is attached to the stored data

This keeps the acquisition layer focused on preserving raw observations, while later scientific extraction can happen in a separate application.

## Export mode

The export binary reads raw frames from SQLite and writes a plain-text TSV file.

Two alignment strategies are currently supported:

- `slowest_device`: one output row per frame of the least frequent device
- `fixed_interval`: one output row per constant time step

At each output timestamp, the exporter keeps the latest known raw payload for each device at or before that timestamp.

### Build the exporter

```bash
task build-export
```

On GNU Make:

```bash
make build-export
```

### Export on the slowest device rhythm

```bash
./bin/geo-export.exe -config examples/export-slowest.toml
```

On Linux/macOS:

```bash
./bin/geo-export -config examples/export-slowest.toml
```

### Export on a fixed interval

```bash
./bin/geo-export.exe -config examples/export-fixed.toml
```

On Linux/macOS:

```bash
./bin/geo-export -config examples/export-fixed.toml
```

The generated TSV file contains:

- a metadata preamble with mission and session information
- one `timestamp_utc` column
- one raw payload column per device

## Acquisition mode

The acquisition binary reads incoming NMEA sentences from every enabled device and stores them in SQLite.

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
- raw storage in `geo-acq-udp-listener.sqlite`

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
- `examples/export-slowest.toml`: export using the slowest device as reference
- `examples/export-fixed.toml`: export using a constant interval
- `docs/udp-test.md`: short UDP test memo

## Notes

- Local Go caches are redirected to `.gocache/` and `.gomodcache/` by the `Taskfile`
- The serial reader now consumes complete NMEA lines terminated by `CRLF`
- UDP is implemented for both acquisition and simulation workflows
- SQLite is used as the raw acquisition store
