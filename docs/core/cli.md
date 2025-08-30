# Acacia CLI Documentation

Welcome to the official documentation for the Acacia Engine's Command Line Interface (CLI). This guide provides everything you need to know to build, configure, and use the `acacia` CLI for managing your projects.

## Getting Started

Before you can use the Acacia CLI, you need to build it from the source code.

### Building the CLI

To build the `acacia` executable, run the following command from the project root:

```bash
go build ./cmd/acacia
```

This will create an `acacia` executable file in the root of your project directory.

### Running the Server

The primary function of the Acacia Engine is to run as a server. You can start the server with the `serve` command:

```bash
./acacia serve
```

The server's network configuration, such as ports, is managed through the configuration file. See the "Gateway Configuration" section for more details.

## Command Overview

The `acacia` CLI is organized into several commands, each with a specific purpose. Here is a quick overview:

| Command           | Description                                                 |
| ----------------- | ----------------------------------------------------------- |
| `acacia serve`    | Runs the Acacia server. This is the default command.        |
| `acacia dev`      | Provides development utilities for building and testing.    |
| `acacia module`   | Helps you create and manage modules.                        |
| `acacia registry` | Manages the registry for modules and gateways.              |

### Global Flags

These flags can be used with any command:

*   `--version`: Displays the current version of the Acacia CLI.

---

## Gateway Configuration

Gateways are configured through a `config.yaml` file in the root of the project. Each gateway has its own section in the `gateways` map.

For example, to configure a gateway named `http`, you would add the following to your `config.yaml`:

```yaml
gateways:
  http:
    addr: ":8080"
    tls:
      enabled: true
      certFile: "/path/to/cert.pem"
      keyFile: "/path/to/key.pem"
```

Each gateway's documentation should specify the available configuration options.

---

## Detailed Command Reference

This section provides a detailed look at each command, its subcommands, and available flags.

### `acacia serve`

Runs the Acacia server. If you run `./acacia` without any command, it will default to `serve`.

**Usage:**

```bash
./acacia serve
```

---

### `acacia dev`

Provides a set of utilities to help with development. If you run `acacia dev` without a subcommand, it defaults to `all`.

**Usage:**

```bash
./acacia dev [subcommand]
```

**Subcommands:**

*   `build`: Builds the `acacia` executable.
*   `tidy`: Runs `go mod tidy` to clean up project dependencies.
*   `test`: Runs all the tests in the project.
*   `all`: A convenient shortcut that runs `build`, `tidy`, and `test` in sequence.

---

### `acacia module`

This command helps you manage your project's modules.

**Usage:**

```bash
./acacia module [subcommand]
```

**Subcommands:**

#### `acacia module create <name>`

Creates a new module with a standard directory structure, including folders for `application`, `domain`, `infrastructure`, and `docs`, along with a `registry.json` file.

**Arguments:**

*   `<name>`: The name of the module to create (e.g., `my-new-module`).

**Flags:**

*   `--dir string`: The base directory where the module will be created (default: `modules`).
*   `--force`: Overwrites the target directory if it already exists.
*   `--version string`: The version to write into the module's `registry.json` (default: `0.1.0`).
*   `--description string`: An optional description for the module.
*   `--dependencies string`: A comma-separated list of module names that this module depends on.

**Example:**

```bash
./acacia module create my-api-module --version 1.0.0 --description "A module for handling API requests" --dependencies "auth,logger"
```

#### `acacia module list`

Lists all available modules and their current status (e.g., enabled, disabled, running).

**Usage:**

```bash
./acacia module list
```

#### `acacia module inspect <name>`

Displays detailed information about a specific module, including its version, description, and dependencies, as read from its `registry.json` file.

**Arguments:**

*   `<name>`: The name of the module to inspect.

**Usage:**

```bash
./acacia module inspect my-api-module
```

#### `acacia module enable <name>`

Enables a module, marking it to be loaded and started by the kernel. This change is persisted in the application's configuration.

**Arguments:**

*   `<name>`: The name of the module to enable.

**Usage:**

```bash
./acacia module enable my-api-module
```

#### `acacia module disable <name>`

Disables a module, preventing it from being loaded and started by the kernel. If the module is currently running, it will be gracefully stopped. This change is persisted in the application's configuration.

**Arguments:**

*   `<name>`: The name of the module to disable.

**Usage:**

```bash
./acacia module disable my-api-module
```

---

### `acacia registry`

Manages the registry of available modules and gateways. The registry is a JSON file that keeps track of the modules and gateways in your project.

By default, the registry file is located at `$ACACIA_CONFIG_DIR/registries.json` or `$(os.UserConfigDir)/acacia/registries.json`.

**Usage:**

```bash
./acacia registry [subcommand]
```

**Subcommands:**

#### `acacia registry modules`

Manages the module entries in the registry.

**Subcommands:**

*   `add <name>`: Adds a module to the registry.
    *   `--company string`: The company or organization associated with the module.
    *   `--author string`: The author of the module.
    *   `--version string`: The version of the module.
*   `remove <name>`: Removes a module from the registry.
*   `list`: Lists all registered modules.

**Example:**

```bash
./acacia registry modules add my-module --company "MyCorp" --author "John Doe"
```

#### `acacia registry gateways`

Manages the gateway entries in the registry.

**Subcommands:**

*   `add <name>`: Adds a gateway to the registry.
    *   `--company string`: The company or organization associated with the gateway.
    *   `--author string`: The author of the gateway.
    *   `--version string`: The version of the gateway.
*   `remove <name>`: Removes a gateway from the registry.
*   `list`: Lists all registered gateways.

**Example:**

```bash
./acacia registry gateways add http-gateway --version "1.0.0"
