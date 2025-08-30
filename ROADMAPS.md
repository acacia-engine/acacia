# Roadmap for Core and Kernel Polishing and Enhancements

This document outlines the proposed plan for polishing and improving the `core/kernel` and `core/auth` packages, as well as introducing new foundational capabilities to the `core` module.

## Remaining Polishing and Improvements:

### 1. Address `auth` Package Inconsistencies and Missing Components:

*   **More Robust Principal Management:**
    *   **Rationale:** The `auth` module is already in `core` because access control is a fundamental, cross-cutting concern. Enhancing `Principal` management (e.g., with factories or helpers) is about improving the usability and flexibility of this core component, especially when integrating with diverse external authentication systems.
    *   **Action:** Consider adding a `PrincipalFactory` interface or a set of helper functions within the `auth` module to simplify the creation of `Principal` objects from various authentication sources (e.g., JWT claims, session data). This would make it easier for gateways or other authentication layers to construct the `Principal` correctly.

### 2. Address `kernel` Package Improvements:

*   **Dynamic Module Dependency Re-evaluation:**
    *   **Status:** Acknowledged in code comments. This is a significant feature change beyond "polishing" and is not implemented.
*   **Gateway Dependency Management:**
    *   **Status:** Not implemented. This is a potential future enhancement.
*   **Configuration Hot Reloading Robustness:**
    *   **Status:** Partially addressed. The kernel logs errors if `OnConfigChanged` fails, but a full rollback mechanism is not implemented. Further investigation is needed for robust strategies.
*   **Resource Management on `OnLoad` Failure:**
    *   **Status:** Documented. Best practices for resource cleanup in `OnLoad` are implicitly handled by the module's implementation. A specific `OnLoadFailed` hook is not implemented.
*   **Context Propagation in `AddGateway` and `RemoveGateway`:**
    *   **Status:** Not explicitly implemented with derived contexts. `context.Background()` is still used for immediate start/stop operations.

## New Core Module Enhancements (To Be Implemented):

*   **Enhanced Distributed Tracing (`core/tracing`):**
    *   **Rationale:** While basic OpenTelemetry integration exists in the kernel, a dedicated `core/tracing` module would centralize and enhance tracing capabilities. This module would provide utilities for creating spans, propagating context, and exporting traces, making it easier to instrument other modules and gateways consistently.
    *   **Action:** Create a new `core/tracing` module that provides standardized utilities for distributed tracing using OpenTelemetry.
*   **Enhanced Job Queueing and Persistence (`core/queue` or `core/broker`):**
    *   **Rationale:** The `jobs` module currently defines excellent interfaces for asynchronous tasks. However, for production environments, an in-memory queue is insufficient. A robust job system requires persistence, retries, and potentially distributed processing capabilities.
    *   **Action:** Introduce a `core/queue` or `core/broker` module that provides concrete implementations of the `jobs.Enqueuer` and `jobs.Worker` interfaces, backed by external message brokers or persistent queues (e.g., Redis, RabbitMQ, Kafka). This module could also include features like scheduled jobs (cron-like) and dead-letter queues.
*   **Standardized Health Check Framework (`core/health`):**
    *   **Rationale:** As the application grows and integrates more modules and gateways, a unified way to report the health status of individual components and the overall system becomes vital for operational monitoring and orchestration (e.g., Kubernetes readiness/liveness probes).
    *   **Action:** Create a new `core/health` module that defines a `HealthChecker` interface. Modules and gateways could implement this interface to report their internal health. The kernel could then expose an aggregated health endpoint (e.g., `/health` or `/ready`) that queries all registered health checkers.
*   **Configuration Reloading Trigger Mechanism:**
    *   **Rationale:** The `config` module supports configuration change hooks (`AddConfigChangeHook`) and the `kernel.Module` interface includes `OnConfigChanged`. However, the mechanism to *trigger* a configuration reload (e.g., by watching the config file, receiving a signal, or via an API call) is not explicitly part of the `config` module's documentation.
    *   **Action:** Implement a core mechanism (perhaps within the `kernel` or a new `core/admin` module) that listens for configuration file changes (e.g., using `fsnotify`) or exposes an internal API endpoint to trigger a reload, which would then propagate to modules via the `OnConfigChanged` method.
