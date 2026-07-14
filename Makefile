BINARY     := s2t
BIN_DIR    := bin

UNAME_S := $(shell uname -s 2>/dev/null)

ifeq ($(OS),Windows_NT)
	BINARY_NAME := $(BINARY).exe
	INSTALL_DIR := $(USERPROFILE)/bin
else ifneq (,$(findstring MINGW,$(UNAME_S))$(findstring MSYS,$(UNAME_S))$(findstring CYGWIN,$(UNAME_S)))
	BINARY_NAME := $(BINARY).exe
	INSTALL_DIR := $(USERPROFILE)/bin
else
	BINARY_NAME := $(BINARY)
	INSTALL_DIR := $(HOME)/.local/bin
endif

.PHONY: all build vet test install clean

all: build

vet:
	go vet ./...

test:
	go test ./...

build: vet test
	go build -o $(BIN_DIR)/$(BINARY_NAME) .

install: build
	mkdir -p $(INSTALL_DIR)
	cp $(BIN_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Installed to $(INSTALL_DIR)/$(BINARY_NAME)"

clean:
	rm -rf $(BIN_DIR)
