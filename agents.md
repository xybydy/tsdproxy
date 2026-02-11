# TsDProxy AI Agents Guide

> **Project:** TsDProxy - Tailscale Docker Proxy  
> **Language:** Go 1.25.5  
> **License:** MIT  
> **Repository:** https://github.com/almeidapaulopt/tsdproxy

---

## Project Overview

TsDProxy is a Go-based application that simplifies exposing Docker containers and services to Tailscale networks. It automatically creates Tailscale machines for tagged containers, enabling secure access via unique URLs without complex configurations.

### Core Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         TsDProxy                                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐   │
│  │   Target     │  │    Proxy     │  │      Dashboard       │   │
│  │  Providers   │──│   Manager    │──│    (Web UI + SSE)    │   │
│  └──────────────┘  └──────────────┘  └──────────────────────┘   │
│         │                 │                                     │
│         ▼                 ▼                                     │
│  ┌──────────────┐  ┌──────────────┐                             │
│  │    Docker    │  │  Tailscale   │                             │
│  │   Provider   │  │   Provider   │                             │
│  └──────────────┘  └──────────────┘                             │
│         │                 │                                     │
│         ▼                 ▼                                     │
│  ┌──────────────┐  ┌──────────────┐                             │
│  │   Docker     │  │  Tailscale   │                             │
│  │   Socket     │  │   Network    │                             │
│  └──────────────┘  └──────────────┘                             │
└─────────────────────────────────────────────────────────────────┘
```

---

## Directory Structure

```
.
├── cmd/                          # Application entry points
│   ├── healthcheck/              # Health check CLI tool
│   │   └── main.go
│   └── server/                   # Main server application
│       └── main.go
│
├── internal/                     # Private application code
│   ├── config/                   # Configuration management
│   │   ├── config.go             # Main config structures & validation
│   │   ├── configfile.go         # Config file I/O operations
│   │   ├── generateproviders.go  # Provider config generation
│   │   └── validator.go          # Config validation logic
│   │
│   ├── consts/                   # Application constants
│   │   ├── files.go
│   │   └── proxymanager.go
│   │
│   ├── core/                     # Core infrastructure
│   │   ├── const.go
│   │   ├── healthcheck.go        # Health check HTTP handler
│   │   ├── http.go               # HTTP server setup
│   │   ├── log.go                # Zerolog configuration
│   │   ├── pprof.go              # Profiling support
│   │   ├── sessions.go           # Session management
│   │   └── version.go            # Version information
│   │
│   ├── dashboard/                # Web dashboard
│   │   ├── dash.go               # Dashboard handlers & SSE
│   │   └── stream.go             # SSE streaming logic
│   │
│   ├── model/                    # Data models
│   │   ├── contextkey.go         # Context key definitions
│   │   ├── default.go            # Default values
│   │   ├── port.go               # Port configuration models
│   │   ├── proxyconfig.go        # Proxy configuration struct
│   │   ├── status.go             # Proxy status enums
│   │   └── whois.go              # Tailscale whois data
│   │
│   ├── proxymanager/             # Proxy lifecycle management
│   │   ├── port.go               # Port handling
│   │   ├── proxy.go              # Individual proxy logic
│   │   └── proxymanager.go       # Main proxy manager
│   │
│   ├── proxyproviders/           # Proxy provider abstractions
│   │   ├── proxyproviders.go     # Provider interface
│   │   └── tailscale/            # Tailscale implementation
│   │       ├── provider.go       # Tailscale provider
│   │       └── proxy.go          # Tailscale proxy instance
│   │
│   ├── targetproviders/          # Target provider abstractions
│   │   ├── targetproviders.go    # Provider interface
│   │   ├── docker/               # Docker implementation
│   │   │   ├── autodetect.go     # Auto-detection logic
│   │   │   ├── consts.go         # Docker labels & constants
│   │   │   ├── container.go      # Container operations
│   │   │   ├── docker.go         # Docker provider
│   │   │   ├── errors.go         # Error definitions
│   │   │   ├── legacy.go         # Legacy label support
│   │   │   └── utils.go          # Utility functions
│   │   └── list/                 # File list provider
│   │       └── list.go           # YAML list provider
│   │
│   └── ui/                       # UI components (Templ)
│       ├── ui.go
│       ├── components/
│       ├── layouts/
│       ├── pages/
│       │   └── proxylist.templ   # Proxy list page template
│       └── static/
│
├── web/                          # Frontend assets (Vite + Datastar)
│   ├── index.html
│   ├── package.json
│   ├── vite.config.js
│   ├── scripts.js                # Datastar/JS interactions
│   ├── styles.css
│   ├── tsdproxy-dark.css
│   ├── tsdproxy-light.css
│   ├── web.go                     # Static file embedding
│   └── public/                   # Static assets
│
├── docs/                         # Hugo documentation
│   ├── content/
│   │   └── docs/                 # Documentation pages
│   └── static/
│
├── dev/                          # Development configs
├── docker-compose.yaml
├── Dockerfile
├── go.mod
└── README.md
```

---

## Key Concepts

### 1. Target Providers

Target providers discover services to proxy. They monitor for changes and emit events.

| Provider | Purpose | Source |
|----------|---------|--------|
| `docker` | Monitor Docker containers | Docker socket API |
| `list` | Static YAML file targets | File system |

**Docker Labels (v2):**
- `tsdproxy.enable=true` - Enable proxying
- `tsdproxy.name=<hostname>` - Custom hostname
- `tsdproxy.proxyprovider=<name>` - Specific Tailscale provider
- `tsdproxy.port.<port>=<options>` - Port configuration
- `tsdproxy.ephemeral=true` - Ephemeral node
- `tsdproxy.runwebclient=true` - Enable web client
- `tsdproxy.tsnet_verbose=true` - Verbose logging
- `tsdproxy.authkey=<key>` - Per-container auth key
- `tsdproxy.tags=<tags>` - Tailscale tags
- `tsdproxy.dash.visible=true` - Dashboard visibility
- `tsdproxy.dash.label=<label>` - Dashboard display label
- `tsdproxy.dash.icon=<icon>` - Dashboard icon

### 2. Proxy Providers

Proxy providers create the actual network endpoints. Currently only Tailscale is supported.

**Tailscale Provider Features:**
- OAuth authentication (clientId/clientSecret)
- Auth key authentication
- Multiple providers per instance
- Custom control URLs (Headscale support)
- Tags support
- Ephemeral nodes

### 3. Proxy Manager

Central coordinator that:
- Manages proxy lifecycle (create/start/stop/remove)
- Routes events between providers
- Maintains proxy state
- Provides SSE stream for dashboard updates

### 4. Configuration

**Main Config (`/config/tsdproxy.yaml`):**

```yaml
defaultProxyProvider: default

