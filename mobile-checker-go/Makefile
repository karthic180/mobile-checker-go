BINARY_CLI    = mobile-checker
BINARY_SERVER = mobile-server
CMD_CLI       = ./cmd/mobile
CMD_SERVER    = ./cmd/server

.PHONY: all build test clean run-setup run-check run-server

all: build

build:
	go build -o $(BINARY_CLI) $(CMD_CLI)
	go build -o $(BINARY_SERVER) $(CMD_SERVER)

test:
	go test ./... -v

run-setup:
	go run $(CMD_CLI) setup

run-check:
	go run $(CMD_CLI) check SW1A1AA

run-server:
	go run $(CMD_SERVER) --addr :5001

clean:
	rm -f $(BINARY_CLI) $(BINARY_SERVER)

install:
	go install $(CMD_CLI)
