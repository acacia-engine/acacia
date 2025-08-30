# Docker Usage

This document details how to build and run the Acacia Engine using Docker.

## Building the Docker Image

You can build the Docker image using the provided `Dockerfile`. This file defines a multi-stage build process designed to create optimized and minimal final images.

### Multi-Stage Build Process

The `Dockerfile` is structured in two main stages:

1.  **Builder Stage**:
    *   Starts from a `golang:1.24-bookworm` base image, which provides the Go toolchain and a Debian "Bookworm" environment.
    *   Copies the entire project source code into the container.
    *   Downloads all necessary Go dependencies using `go mod download`.
    *   Installs diagnostic tools (`file` and `ldd`) to help analyze compiled binaries if needed.
    *   Compiles the main `acacia` application executable.
    *   **Plugin Compilation**:
        *   Sets `GOOS=linux` and `GOARCH=arm64` to ensure all plugins are cross-compiled for the correct target environment.
        *   Iterates through each gateway and module directory using a `find` command.
        *   Builds each plugin in its own isolated environment by changing into the plugin's directory before compiling. This ensures that modules with their own `go.mod` files are handled correctly and prevents build conflicts.
    *   Includes a diagnostic step that runs `file` and `ldd` on each compiled plugin (`.so` file) to verify its format and dynamic dependencies.

2.  **Final Stage**:
    *   Starts from a minimal `debian:stable-slim` base image.
    *   Copies the compiled `acacia` executable and the `build/plugins` directory from the builder stage.
    *   Copies necessary configuration files (`config.yaml`, `go.work`, `go.work.sum`).
    *   Sets the `ACACIA_ENVIRONMENT` environment variable based on the `APP_ENV` build argument.
    *   Defines the default command to run the application (`./acacia serve`).

### Build Commands

The build process is environment-aware, allowing you to create optimized images for different environments using the `APP_ENV` build argument.

**Build for Production:**

```bash
docker build --build-arg APP_ENV=production -t acacia-app:latest .
```

**Build for Development:**

```bash
docker build --build-arg APP_ENV=development -t acacia-app:dev .
```

If the `APP_ENV` build argument is not specified, it defaults to `development`.

## Running the Docker Container

To run the application, use the `docker run` command. The Docker image's default command is `./acacia serve`, which starts the Acacia server. You can pass configuration to the application using environment variables with the `ACACIA_` prefix. The `ACACIA_ENVIRONMENT` environment variable is automatically set based on the `APP_ENV` build argument used during image creation.

```bash
docker run -p 8080:8080 -e ACACIA_ENVIRONMENT=production acacia-app:latest
```

## Getting Started with Docker

### Prerequisites

*   Go 1.23 or higher
*   PostgreSQL (or another preferred database)
*   Docker

### Local Development with Docker Compose

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/your-username/acacia-engine-go.git
    cd acacia-engine-go
    ```

2.  **Set up configuration:**
    Copy the `.env.example` file to `.env` and configure your database connection and other settings.
    ```bash
    cp .env.example .env
    ```

3.  **Run with Docker Compose:**
    The provided `docker-compose.yml` will start the Acacia server and a PostgreSQL database.
    ```bash
    docker-compose up --build
    ```
    The Acacia server will now be running on `http://localhost:8080`.
