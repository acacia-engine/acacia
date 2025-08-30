# Gateway Integration Patterns

This guide covers how Acacia modules integrate with gateways, based on the auth module's HTTP API integration and the framework's gateway system.

## Gateway System Overview

Gateways in Acacia provide external interfaces for modules:

- **HTTP API Gateway** - RESTful endpoints, middleware
- **Custom Gateways** - Message queues, WebSockets, etc.
- **Lifecycle Management** - Started after modules, stopped before modules
- **Security Integration** - Principal-based access control

## Gateway Interface

All gateways implement the `kernel.Gateway` interface:

```go
type Gateway interface {
    Name() string
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Configure(cfg interface{}) error
    ShutdownTimeout() time.Duration
}
```

## HTTP API Gateway Integration

### Basic HTTP Handler Registration

```go
// Gateway interface for HTTP API
type HTTPAPIGateway interface {
    RegisterRoute(method, path string, handler http.HandlerFunc) error
    RespondWithJSON(w http.ResponseWriter, r *http.Request, code int, payload interface{})
    RespondWithError(w http.ResponseWriter, r *http.Request, code int, message string)
}

// Module registering HTTP handlers
func (m *MyModule) RegisterHTTPHandlers(gateway interface{}) error {
    httpGateway, ok := gateway.(HTTPAPIGateway)
    if !ok {
        return fmt.Errorf("gateway does not implement HTTPAPIGateway")
    }

    // Register routes
    return httpGateway.RegisterRoute("GET", "/health", m.healthHandler)
}

func (m *MyModule) healthHandler(w http.ResponseWriter, r *http.Request) {
    // Handle health check request
    response := map[string]string{"status": "healthy"}
    httpGateway.RespondWithJSON(w, r, http.StatusOK, response)
}
```

## Real-World Examples

### Auth Module HTTP Integration

```go
// From modules/auth/auth.go
func (m *AuthModule) RegisterHTTPHandlers(gateway interface{}) error {
    httpGateway, ok := gateway.(interface {
        RegisterRoute(method, path string, handler http.HandlerFunc) error
        RespondWithJSON(w http.ResponseWriter, r *http.Request, code int, payload interface{})
        RespondWithError(w http.ResponseWriter, r *http.Request, code int, message string)
    })
    if !ok {
        return fmt.Errorf("gateway does not implement required HTTP API methods")
    }

    // Create wrapper functions for the gateway methods
    respondWithJSON := func(w http.ResponseWriter, r *http.Request, code int, payload interface{}) {
        httpGateway.RespondWithJSON(w, r, code, payload)
    }
    respondWithError := func(w http.ResponseWriter, r *http.Request, code int, message string) {
        httpGateway.RespondWithError(w, r, code, message)
    }

    // Register /register endpoint
    err := httpGateway.RegisterRoute("POST", "/register", application.NewRegisterHandler(m.authService, respondWithJSON, respondWithError))
    if err != nil {
        return fmt.Errorf("failed to register /register route: %w", err)
    }

    // Register /login endpoint
    err = httpGateway.RegisterRoute("POST", "/login", application.NewLoginHandler(m.authService, m.authService.JwtService, respondWithJSON, respondWithError))
    if err != nil {
        return fmt.Errorf("failed to register /login route: %w", err)
    }

    // Register /verify endpoint
    err = httpGateway.RegisterRoute("GET", "/verify", application.NewVerifyHandler(m.authService, m.authService.JwtService, respondWithJSON, respondWithError))
    if err != nil {
        return fmt.Errorf("failed to register /verify route: %w", err)
    }

    return nil
}
```

### Event-Driven Gateway Registration

