export GO32="C:/tools/go1.5.1.windows-386"
export GO64="C:/tools/go"

gotour32.exe: main.go
	GOPATH="`pwd`/go32" GOROOT="$(GO32)" PATH="$(GO32)/bin:$$PATH" $(GO32)/bin/go get -t; true
	GOPATH="`pwd`/go32" GOROOT="$(GO32)" PATH="$(GO32)/bin:$$PATH" $(GO32)/bin/go build -ldflags '-H windowsgui' -o gotour32.exe

gotour64.exe: main.go
	GOPATH="`pwd`/go64" GOROOT="$(GO64)" PATH="$(GO64)/bin:$$PATH" $(GO64)/bin/go get -t; true
	GOPATH="`pwd`/go64" GOROOT="$(GO64)" PATH="$(GO64)/bin:$$PATH" $(GO64)/bin/go build -ldflags '-H windowsgui' -o gotour64.exe

pano32: gotour32.exe pano
pano64: gotour64.exe pano

panofiles := $(shell find "${PANO}" -type f)

pano: ${panofiles} gotour64.exe
	./gotour64.exe "${PANO}" pano

