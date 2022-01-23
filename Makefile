all: build

build:
	go build 
	strip svc-monitor