```go
// From modules/auth/auth.go - listenForGatewayEvents
func (m *AuthModule) listenForGatewayEvents(ctx context.Context, eventCh <-chan events.TypedEvent) {
    for {
        select {
        case event, ok := <-eventCh:
            if !ok {
                return
            }
            if gatewayStartedEvent, isGatewayStarted := event.(kernel.GatewayStartedEvent); isGatewayStarted {
                if gatewayStartedEvent.GatewayName == "httpapi" {
                    // Create a context with Principal that has permission to access the httpapi gateway
                    principal := auth.NewDefaultPrincipal("auth", "module", []string{"core.*", "gateway.httpapi.access"})
                    gatewayCtx := auth.ContextWithPrincipal(ctx, principal)

                    // Retrieve the httpapi gateway and register handlers
                    gateway, err := m.registry.GetGateway(gatewayCtx, "httpapi")
                    if err != nil {
                        logger.Error(ctx, "Auth module: failed to get 'httpapi' gateway from registry during event handling", zap.Error(err))
                        continue
                    }

                    if err := m.RegisterHTTPHandlers(gateway); err != nil {
                        logger.Error(ctx, "Auth module: failed to register HTTP handlers with httpapi gateway during event handling", zap.Error(err))
                        continue
                    }

                    logger.Info(ctx, "Auth module: Successfully registered HTTP handlers with httpapi gateway.")
                    return // Handlers registered, no need to listen further for httpapi
                }
            }
        case <-ctx.Done():
            return
        }
    }
}
```

## Advanced Gateway Patterns

### Middleware Integration

```go
type AuthMiddleware struct {
    authService *application.AuthService
}

func (m *AuthMiddleware) Middleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Extract JWT token from header
        token := r.Header.Get("Authorization")
        if token == "" {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }

        // Verify token
        claims, err := m.authService.JwtService.ValidateToken(token)
        if err != nil {
            http.Error(w, "Invalid token", http.StatusUnauthorized)
            return
        }

        // Create principal from claims
        principal := auth.NewDefaultPrincipal(claims.UserID, "user", claims.Roles)

        // Add principal to request context
        ctx := auth.ContextWithPrincipal(r.Context(), principal)
        r = r.WithContext(ctx)

        // Call next handler
        next(w, r)
    }
}

// Register protected routes with middleware
func (m *MyModule) RegisterHTTPHandlers(gateway interface{}) error {
    httpGateway := gateway.(HTTPAPIGateway)
    middleware := &AuthMiddleware{authService: m.authService}

    // Protected route
    protectedHandler := middleware.Middleware(m.myProtectedHandler)
    return httpGateway.RegisterRoute("GET", "/protected", protectedHandler)
}
```

### CORS and Security Headers

```go
type CORSMiddleware struct {
    allowedOrigins []string
    allowedMethods []string
    allowedHeaders []string
}

func (c *CORSMiddleware) Middleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        origin := r.Header.Get("Origin")

        // Check if origin is allowed
        for _, allowed := range c.allowedOrigins {
            if allowed == "*" || allowed == origin {
                w.Header().Set("Access-Control-Allow-Origin", origin)
                break
            }
        }

        // Set other CORS headers
        w.Header().Set("Access-Control-Allow-Methods", strings.Join(c.allowedMethods, ", "))
        w.Header().Set("Access-Control-Allow-Headers", strings.Join(c.allowedHeaders, ", "))

        // Handle preflight requests
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }

        // Call next handler
        next(w, r)
    }
}
```

### Rate Limiting

```go
type RateLimiter struct {
    requests map[string][]time.Time
    mu       sync.RWMutex
    limit    int
    window   time.Duration
}

func (rl *RateLimiter) Middleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Get client identifier (IP, user ID, etc.)
        clientID := getClientID(r)

        rl.mu.Lock()
        now := time.Now()

        // Clean old requests
        if requests, exists := rl.requests[clientID]; exists {
            cutoff := now.Add(-rl.window)
            newRequests := make([]time.Time, 0)
            for _, reqTime := range requests {
                if reqTime.After(cutoff) {
                    newRequests = append(newRequests, reqTime)
                }
            }
            rl.requests[clientID] = newRequests
        }

        // Check rate limit
        if len(rl.requests[clientID]) >= rl.limit {
            rl.mu.Unlock()
            http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
            return
        }

        // Add current request
        rl.requests[clientID] = append(rl.requests[clientID], now)
        rl.mu.Unlock()

        // Call next handler
        next(w, r)
    }
}
```

## Gateway Discovery and Registration

### Safe Gateway Discovery