docker:
  local:
    host: unix:///var/run/docker.sock
    targetHostname: host.docker.internal
    defaultProxyProvider: default
    tryDockerInternalNetwork: false

lists:
  critical:
    filename: /config/critical.yaml
    defaultProxyProvider: default
    defaultProxyAccessLog: true

tailscale:
  providers:
    default:
      authKey: ""
      authKeyFile: ""
      clientId: ""
      clientSecret: ""
      tags: ""
      controlUrl: https://controlplane.tailscale.com
  dataDir: /data/

http:
  hostname: 0.0.0.0
  port: 8080

log:
  level: info
  json: false

proxyAccessLog: true
```

**List Config (`/config/<name>.yaml`):**

```yaml
servicename:
  ports:
    - port: 8080
      scheme: http
      redirects:
        - scheme: https
          port: 8443
  proxyProvider: default
  proxyAccessLog: true
  tlsValidate: true
  tailscale:
    authKey: ""
    ephemeral: false
    runWebClient: false
    verbose: false
    tags: ""
  dashboard:
    visible: true
    label: "My Service"
    icon: "tsdproxy"
```

---

## Development Guidelines

### Go Best Practices & Patterns

#### 1. Project Structure (Standard Go Layout)

```
cmd/                    # Main applications - one per subdirectory
    server/             # Main server binary
    healthcheck/        # Health check CLI tool

