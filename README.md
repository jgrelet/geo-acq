## Description

GEO-ACQ is an acquisition program

- GPS
- Echosounder
- OTT Radar

running on a Raspberry Pi

## Windows prerequisites  

Install MinGW with Msys. If you use Visual Studio Code, configure the terminal shell with msys. See: https://code.visualstudio.com/docs/editor/integrated-terminal#_configuration

Add the following line to your user/setting.json file:

  "terminal.integrated.shell.windows": "C:\\MinGW\\msys\\1.0\\bin\\bash.exe",

You must define Windows env variables :

GOBIN=%USERPROFILE%\go\bin 

and 

GOPATH=%USERPROFILE%\go

and add the C:\MinGW\msys\1.0\bin directory to your Windows path.

Test inside terminal:

> make --version
GNU Make 3.81


## Development

Clone the go-serial git repository directly into your src folder under src/go.bug.st/serial.v1 and checkout the branch v1.

```bash
cd $GOPATH
mkdir -p src/go.bug.st/
git clone https://github.com/bugst/go-serial.git -b v1 src/go.bug.st/serial.v1
go test go.bug.st/serial.v1
```

Install and use package:

A fork of github.com/pilebones/go-nmea:

- github.com/jgrelet/go-nmea
- github.com/creack/goselect
- github.com/pborman/getopt/v2
- github.com/BurntSushi/toml

## Compilation

Under development:

```bash
> go build
```

To build all plateform targets under production:

```bash
> make
```

To build specific  targets (linux/amd64, windows/amd64, linux/arm or darwin/amd64windows) under production:

```bash
> make linux/arm
...
>  ls -l *linux-arm*
-rw-r--r-- 1  nmea-proto-linux-arm
```

## Usage

```bash
> ./nmea-proto-linux-arm -h
Usage: c:\users\jgrelet\go\src\bitbucket.org\jgrelet\raspberry\go\dev\dev.exe [-dehlv] [-c nmea.toml] [-s value] [-t value] [parameters ...]
 -c, --config=nmea.toml
                    use specific configuration .toml file
 -d, --debug        Display debug info
 -e, --echo         Display processing in stdout
 -h, --help         Help
 -l, --log          Write log, defaut is true
 -s, --simul=value  Simulate: GPS, Echo-sounder or Radar
 -t, --trace=value  Display terminal for: GPS, Echo-sounder or Radar
 -v, --version      Show version, then exit.
```

Run in simulation mode, with error log to stdout and terminal mode:

```bash
> ./under production: -e -s gps,sounder -t gps,sounder -l -c linux.toml

[NMEA START]2017/12/06 09:19:58 Acquisition Begin
$GPDBT,355.45,f,111.36,M,59.24,F*0A
Depth in meters:  111.36
$GPGGA,091959.871,.999977,N,2300.000045,E,1,17,0.6,0051.6,M,0.0,M,,*6D
Time: 2017/12/06 09:19:59Z
Quality: GNSS fix
Latitude: 0° 0' 59.998620"
Longitude: 23° 0' 0.002700"
$GPDBT,355.45,f,113.04,M,59.24,F*09
Depth in meters:  113.04
$GPDBT,355.45,f,111.66,M,59.24,F*0F
Depth in meters:  111.66
$GPGGA,092000.871,.999954,N,2300.000091,E,1,17,0.6,0051.6,M,0.0,M,,*63
Time: 2017/12/06 09:20:00Z
Quality: GNSS fix
Latitude: 0° 0' 59.997240"
Longitude: 23° 0' 0.005460"
...
```

Option -d (debug) is an alias for options: -e -s gps,sounder -t gps,sounder

```bash
> nmea-proto-linux-arm -d -c linux.toml
```