```go
func (m *MyModule) getGatewaySafely(ctx context.Context, gatewayName string) (interface{}, error) {
    // Create appropriate principal for gateway access
    principal := auth.NewDefaultPrincipal(
        m.Name(),
        "module",
        []string{"gateway." + gatewayName + ".access"},
    )

    gatewayCtx := auth.ContextWithPrincipal(ctx, principal)

    // Attempt to get gateway from registry
    gateway, err := m.registry.GetGateway(gatewayCtx, gatewayName)
    if err != nil {
        // Check for specific errors
        if errors.Is(err, registry.ErrServiceNotFound) {
            return nil, fmt.Errorf("gateway '%s' not available - ensure it is loaded", gatewayName)
        }
        return nil, fmt.Errorf("failed to access gateway '%s': %w", gatewayName, err)
    }

    return gateway, nil
}
```

### Deferred Gateway Registration

```go
type PendingRegistration struct {
    GatewayName string
    RegisterFunc func(gateway interface{}) error
}

func (m *MyModule) Start(ctx context.Context) error {
    // Subscribe to gateway events
    eventCh, cancel, err := m.eventBus.Subscribe(kernel.GatewayStartedEventType)
    if err != nil {
        return err
    }
    m.cancelSubscription = cancel

    // Store pending registrations
    m.pendingRegistrations = []PendingRegistration{
        {
            GatewayName: "httpapi",
            RegisterFunc: m.registerHTTPHandlers,
        },
        {
            GatewayName: "websocket",
            RegisterFunc: m.registerWebSocketHandlers,
        },
    }

    go m.handleGatewayEvents(ctx, eventCh)
    return nil
}

func (m *MyModule) handleGatewayEvents(ctx context.Context, eventCh <-chan events.TypedEvent) {
    for {
        select {
        case event, ok := <-eventCh:
            if !ok {
                return
            }
            if gatewayStartedEvent, isGatewayStarted := event.(kernel.GatewayStartedEvent); isGatewayStarted {
                m.processPendingRegistration(ctx, gatewayStartedEvent.GatewayName)
            }
        case <-ctx.Done():
            return
        }
    }
}

func (m *MyModule) processPendingRegistration(ctx context.Context, gatewayName string) {
    for i, pending := range m.pendingRegistrations {
        if pending.GatewayName == gatewayName {
            // Try to get gateway and register handlers
            if gateway, err := m.getGatewaySafely(ctx, gatewayName); err == nil {
                if err := pending.RegisterFunc(gateway); err == nil {
                    // Success - remove from pending list
                    m.pendingRegistrations = append(m.pendingRegistrations[:i], m.pendingRegistrations[i+1:]...)
                    logger.Info(ctx, "Successfully registered handlers with gateway", zap.String("gateway", gatewayName))
                } else {
                    logger.Error(ctx, "Failed to register handlers with gateway", zap.String("gateway", gatewayName), zap.Error(err))
                }
            }
        }
    }
}
```

## Custom Gateway Implementation

### WebSocket Gateway Example

```go
// Custom WebSocket gateway
type WebSocketGateway struct {
    name       string
    config     WebSocketConfig
    server     *WebSocketServer
    eventBus   events.Bus
}

type WebSocketConfig struct {
    Port     int    `mapstructure:"port"`
    Path     string `mapstructure:"path"`
    CertFile string `mapstructure:"certFile"`
    KeyFile  string `mapstructure:"keyFile"`
}

func (g *WebSocketGateway) Name() string {
    return g.name
}

func (g *WebSocketGateway) Configure(cfg interface{}) error {
    return mapstructure.Decode(cfg, &g.config)
}

func (g *WebSocketGateway) Start(ctx context.Context) error {
    g.server = NewWebSocketServer(g.config)

    // Register message handlers
    g.server.OnMessage("chat", g.handleChatMessage)
    g.server.OnConnect(g.handleClientConnect)
    g.server.OnDisconnect(g.handleClientDisconnect)

    // Start server
    return g.server.Start(ctx)
}

func (g *WebSocketGateway) Stop(ctx context.Context) error {
    if g.server != nil {
        return g.server.Stop(ctx)
    }
    return nil
}

func (g *WebSocketGateway) ShutdownTimeout() time.Duration {
    return 30 * time.Second
}

// Message handling
func (g *WebSocketGateway) handleChatMessage(clientID string, message []byte) {
    // Process message and publish event
    event := ChatMessageEvent{
        ClientID: clientID,
        Message:  string(message),
        Timestamp: time.Now(),
    }

    g.eventBus.Publish(context.Background(), "chat.message", event)
}
```

