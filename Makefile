.PHONY: build build-windows run test clean install

BIN_NAME := wordle_solver
BIN_NAME_WIN := wordle_solver.exe

# Detect OS and set binary name
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
	BIN := $(BIN_NAME)
endif
ifeq ($(UNAME_S),Linux)
	BIN := $(BIN_NAME)
endif
ifeq ($(OS),Windows_NT)
	BIN := $(BIN_NAME_WIN)
endif

ifndef BIN
	BIN := $(BIN_NAME)
endif

# Default target
build:
	go build -o $(BIN) .

# Windows build with GUI flag to hide console window
build-windows:
	go build -ldflags "-H=windowsgui" -o $(BIN_NAME_WIN) .

# Run the application
run: build
	./$(BIN)

# Run tests
test:
	go test ./...

# Install via go install
install:
	go install github.com/Kyza/wordle_solver@latest

# Clean build artifacts
clean:
	rm -f $(BIN_NAME) $(BIN_NAME_WIN)
	go clean
