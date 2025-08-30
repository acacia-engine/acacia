# Acacia Engine Documentation

This document provides comprehensive documentation for the Acacia Engine, a high-performance, open-source, and modular game backend engine written in Go.

## 1. Introduction

Acacia is designed to provide game developers with a robust, scalable, and easy-to-use foundation for their game's backend services. It prioritizes raw performance, resource efficiency, and architectural clarity by leveraging Go's strengths in concurrency and its straightforward, explicit approach to application design.

For more details on philosophy and module guidelines, see [core/GUIDELINES.md](GUIDELINES.md).

### Core Philosophy

Acacia is built on the following core principles:

*   **Modular, Not Monolithic:** The engine is built as a lean **Core** with features added as self-contained **Modules**. This allows developers to use only the features they need and makes the codebase easy to navigate and contribute to.
*   **Clean, Layered Architecture:** Each module follows a strict, layered architecture (`Domain`, `Application`, `Infrastructure`, `Gateway`) inspired by Domain-Driven Design. This ensures a clean separation of concerns, making the code highly testable and maintainable.
*   **Explicit Over Magic:** We favor Go's idiomatic, explicit approach to dependency injection. There is no "magic" container. All dependencies are wired together explicitly in the main application entry point, making the entire application flow easy to trace and understand.
*   **High Performance by Default:** By using Go, Acacia achieves incredible performance and concurrency with a low memory footprint, making it suitable for games that scale from a small Game Jam project to a massive commercial success.

## 2. Architecture Overview

Acacia's architecture is composed of a central **Core** and a collection of **Modules**. Each module is a "vertical slice" that cuts across the horizontal layers. The system follows a clean, layered architecture (Gateway, Application, Domain, Infrastructure) where dependencies only point inwards, protecting the core business logic.

## Additional Documentation

*   [**Auth Module**](auth.md): Provides interfaces and types for access control.
*   [**Config Module**](config.md): Handles application configuration loading and management.
*   [**Errors Module**](errors.md): Defines common application-wide error types and utilities.
*   [**Events Module**](events.md): Implements a minimal publish-subscribe event bus.
*   [**Kernel Module**](kernel.md): The central coordinator for managing the lifecycle of modules and gateways.
*   [**Logger Module**](logger.md): Sets up structured logging with access control.
*   [**Jobs Module**](jobs.md): Provides interfaces for asynchronous task processing and background jobs.
*   [**Registry Module**](registry.md): Offers a centralized mechanism for service registration and retrieval.
*   [**Metrics Module**](metrics.md): Manages application metrics collection using Prometheus.
*   [**Utils Module**](utils.md): Contains generic utility functions.
*   [**CLI Usage**](cli.md): Details on building and using the Acacia Engine CLI.
*   [**Docker Usage**](docker.md): Information on building and running the Acacia Engine with Docker.
