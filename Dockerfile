# Stage 1: Build the Go application
ARG APP_ENV=development
FROM golang:1.25-bookworm AS builder

WORKDIR /app

# Copy go.mod and go.sum first to leverage caching for go mod download
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the entire project directory into the builder stage
COPY . .

# Install 'file' and 'ldd' (part of libc-bin) for diagnostics
RUN apt-get update && apt-get install -y file libc-bin && rm -rf /var/lib/apt/lists/*

# Build the main application
RUN CGO_ENABLED=1 go build -ldflags '-s -w' -o acacia ./cmd/acacia

# Build plugins
# Create the plugins directory
RUN mkdir -p build/plugins

# Explicitly set GOOS and GOARCH for plugin compilation to ensure Linux ARM64 binaries
ENV GOOS=linux
ENV GOARCH=arm64

# Dynamically build all gateway plugins in their own environment using find
RUN for dir in $(find ./gateways -mindepth 1 -maxdepth 1 -type d); do \
    if [ -n "$(find "$dir" -maxdepth 1 -name '*.go' ! -name '*_test.go')" ]; then \
        echo "Building gateway plugin in $dir..."; \
        (cd "$dir" && CGO_ENABLED=1 go build -buildmode=plugin -o ../../build/plugins/$(basename "$dir").so .); \
    fi; \
done

# Dynamically build all module plugins in their own environment using find
RUN for dir in $(find ./modules -mindepth 1 -maxdepth 1 -type d); do \
    if [ -n "$(find "$dir" -maxdepth 1 -name '*.go')" ]; then \
        echo "Building module plugin in $dir..."; \
        (cd "$dir" && CGO_ENABLED=1 go build -buildmode=plugin -o ../../build/plugins/$(basename "$dir").so .); \
    fi; \
done

# Add diagnostic commands for built plugins
RUN echo "--- Plugin Diagnostics ---" && \
    for plugin_path in build/plugins/*.so; do \
        echo "Inspecting $plugin_path:"; \
        file "$plugin_path"; \
        ldd "$plugin_path"; \
        echo ""; \
    done && \
    echo "--- End Plugin Diagnostics ---"

# Stage 2: Create the final, minimal image
FROM debian:stable-slim

ARG APP_ENV

WORKDIR /root/

# Copy the built executable from the builder stage
COPY --from=builder /app/acacia .

# Copy the plugins directory (now built within the builder stage)
COPY --from=builder /app/build/plugins ./build/plugins

# Copy go.work and go.work.sum for runtime if needed (though typically not for final image)
# This is primarily for ensuring the workspace context is available if any runtime operations
# were to depend on it, which is unlikely for a compiled binary.
# However, including it here for completeness if the application were to inspect its own workspace.
COPY --from=builder /app/go.work /app/go.work.sum ./

# Copy the configuration file
COPY config.yaml .

# Set the environment variable for the application
ENV ACACIA_ENVIRONMENT=${APP_ENV}

# Command to run the application
CMD ["./acacia", "serve"]
