# Makefile for the Acacia project

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOTOOL=$(GOCMD) tool
BINARY_NAME=acacia
PLUGIN_DIR=./build/plugins

.PHONY: all build build-plugins run clean

all: build

# Build the main application
build:
	$(GOBUILD) -o $(BINARY_NAME) ./cmd/acacia

# Build all plugins
build-plugins:
	@echo "Building plugins..."
	@mkdir -p $(PLUGIN_DIR)
	@for dir in $$(find ./gateways ./modules -maxdepth 1 -mindepth 1 -type d); do \
		output_path="$(PLUGIN_DIR)/$$(basename $$dir).so"; \
		echo "Building plugin: $(GOBUILD) -buildmode=plugin -o \"$$output_path\" \"$$dir\""; \
		$(GOBUILD) -buildmode=plugin -o "$$output_path" "$$dir"; \
	done

# Run the application
run: build-plugins build
	./$(BINARY_NAME) serve

# Run development tasks
dev: build-plugins build
	./$(BINARY_NAME) dev all

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -rf $(PLUGIN_DIR)
