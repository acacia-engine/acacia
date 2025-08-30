# Acacia Game Backend Engine

Acacia is a high-performance, open-source, and modular backend engine written in Go, designed to provide developers with a robust, scalable, and easy-to-use foundation for their backend services.

The project prioritizes **raw performance, resource efficiency, and architectural clarity** by leveraging Go's strengths in concurrency and its straightforward approach to application design.

## Core Philosophy

Acacia stands out with these fundamental principles:

1. **Modular Design**: A lean core with self-contained modules, allowing developers to pick only what they need.
2. **Clean Architecture**: Inspired by Domain-Driven Design, ensuring separation of concerns and high maintainability.
3. **Explicit Approach**: No hidden magic—dependencies are wired explicitly for easy traceability.
4. **High Performance**: Built in Go for exceptional concurrency, low memory use, and scalability.

## High-Level Architecture

Acacia follows a layered architecture with dependencies pointing inwards to protect business logic:

- **Gateway Layer**: Handles protocols like REST APIs or WebSockets, translating requests to use cases.
- **Application Layer**: Orchestrates business processes using domain objects and repositories.
- **Domain Layer**: Contains pure business logic, entities, and rules, isolated from external systems.
- **Infrastructure Layer**: Implements concrete adapters for databases, APIs, etc.

The structure is modular, with a central core and vertical modules cutting across layers.

## Getting Started

### Prerequisites
- Go 1.23 or higher
- Docker & Docker Compose (recommended for local development)

### Quick Start
1. Clone the repository:
   ```bash
   git clone https://github.com/acacia-engine/acacia.git
   cd acacia
   ```
2. Run the application:
   ```bash
   make run
   ```

For detailed instructions, see the project documentation.