internal/               # Private application code
    config/             # Configuration management
    core/               # Core infrastructure (HTTP, logging, sessions)
    model/              # Domain models and data structures
    proxymanager/       # Business logic - proxy lifecycle
    proxyproviders/     # Proxy provider implementations
    targetproviders/    # Target provider implementations
    dashboard/          # Web UI handlers
    ui/                 # Templ templates

web/                    # Frontend assets (Vite + Datastar)
docs/                   # Hugo documentation
dev/                    # Development configs
```

**Rules:**
- Keep `cmd/` thin - only entry points and wiring
- Business logic lives in `internal/`
- Use `internal/` to prevent external imports
- Group by domain/feature, not by layer

#### 2. Interface Design

**Define interfaces where they're used (consumer-side):**

```go
// In proxymanager package - defines what IT needs
type TargetProvider interface {
    AddTarget(id string) (*model.Config, error)
    RemoveTarget(id string)
    WatchEvents(ctx context.Context)
    Close()
}

// In proxyproviders package - defines what IT needs
type Provider interface {
    NewProxy(config *model.Config) (ProxyInterface, error)
}
```

**Interface Segregation:**
- Keep interfaces small and focused
- Compose larger interfaces from smaller ones
- Use compile-time interface checks: `var _ TargetProvider = (*Client)(nil)`

#### 3. Error Handling

**Use wrapped errors with context:**

```go
import "fmt"

// Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to inspect container %s: %w", id, err)
}

// Define sentinel errors for specific cases
var (
    ErrProxyProviderNotFound  = errors.New("proxy provider not found")
    ErrTargetProviderNotFound = errors.New("target provider not found")
)

// Check with errors.Is
if errors.Is(err, ErrProxyProviderNotFound) {
    // handle specific error
}
```

**Error Handling Rules:**
- Never ignore errors: `_ = someFunc()` ❌
- Wrap errors at package boundaries
- Use sentinel errors for API contracts
- Log errors at appropriate level (not everywhere)

#### 4. Structured Logging (Zerolog)

**Always use structured logging with context:**

```go
// Create logger with context
log := logger.With().
    Str("module", "proxymanager").
    Str("proxy", hostname).
    Logger()

// Use appropriate levels
log.Trace().Msg("Detailed debugging info")
log.Debug().Msg("Development debugging")
log.Info().Msg("General information")
log.Warn().Msg("Warning conditions")
log.Error().Err(err).Msg("Error occurred")
log.Fatal().Msg("Application cannot continue")

// Structured fields
log.Info().
    Str("container", containerID).
    Int("port", port).
    Bool("ephemeral", ephemeral).
    Msg("Starting proxy")
```

#### 5. Context Usage

**Pass context as first parameter:**

```go
func (c *Client) AddTarget(ctx context.Context, id string) (*model.Config, error) {
    // Use context for cancellation, timeouts, values
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    
    return c.docker.ContainerInspect(ctx, id)
}
```

**Rules:**
- Context is always the first parameter: `ctx context.Context`
- Don't store context in structs
- Pass context through the call chain
- Use `context.Background()` only at entry points

#### 6. Concurrency Patterns

**Use sync.RWMutex for shared state:**

```go
type ProxyManager struct {
    Proxies ProxyList
    mtx     sync.RWMutex
}

// Read operations
func (pm *ProxyManager) GetProxy(name string) (*Proxy, bool) {
    pm.mtx.RLock()
    defer pm.mtx.RUnlock()
    p, ok := pm.Proxies[name]
    return p, ok
}