### Module Integration with Custom Gateway

```go
func (m *MyModule) registerWebSocketHandlers(gateway interface{}) error {
    wsGateway, ok := gateway.(interface{
        OnMessage(eventType string, handler func(clientID string, data []byte))
        Broadcast(eventType string, data []byte) error
    })
    if !ok {
        return fmt.Errorf("gateway does not implement WebSocket interface")
    }

    // Register message handlers
    wsGateway.OnMessage("myevent", m.handleWebSocketMessage)

    // Store gateway reference for sending messages
    m.webSocketGateway = wsGateway

    return nil
}

func (m *MyModule) handleWebSocketMessage(clientID string, data []byte) {
    // Process WebSocket message
    // Can publish events or interact with other modules
}

func (m *MyModule) broadcastToClients(message string) error {
    if m.webSocketGateway == nil {
        return fmt.Errorf("WebSocket gateway not available")
    }

    return m.webSocketGateway.Broadcast("notification", []byte(message))
}
```

## Gateway Testing Patterns

### Mock Gateway for Testing

```go
type MockHTTPGateway struct {
    registeredRoutes map[string]map[string]http.HandlerFunc // method -> path -> handler
    responses        []MockResponse
    mu               sync.RWMutex
}

type MockResponse struct {
    Method     string
    Path       string
    StatusCode int
    Response   interface{}
}

func (m *MockHTTPGateway) RegisterRoute(method, path string, handler http.HandlerFunc) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    if m.registeredRoutes == nil {
        m.registeredRoutes = make(map[string]map[string]http.HandlerFunc)
    }
    if m.registeredRoutes[method] == nil {
        m.registeredRoutes[method] = make(map[string]http.HandlerFunc)
    }

    m.registeredRoutes[method][path] = handler
    return nil
}

func (m *MockHTTPGateway) RespondWithJSON(w http.ResponseWriter, r *http.Request, code int, payload interface{}) {
    m.mu.Lock()
    defer m.mu.Unlock()

    m.responses = append(m.responses, MockResponse{
        Method:     r.Method,
        Path:       r.URL.Path,
        StatusCode: code,
        Response:   payload,
    })

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    json.NewEncoder(w).Encode(payload)
}

func (m *MockHTTPGateway) GetRegisteredRoutes() map[string]map[string]http.HandlerFunc {
    m.mu.RLock()
    defer m.mu.RUnlock()

    routes := make(map[string]map[string]http.HandlerFunc)
    for method, paths := range m.registeredRoutes {
        routes[method] = make(map[string]http.HandlerFunc)
        for path, handler := range paths {
            routes[method][path] = handler
        }
    }
    return routes
}

func (m *MockHTTPGateway) GetResponses() []MockResponse {
    m.mu.RLock()
    defer m.mu.RUnlock()

    responses := make([]MockResponse, len(m.responses))
    copy(responses, m.responses)
    return responses
}
```

### Testing Gateway Integration

