BINARY     := s2t
BIN_DIR    := bin
INSTALL_DIR := $(HOME)/.local/bin

.PHONY: all build vet test install clean

all: build

vet:
	go vet ./...

test:
	go test ./...

build: vet test
	go build -o $(BIN_DIR)/$(BINARY) .

install: build
	mkdir -p $(INSTALL_DIR)
	cp $(BIN_DIR)/$(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo "Installed to $(INSTALL_DIR)/$(BINARY)"

clean:
	rm -rf $(BIN_DIR)