// Write operations
func (pm *ProxyManager) addProxy(id string, proxy *Proxy) {
    pm.mtx.Lock()
    defer pm.mtx.Unlock()
    pm.Proxies[id] = proxy
}
```

**Use WaitGroup for goroutine coordination:**

```go
func (pm *ProxyManager) StopAllProxies() {
    wg := sync.WaitGroup{}
    
    pm.mtx.RLock()
    for id := range pm.Proxies {
        wg.Add(1)
        go func(id string) {
            defer wg.Done()
            pm.removeProxy(id)
        }(id)
    }
    pm.mtx.RUnlock()
    
    wg.Wait()
}
```

**Channel patterns for event streaming:**

```go
// Buffered channels for async communication
events := make(chan model.ProxyEvent, 100)

// Always close channels from sender
defer close(events)

// Use select for non-blocking operations
select {
case event := <-events:
    process(event)
case <-ctx.Done():
    return ctx.Err()
default:
    // non-blocking
}
```

#### 7. Configuration Management

**Use struct tags for validation and defaults:**

```go
type Config struct {
    // Validation tags
    Hostname string `validate:"ip|hostname,required"`
    Port     uint16 `validate:"numeric,min=1,max=65535"`
    
    // Default values
    Level string `default:"info" validate:"oneof=debug info warn error"`
    JSON  bool   `default:"false" validate:"boolean"`
    
    // YAML mapping
    DataDir string `yaml:"dataDir" validate:"dir"`
}

// Initialize with defaults
if err := defaults.Set(config); err != nil {
    return fmt.Errorf("error loading defaults: %w", err)
}

// Validate
if err := validator.Validate(config); err != nil {
    return fmt.Errorf("config validation failed: %w", err)
}
```

#### 8. Testing

**Table-driven tests:**

```go
func TestAddTarget(t *testing.T) {
    tests := []struct {
        name      string
        container string
        wantErr   bool
        errType   error
    }{
        {
            name:      "valid container",
            container: "nginx",
            wantErr:   false,
        },
        {
            name:      "not found",
            container: "missing",
            wantErr:   true,
            errType:   ErrContainerNotFound,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            client := New(mockLogger, "test", &config)
            _, err := client.AddTarget(context.Background(), tt.container)
            
            if tt.wantErr {
                assert.Error(t, err)
                if tt.errType != nil {
                    assert.ErrorIs(t, err, tt.errType)
                }
                return
            }
            assert.NoError(t, err)
        })
    }
}
```

**Use testify for assertions:**

```go
import "github.com/stretchr/testify/assert"
import "github.com/stretchr/testify/require"

// assert continues on failure
assert.Equal(t, expected, actual)
assert.NoError(t, err)

// require stops on failure
require.NoError(t, err)  // Fatal if error
require.NotNil(t, obj)   // Fatal if nil
```

**Mock external dependencies:**

```go
// Define interface for testability
type DockerClient interface {
    ContainerInspect(ctx context.Context, id string) (types.ContainerJSON, error)
}

// Use mock in tests
type mockDockerClient struct {
    inspectFunc func(ctx context.Context, id string) (types.ContainerJSON, error)
}

func (m *mockDockerClient) ContainerInspect(ctx context.Context, id string) (types.ContainerJSON, error) {
    return m.inspectFunc(ctx, id)
}
```

#### 9. Dependency Injection

**Wire dependencies explicitly:**

```go
// Dependencies as interfaces
type WebApp struct {
    Log          zerolog.Logger
    HTTP         *core.HTTPServer
    ProxyManager *pm.ProxyManager
    Dashboard    *dashboard.Dashboard
}

// Constructor injection
func NewProxyManager(logger zerolog.Logger) *ProxyManager {
    return &ProxyManager{
        Proxies:           make(ProxyList),
        TargetProviders:   make(TargetProviderList),
        ProxyProviders:    make(ProxyProviderList),
        log:               logger.With().Str("module", "proxymanager").Logger(),
    }
}

