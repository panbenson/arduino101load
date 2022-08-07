# arduino101load
multiplatform launcher for Arduino101 dfu-util flashing utility

## Compiling

* Download go package from [here](https://golang.org/dl/) or using your package manager
* `cd` into the root folder of this project
* execute
```bash
export GOPATH=$PWD
export GOBIN=.
# run with go < 1.18, set flag for compiling non-module go
export GO111MODULE=auto
go get -d
go build
# replace recompiled version in Arduino15
cp arduino101load /Users/$USER/Library/Arduino15/packages/Intel/tools/arduino101load/2.0.1/arduino101load
```
to produce a binary of `arduino101load` for your architecture.

To cross compile for different OS/architecture combinations, execute
```bash
GOOS=windows GOARCH=386 go build  #windows
GOOS=darwin GOARCH=amd64 go build #osx
GOOS=linux GOARCH=386 go build    #linux_x86
GOOS=linux GOARCH=amd64 go build  #linux_x86-64
```
