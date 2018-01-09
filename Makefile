BINARY = nmea-proto
VERSION = 0.0.1

# user define
DEST = Z:/grelet/legos-gps-sondeur

# Setup the -ldflags option for go build here, interpolate the variable values
LDFLAGS = -ldflags "-X main.Version=${VERSION}  \
-X main.BuildTime=`TZ=UTC date -u '+%Y-%m-%dT%H:%M:%SZ'`"

PLATFORMS := linux/amd64 windows/amd64 linux/arm darwin/amd64

temp = $(subst /, ,$@)
os = $(word 1, $(temp))
arch = $(word 2, $(temp))

all: release copy

release: $(PLATFORMS)

$(PLATFORMS):
ifeq ($(os),linux)	
	GOOS=$(os) GOARCH=$(arch) go build -o $(BINARY)-'$(os)-$(arch)'.exe -$(LDFLAGS)	
else
	GOOS=$(os) GOARCH=$(arch) go build -o $(BINARY)-'$(os)-$(arch)' -$(LDFLAGS)
endif

copy: 
	-cp $(BINARY)-* $(DEST)
	-cp *.toml $(DEST)
	
clean:
	-rm -f ${BINARY}-*
	-rm -f dev.exe
	
.PHONY: release $(PLATFORMS) clean