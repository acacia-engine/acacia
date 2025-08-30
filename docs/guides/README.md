# Acacia Module Development Guides

This directory contains comprehensive guides for developing modules in the Acacia framework. These guides are based on the actual codebase patterns and real module implementations.

## 🚀 Quick Start

If you're new to Acacia module development:

1. **Start with [Domain-Driven Module Architecture](./domain-driven-design.md)** - Understand the layered architecture
2. **Learn [Service Registry Patterns](./service-registry.md)** - Master cross-module communication
3. **Study the [Auth Module Example](./examples/auth-module.md)** - See a complete implementation

## 📚 Guide Overview

### Core Patterns (Phase 1) ✅
These guides cover the fundamental patterns every module developer should know:

#### [Security and Authentication Patterns](./security.md)
- Principal-based security model
- Access control integration
- RBAC (Role-Based Access Control)
- Context propagation
- Authentication best practices
- **Based on**: Core authentication system patterns

#### [Cross-Module Communication](./service-registry.md)
- Service registry usage
- Interface-based service registration
- Service discovery patterns
- Event-driven communication
- Gateway integration
- **Based on**: Auth module's service registration and discovery

#### [Domain-Driven Module Architecture](./domain-driven-design.md)
- Layered architecture (domain, application, infrastructure)
- Repository patterns
- Clean Architecture principles
- Dependency injection
- Testing strategies
- **Based on**: Auth module's DDD implementation

### Advanced Integration (Phase 2) ✅
These guides cover advanced patterns for complex integrations:

#### [Event-Driven Architecture](./events.md)
- Event publishing and subscription
- System events (module/gateway lifecycle)
- Custom domain events
- Event handling patterns
- Error handling and dead letter queues
- **Based on**: Auth module's complex event handling

#### [Gateway Integration Patterns](./gateway-integration.md)
- HTTP API gateway integration
- Event-driven gateway registration
- Middleware patterns (auth, CORS, rate limiting)
- Custom gateway implementation
- Security considerations
- **Based on**: Auth module's HTTP API integration

### Quality and Testing (Phase 3) 🔄
*Coming in next phase - Testing strategies, performance optimization, debugging*

## 🎯 Learning Path

### For New Module Developers

1. **Read**: [Domain-Driven Module Architecture](./domain-driven-design.md)
   - Understand layered architecture
   - Learn repository and service patterns

2. **Read**: [Service Registry Patterns](./service-registry.md)
   - Master cross-module communication
   - Learn event-driven patterns

3. **Study**: Module patterns and best practices
   - Apply the patterns from these guides
   - Understand the framework architecture
   - Build your first module incrementally

4. **Read**: [Security and Authentication](./security.md)
   - Understand principal-based security
   - Learn access control patterns

### For Advanced Integration

1. **Read**: [Event-Driven Architecture](./events.md)
   - Advanced event patterns
   - Error handling strategies

2. **Read**: [Gateway Integration Patterns](./gateway-integration.md)
   - HTTP API integration
   - Custom gateway development

## 📖 Pattern References

The guides draw inspiration from common patterns used across module implementations, including:

- **Authentication patterns** - User management, JWT handling, access control
- **Data persistence** - Repository patterns, database integration
- **API services** - HTTP handlers, middleware, request/response patterns
- **Event processing** - Domain events, system events, async processing

These patterns are demonstrated through generic examples that can be adapted to any module type.

## 🔧 Development Workflow

### 1. Module Creation
```bash
# Create module structure
mkdir -p modules/mymodule/{domain,application,infrastructure}
mkdir -p modules/mymodule/domain/{model,service,repository}
```

### 2. Implement Core Interfaces
- Implement `kernel.Module` interface
- Define domain entities and repository interfaces
- Create application services

### 3. Add Communication
- Register services with the registry
- Subscribe to relevant events
- Integrate with gateways if needed

### 4. Security Integration
- Implement proper principal handling
- Add access control checks
- Use secure communication patterns

## 🧪 Testing Your Module

Each guide includes testing patterns:

- **Unit tests** for domain logic
- **Integration tests** for service interactions
- **Mock testing** for external dependencies
- **Event testing** for asynchronous behavior

## 📋 Best Practices Summary

### Architecture
- ✅ Use domain-driven design principles
- ✅ Keep domain layer pure and testable
- ✅ Implement repository pattern for data access
- ✅ Use dependency injection

### Communication
- ✅ Register interfaces, not implementations
- ✅ Use events for loose coupling
- ✅ Handle service discovery errors gracefully
- ✅ Propagate context with principals

### Security
- ✅ Implement principal-based access control
- ✅ Use least privilege principle
- ✅ Validate all inputs
- ✅ Handle authentication in middleware

### Error Handling
- ✅ Provide meaningful error messages
- ✅ Use structured logging
- ✅ Implement proper resource cleanup
- ✅ Handle asynchronous errors

## 🚨 Common Pitfalls

### Avoid These Anti-Patterns:
- ❌ Direct module imports (use service registry)
- ❌ Global state management
- ❌ Tight coupling between modules
- ❌ Missing error handling
- ❌ Ignoring security considerations

## 📚 Additional Resources

- **[API Reference](../api/modules_and_gateways.md)** - Interface documentation
- **[Core Kernel Documentation](../core/kernel.md)** - Kernel API reference
- **[Configuration Guide](./configuration.md)** - Coming soon
- **[Testing Guide](./testing.md)** - Coming soon

## 🤝 Contributing

These guides are based on your actual codebase patterns. When you implement new patterns:

1. Document them in the appropriate guide
2. Add real code examples from your implementation
3. Update the examples and best practices

## 📞 Support

If you need help with module development:

1. **Check the guides** for your specific pattern
2. **Study the auth module** as a complete example
3. **Review the API documentation** for interface details
4. **Look at existing modules** for implementation patterns

---

*These guides are continuously updated based on real-world usage patterns and feedback from the Acacia community.*