// Initialize in main
func InitializeApp() (*WebApp, error) {
    logger := core.NewLog()
    httpServer := core.NewHTTPServer(logger)
    proxyManager := pm.NewProxyManager(logger)
    dash := dashboard.NewDashboard(httpServer, logger, proxyManager)
    
    return &WebApp{
        Log:          logger,
        HTTP:         httpServer,
        ProxyManager: proxyManager,
        Dashboard:    dash,
    }, nil
}
```

#### 10. Code Quality Tools

```bash
# Format code
go fmt ./...

# Vet for common mistakes
go vet ./...

# Run linter
golangci-lint run

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Race detector
go test -race ./...

# Static analysis
codeql analyze

# Security scanning
gosec ./...
```

#### 11. Documentation

**Document all exported items:**

```go
// ProxyManager manages the lifecycle of all proxies.
// It coordinates between target providers and proxy providers,
// handling events and maintaining proxy state.
type ProxyManager struct {
    // Proxies holds all active proxy instances.
    // Protected by mtx mutex.
    Proxies ProxyList
    
    // log is the structured logger for this component.
    log zerolog.Logger
    
    // mtx protects Proxies map for concurrent access.
    mtx sync.RWMutex
}

// NewProxyManager creates a new ProxyManager with the given logger.
// It initializes empty maps for providers and proxies.
func NewProxyManager(logger zerolog.Logger) *ProxyManager

// Start initializes all providers and begins watching for events.
// Must be called after initialization and before any proxy operations.
func (pm *ProxyManager) Start()
```

### Adding a New Target Provider

1. **Create directory:** `internal/targetproviders/<name>/`
2. **Define config struct:**
   ```go
   type MyProviderConfig struct {
       Endpoint string `validate:"required,url" yaml:"endpoint"`
       Timeout  int    `default:"30" validate:"min=1" yaml:"timeout"`
   }
   ```
3. **Implement interface:**
   ```go
   type Client struct { /* ... */ }
   var _ targetproviders.TargetProvider = (*Client)(nil)
   ```
4. **Register in config:** Add to `internal/config/config.go`
5. **Register in manager:** Add to `proxymanager.addTargetProviders()`
6. **Add tests:** Create `internal/targetproviders/<name>/<name>_test.go`
7. **Document:** Update `docs/content/docs/`

### Adding a New Proxy Provider

1. **Create directory:** `internal/proxyproviders/<name>/`
2. **Implement Provider interface:**
   ```go
   type Client struct { /* ... */ }
   func (c *Client) NewProxy(config *model.Config) (proxyproviders.ProxyInterface, error)
   ```
3. **Implement ProxyInterface:**
   ```go
   type Proxy struct { /* ... */ }
   func (p *Proxy) Start(ctx context.Context) error
   func (p *Proxy) Stop() error
   func (p *Proxy) GetStatus() model.ProxyStatus
   ```
4. **Register in config:** Add to `internal/config/config.go`
5. **Register in manager:** Add to `proxymanager.addProxyProviders()`
6. **Add tests** and **document**

### Testing Commands

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific test
go test -run TestAddTarget ./internal/targetproviders/docker/...

# Run with race detector
go test -race ./...

# Run benchmarks
go test -bench=. ./...

# Generate coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Integration tests
go test -tags=integration ./...

# Build and run
go build -o tsdproxy ./cmd/server
./tsdproxy -config=/path/to/config.yaml

# Development with hot reload
air  # requires .air.toml

# Docker development
docker compose -f dev/docker-compose.yaml up --build
```

### Debugging & Profiling

```go
// Add to main.go for pprof
import _ "net/http/pprof"

// Or use internal/core/pprof.go
pprof.StartCPUProfile(w)
defer pprof.StopCPUProfile()

// Runtime metrics
log.Debug().
    Int("goroutines", runtime.NumGoroutine()).
    Uint64("memory", memStats.Alloc).
    Msg("runtime stats")
```