```go
func TestMyModule_HTTPRegistration(t *testing.T) {
    // Setup
    mockGateway := &MockHTTPGateway{}
    mockRegistry := &MockRegistry{}
    mockEventBus := &MockEventBus{}

    // Register mock gateway
    mockRegistry.RegisterGateway("httpapi", mockGateway)

    module := &MyModule{
        registry: mockRegistry,
        eventBus: mockEventBus,
    }

    // Start module
    ctx := context.Background()
    err := module.Start(ctx)
    assert.NoError(t, err)

    // Simulate gateway started event
    event := kernel.GatewayStartedEvent{GatewayName: "httpapi"}
    err = mockEventBus.Publish(ctx, kernel.GatewayStartedEventType, event)
    assert.NoError(t, err)

    // Wait for event processing
    time.Sleep(100 * time.Millisecond)

    // Verify routes were registered
    routes := mockGateway.GetRegisteredRoutes()
    assert.Contains(t, routes["GET"], "/myendpoint")
    assert.Contains(t, routes["POST"], "/myaction")
}

func TestMyModule_HTTPHandler(t *testing.T) {
    mockGateway := &MockHTTPGateway{}
    module := &MyModule{}

    // Register handlers
    err := module.RegisterHTTPHandlers(mockGateway)
    assert.NoError(t, err)

    // Simulate request
    req := httptest.NewRequest("GET", "/myendpoint", nil)
    w := httptest.NewRecorder()

    // Get registered handler and call it
    routes := mockGateway.GetRegisteredRoutes()
    handler := routes["GET"]["/myendpoint"]
    handler(w, req)

    // Verify response
    assert.Equal(t, http.StatusOK, w.Code)

    responses := mockGateway.GetResponses()
    assert.Len(t, responses, 1)
    assert.Equal(t, http.StatusOK, responses[0].StatusCode)
}
```

## Best Practices

### 1. Gateway Access Security

```go
// ✅ Good: Use appropriate principal for gateway access
principal := auth.NewDefaultPrincipal(
    m.Name(),
    "module",
    []string{
        "gateway.httpapi.register",  // Only what you need
        "gateway.httpapi.access",
    },
)

// ❌ Bad: Overly broad permissions
principal := auth.NewDefaultPrincipal(m.Name(), "module", []string{"*"})
```

### 2. Error Handling in Gateway Operations

```go
// ✅ Good: Graceful error handling
func (m *MyModule) RegisterHTTPHandlers(gateway interface{}) error {
    httpGateway, ok := gateway.(HTTPAPIGateway)
    if !ok {
        return fmt.Errorf("incompatible gateway type: expected HTTPAPIGateway")
    }

    if err := httpGateway.RegisterRoute("GET", "/health", m.healthHandler); err != nil {
        return fmt.Errorf("failed to register health endpoint: %w", err)
    }

    return nil
}

// ❌ Bad: Silent failures
func (m *MyModule) RegisterHTTPHandlers(gateway interface{}) {
    httpGateway := gateway.(HTTPAPIGateway) // Panic if wrong type
    httpGateway.RegisterRoute("GET", "/health", m.healthHandler) // Ignore errors
}
```

### 3. Resource Management

```go
// ✅ Good: Proper cleanup
func (m *MyModule) Stop(ctx context.Context) error {
    // Clean up gateway connections, close channels, etc.
    if m.cancelSubscription != nil {
        m.cancelSubscription()
    }

    if m.webSocketServer != nil {
        return m.webSocketServer.Close()
    }

    return nil
}
```

### 4. Gateway Compatibility

```go
// ✅ Good: Interface-based design
type HTTPAPIGateway interface {
    RegisterRoute(method, path string, handler http.HandlerFunc) error
    RespondWithJSON(w http.ResponseWriter, r *http.Request, code int, payload interface{})
    RespondWithError(w http.ResponseWriter, r *http.Request, code int, message string)
}

// ❌ Bad: Concrete type coupling
func registerWithConcreteGateway(gateway *SpecificHTTPGatewayV1) error {
    // Breaks if gateway implementation changes
    return gateway.RegisterRoute("GET", "/test", handler)
}
```

## Common Patterns and Anti-Patterns

### ✅ Good Patterns

1. **Event-Driven Registration** - Register handlers when gateways become available
2. **Interface-Based Design** - Program to gateway interfaces, not implementations
3. **Security-First** - Use appropriate principals for gateway access
4. **Graceful Degradation** - Handle missing gateways gracefully
5. **Resource Cleanup** - Properly clean up connections and subscriptions

### ❌ Anti-Patterns

1. **Tight Coupling** - Direct dependency on specific gateway implementations
2. **Silent Failures** - Not handling gateway registration errors
3. **Over-Permissioning** - Requesting more gateway permissions than needed
4. **Blocking Operations** - Performing slow operations in event handlers
5. **Missing Error Handling** - Not checking for gateway availability

This guide provides comprehensive patterns for integrating Acacia modules with gateways, ensuring robust and secure communication with external systems.
