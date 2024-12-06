.PHONY: all build run clean

all: build

build:
	go build -o bin/myshell cmd/myshell/*.go

run: build
	./bin/myshell

clean:
	rm -f bin/myshell