Access pprof at `http://localhost:8080/debug/pprof/`

### Build & Release

```bash
# Local build
go build -ldflags "-X main.version=dev -X main.commit=$(git rev-parse --short HEAD)" \
    -o tsdproxy ./cmd/server

# Cross-compile
GOOS=linux GOARCH=amd64 go build -o tsdproxy-linux-amd64 ./cmd/server
GOOS=darwin GOARCH=arm64 go build -o tsdproxy-darwin-arm64 ./cmd/server

# Docker build
docker build -t tsdproxy:dev .

# Release with goreleaser
goreleaser release --clean

# Development release
goreleaser release -f .goreleaser-dev.yaml --clean --snapshot
```

### Common Pitfalls to Avoid

1. **Don't ignore errors** - Always handle errors appropriately
2. **Don't use init()** - Use explicit initialization
3. **Don't use global state** - Pass dependencies explicitly
4. **Don't panic** - Return errors and handle gracefully
5. **Don't use reflection unnecessarily** - Prefer type safety
6. **Don't ignore context cancellation** - Respect ctx.Done()
7. **Don't leak goroutines** - Always have exit conditions
8. **Don't forget mutex unlock** - Use defer
9. **Don't use bare select{}** - Always include exit case
10. **Don't log and return** - Choose one: log OR return

### Code Review Checklist

- [ ] Errors are wrapped with context
- [ ] Context is passed and respected
- [ ] Mutexes protect shared state
- [ ] Interfaces are consumer-defined
- [ ] Tests cover success and error paths
- [ ] Logging uses appropriate levels
- [ ] Documentation is complete
- [ ] No goroutine leaks
- [ ] Race detector passes
- [ ] Linter passes with no issues
- [ ] Dependencies are injected
- [ ] Configuration has validation tags
- [ ] Exported items are documented
- [ ] No magic numbers/strings (use consts)

---

## Common Tasks

### Adding a New Docker Label

1. Add constant in `internal/targetproviders/docker/consts.go`
2. Parse in `internal/targetproviders/docker/container.go`
3. Map to `model.Config` field
4. Document in `docs/content/docs/docker.md`

### Modifying Proxy Behavior

1. Update `model.Config` struct in `internal/model/proxyconfig.go`
2. Modify proxy implementation in `internal/proxymanager/proxy.go`
3. Update validation in `internal/config/validator.go`

### Dashboard Updates

The dashboard uses Server-Sent Events (SSE) for real-time updates:
- Events flow: `ProxyManager` → `Dashboard` → `SSE` → Browser
- Frontend uses Datastar framework (`web/scripts.js`)
- Templates use Templ (`internal/ui/pages/`)

---

## Dependencies

**Key External Libraries:**
- `tailscale.com` - Tailscale tsnet client
- `github.com/docker/docker` - Docker API client
- `github.com/a-h/templ` - HTML templating
- `github.com/starfederation/datastar` - Frontend reactivity
- `github.com/rs/zerolog` - Structured logging
- `github.com/go-playground/validator` - Config validation
- `github.com/fsnotify/fsnotify` - File watching

---

## Version Information

- **v1.x**: Stable, maintenance mode
- **v2.x** (current): Beta, active development
  - Multi-port support
  - OAuth authentication
  - Real-time dashboard
  - Tag support
  - Swarm stack support

---

## Resources

- [Official Documentation](https://almeidapaulopt.github.io/tsdproxy/)
- [GitHub Issues](https://github.com/almeidapaulopt/tsdproxy/issues)
- [Tailscale Documentation](https://tailscale.com/kb/)
- [tsnet Documentation](https://pkg.go.dev/tailscale.com/tsnet)

---

*Last Updated: 2025-02-02*
