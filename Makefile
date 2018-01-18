GEO-ACQ = geo-acq
GEO-ACQ-PATH = cmd/${GEO-ACQ}
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

all:
	cd $(GEO-ACQ-PATH);go build -$(LDFLAGS)

allos: release copy

release: $(PLATFORMS)

$(PLATFORMS):
ifeq ($(os),linux)	
	cd $(GEO-ACQ-PATH);GOOS=$(os) GOARCH=$(arch) go build -o $(GEO-ACQ)-'$(os)-$(arch)'.exe -$(LDFLAGS)	
else
	cd $(GEO-ACQ-PATH);GOOS=$(os) GOARCH=$(arch) go build -o $(GEO-ACQ)-'$(os)-$(arch)' -$(LDFLAGS)
endif

run:
	$(GEO-ACQ-PATH)/$(GEO-ACQ)

copy: 
	-cp $(GEO-ACQ-PATH)/$(GEO-ACQ)-* $(DEST)
	-cp *.toml $(DEST)
	
clean:
	-rm -f $(GEO-ACQ-PATH)/$(GEO-ACQ)-*
	-rm -f $(GEO-ACQ-PATH)/geo-acq.exe
	
.PHONY: release $(PLATFORMS) clean