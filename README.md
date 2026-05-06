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
- `storage/`: SQLite acquisition persistence for raw and processed data
- `simul/`: simulation logic
- `examples/`: ready-to-use sample configurations

## Requirements

- Go 1.19 or newer
- GNU Make if you want to use the `Makefile`
- [Task](https://taskfile.dev/) if you want to use `Taskfile.yml`
- Node.js and `npm` for the Wails GUI frontend build
- On Linux, `pkg-config`, GTK 3, and WebKitGTK development packages for the Wails GUI build

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
task install-wails
task check-gui-prereqs
task build-gui-wails
```

### Make

```bash
make help
make test
make build
make build-sim
make build-export
make install-wails
make check-gui-prereqs
make build-gui-wails
make cross-build
```

Build outputs are written to `bin/` and release artifacts to `dist/`.

## Developer notes

- `docs/git-worktree-memo.md`: short memo for the current `git worktree` workflow used in this project

## Wails GUI prototype

A first desktop GUI prototype is available with Wails at the repository root.
It is intended for evaluation, not as a final production UI yet.

What it currently provides:

- load and display the active TOML configuration
- choose a `.toml` file from a native file dialog
- edit the full TOML file inside the GUI and save it back to disk
- tabbed views for configuration, device panels, raw terminal frames, and available inputs
- separate device panels for configured devices
- live raw-frame terminal view with source filtering
- mixed acquisition mode using configured live and simulated devices

Build the desktop binary:

```bash
task install-wails
task build-gui-wails
```

or:

```bash
make install-wails
make build-gui-wails
```

Notes:

- `task test` / `make test` now work on a fresh clone even if the Wails frontend has not been built yet.
- the Wails CLI is installed automatically by `build-gui-wails` and `run-gui-wails` if it is missing from `$HOME/go/bin`
- `npm` must be available in `PATH` because Wails installs frontend dependencies and builds the Vite bundle during GUI build/dev
- on Ubuntu/Debian, a simple starting point is `sudo apt install nodejs npm`
- on Ubuntu 24.04/Noble, Wails needs `sudo apt install pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev`
- on Ubuntu 22.04/Jammy, Wails needs `sudo apt install pkg-config libgtk-3-dev libwebkit2gtk-4.0-dev`
- on Ubuntu 24.04/Noble, Wails builds also need the `webkit2_41` build tag; the `task` and `make` targets in this repo now add it automatically when `pkg-config` detects WebKitGTK 4.1

The generated executable is written to `build/bin/geo-acq-gui.exe` on Windows.

### Current Wails GUI behaviour

The current Wails desktop prototype is organised as follows:

- a startup banner showing application readiness
- a control bar with:
  - config path field
  - `Choose file`
  - `Load config`
  - `Edit config`
  - `Refresh ports`
  - `Start`
  - `Stop`
- a central tabbed area with:
  - `Configuration`
  - `Device panels`
  - `Terminal raw frames`
  - `Available inputs`

The default operational tab is `Device panels`.
When `Start` is used, the interface stays focused on the device view.

### Config workflow in the Wails GUI

Current configuration handling works like this:

1. The path field contains the current TOML path.
2. `Choose file` opens a native file dialog restricted to `*.toml`.
3. `Load config` reads the selected file and refreshes the GUI state.
4. `Edit config` opens a full-screen TOML editor overlay.
5. `Validate config` writes the edited TOML back to the current file and reloads it.

The GUI does not yet expose structured per-field forms for mission, devices, serial ports, or UDP settings.
At the moment, configuration editing is done on the raw TOML file as text.

### Device panels in the Wails GUI

The `Device panels` tab currently displays one card per configured device.
Each panel shows:

- device name
- transport and configured port
- current status
- number of frames seen
- configured sentence filter
- decoded values rendered as operator-friendly name/value pairs when available

The raw frame is no longer repeated in the device card because it is already visible in the terminal tab.

### Terminal view in the Wails GUI

The `Terminal raw frames` tab is intended as the diagnostic console.

It currently shows:

- the latest raw terminal lines produced from incoming frames
- timestamps
- device name
- transport
- port
- sentence type
- raw NMEA payload

A source selector filters the displayed frames.
The current filter values are built from the known runtime sources, for example:

- `serial:COM3`
- `serial:COM16`
- `udp:10183`
- `udp:10184`

### Available inputs tab

The `Available inputs` tab currently lists the detected serial ports exposed by the runtime.
This is mainly a quick operator check to confirm that expected serial inputs are visible before starting acquisition.

### Start mode in the Wails GUI

When `Start` is used:

1. the current configuration is validated and loaded if needed
2. each device mode is read from `[devices].mode`
3. devices in `ready` mode are opened using the configured transport
4. devices in `simulate` mode are started from the built-in simulator logic
5. devices in `disabled` mode are ignored
6. frames are read from active devices
7. each frame is timestamped
8. each frame is decoded when possible
9. the configured raw and/or processed backups are updated in SQLite
10. the GUI is updated through Wails runtime events

The session badge can therefore show a mixed session with both live and simulated devices.

### Device mode values

Each device now supports:

- `mode = "ready"`: use the configured serial or UDP transport
- `mode = "simulate"`: use the built-in simulator for this device only
- `mode = "disabled"`: do not start this device

For example:

```toml
[devices.gps]
type = "nmea"
mode = "ready"
device = "serial"
sentence = "GGA,RMC"

[devices.echosounder]
type = "nmea"
mode = "simulate"
device = "serial"
sentence = "DBT"
```

### Terminal view behaviour

The terminal tab stays active for both live and simulated devices.

It currently shows:

- the latest raw terminal lines produced from incoming frames
- timestamps
- device name
- transport
- port
- sentence type
- raw NMEA payload

Simulated devices appear with `simulate` as transport, and they can still be filtered from the source selector using the usual `transport:port` format.

### Current implementation limits

At the moment, the Wails GUI is still a prototype.
Some known limits of the current implementation are:

- decoded payloads are still shown as pretty-printed JSON, not instrument-specific widgets
- configuration editing is raw TOML only
- the source filter is simple and based on transport/port labels
- there is no dedicated form validation UI beyond TOML parsing errors
- there is no long-term session history browser inside the GUI yet

This section documents the current behaviour intentionally, so it can be updated later as the GUI evolves.

## Configuration

The runtime selects a default configuration file from the OS:

- `windows.toml` on Windows
- `linux.toml` on Linux and macOS

## Linux serial troubleshooting

If a USB serial adapter such as a PL2303 is detected by `lsusb` but the GUI does
not show decoded frames yet, first validate the raw NMEA stream outside the
application.

Check that the device node exists and that your user can access it:

```bash
ls -l /dev/ttyUSB0
groups
```

Read raw frames directly from the port with the same settings as the default
Linux configuration:

```bash
stty -F /dev/ttyUSB0 4800 cs8 -cstopb -parenb
timeout 10 cat /dev/ttyUSB0
```

Expected output is a stream of NMEA sentences such as:

```text
$GPGGA,...
$GPGSA,...
$GPRMC,...
```

If these commands work but the GUI still shows `RAW`, compare the serial
settings in [linux.toml](/home/jgrelet/git/go/geo-acq/linux.toml:28) with the
actual device output and inspect the `Terminal raw frames` tab for malformed or
partial first sentences.

You can always override it with `-config`:

```bash
./bin/geo-acq.exe -config windows.toml
./bin/geo-acq -config linux.toml
```

Each device is configured with:

- a logical section in `[devices]`
- a mode: `ready`, `simulate`, or `disabled`
- a transport type: `serial` or `udp`
- a matching section in `[serials]` or `[udp]`

For UDP:

- `host = ""` means listener mode
- `host = "127.0.0.1"` or another IP means sender mode

Mission metadata is configured in the `[mission]` section:

- `name`: mission or campaign name
- `pi`: principal investigator
- `organization`: lab, institute, or operator organization

Acquisition persistence is configured in `[backup]`.

Example:

```toml
[backup]
raw       = true
processed = true
```

With this configuration:

- `raw = true` enables the append-only raw acquisition database `<mission>-raw.sqlite`
- `processed = true` enables the decoded acquisition database `<mission>-data.sqlite`

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
2. The acquisition runtime starts every active device from `[devices]`.
3. A `ready` device opens either a serial port or a UDP socket.
4. A `simulate` device uses the built-in simulator for that instrument.
5. Incoming bytes are read line by line until `LF`.
6. Trailing `CRLF` is removed.
7. Each complete NMEA sentence is pushed to the device `Data` channel.
8. `geo-acq` timestamps the sentence at reception time.
9. The sentence can be stored in a raw SQLite database, a processed SQLite database, or both.
10. If `global.echo = true`, the sentence is also printed to stdout.

In practice, the data path is:

- transport setup in `devices/`
- sentence framing in `devices.Device.Read()`
- dispatch through `devices.Device.Data`
- persistence in `storage/`
- optional display in `cmd/geo-acq`

### What is processed today

At the moment, `geo-acq`:

- reads raw NMEA sentences from all active configured devices
- keeps sentence boundaries intact
- timestamps frames on reception
- decodes supported NMEA sentences during acquisition
- can store raw frames and/or processed records in SQLite
- optionally prints received sentences to standard output

There is not yet a higher-level processing stage that merges devices together or computes scientific products directly during acquisition.

### What is stored today

When `backup.raw = true`, the runtime persists acquisition data in a mission-derived SQLite database named `<mission>-raw.sqlite`.

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

When `backup.processed = true`, the runtime also persists decoded acquisition data in a mission-derived SQLite database named `<mission>-data.sqlite`.

The current processed schema is organized by sentence type:

- `gga_records`
- `rmc_records`
- `vtg_records`
- `dbt_records`

Each processed record keeps:

- mission reference
- acquisition session reference
- UTC reception timestamp
- device name
- transport type
- decoded fields in typed SQLite columns

The `log` field is still present in the configuration, but the current runtime mainly writes operational messages to stdout rather than managing a dedicated log file.

### Current implication

If you run `geo-acq` today:

- incoming NMEA sentences can be stored in one or two SQLite databases depending on `[backup]`
- incoming NMEA sentences are visible in the terminal only if `global.echo = true`
- transport errors stop the process
- one acquisition session is created for each program start
- mission metadata is attached to the stored data

This keeps the acquisition layer focused on preserving observations while making later scientific extraction easier from the processed store.

## Export mode

The export binary reads either a raw or a processed SQLite database and writes a plain-text TSV file.

Two alignment strategies are currently supported:

- `slowest_device`: one output row per frame of the least frequent device
- `fixed_interval`: one output row per constant time step

At each output timestamp, the exporter keeps the latest known values for each device at or before that timestamp.

If `export.database` is omitted:

- `geo-export` prefers `<mission>-data.sqlite` when `backup.processed = true` and the file exists
- otherwise it falls back to `<mission>-raw.sqlite`

If `export.database` is provided explicitly, `geo-export` detects automatically whether it is a raw or processed acquisition database.

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
- one column per device when exporting raw data
- structured columns such as `gps.rmc.latitude` or `echosounder.dbt.depth_meters` when exporting processed data

## Acquisition mode

The acquisition binary reads incoming NMEA sentences from every active device and stores them in SQLite.

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
- raw storage in `udp-listener-raw.sqlite`
- processed storage in `udp-listener-data.sqlite` when `processed = true`

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
- SQLite is used for both raw and processed acquisition stores
