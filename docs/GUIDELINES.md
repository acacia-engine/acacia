# Core Module Development Philosophy

This document outlines the guiding principles for developing modules within the `core` directory of the Acacia Engine. These principles apply to both human developers and automated systems (AI) contributing to the codebase. The aim is to ensure a robust, maintainable, and scalable architecture by enforcing strict separation of concerns and clear responsibilities.

## Core Principles

1.  **Strict Abstraction and Decoupling:** Core modules must operate at a high level of abstraction. They should define interfaces and contracts, but must not depend on concrete implementations or internal details of specific modules.

2.  **Unidirectional Dependencies:** Dependencies flow from modules *towards* core, never the other way around. Core modules provide foundational services that modules consume.

3.  **Clarity and Specificity:** Guidelines must be explicit and unambiguous. Avoid vague or abstract language. Clearly state what actions are permitted and what are prohibited.

## Specific Guidelines

### What Core Modules MUST NOT Do

*   **Know Module Details:** Core modules (including the kernel) must never know about any specific detail of a module. This includes:
    *   Importing module-specific packages or types
    *   Referencing module-specific functions, variables, or data structures
    *   Making assumptions about a module's internal logic, state, or implementation
    *   Accessing module-specific configuration directly
*   **Initiate Module-Specific Logic:** Core modules must not directly call or trigger logic specific to a particular module. Interaction must occur through well-defined, generic interfaces or event mechanisms.
*   **Depend on Module Existence:** Core functionality must not depend on the presence or absence of any specific module. It should function independently of individual modules.

### What Core Modules SHOULD Do

*   **Define Generic Interfaces:** Provide abstract interfaces and contracts for modules to implement or interact with.
*   **Emit Generic Events:** Use a generic event bus (e.g., `core/events/bus.go`) to communicate state changes or trigger actions. Events should carry only generic data, not module-specific payloads.
*   **Provide Foundational Services:** Offer essential, generalized services (e.g., logging, configuration, metrics, error handling) that are universally applicable and do not embed module-specific logic.
*   **Enforce Security and Access Control:** Implement generic access control mechanisms (see `core/auth/access_controller.go`) that can be configured by modules but do not hardcode module-specific permissions.

## Rationale

Adhering to these guidelines ensures:
*   **Modularity:** Modules can be developed, tested, and deployed independently.
*   **Maintainability:** Changes in one module do not necessitate changes in core, reducing ripple effects.
*   **Scalability:** The system can easily integrate new modules without altering core functionality.
*   **Testability:** Core components can be tested in isolation without needing specific modules.
*   **Predictability:** The system's behavior becomes more predictable due to clear architectural boundaries.

---

For engine build, test, and project structure guidelines, see [docs/README.md](README.md).
