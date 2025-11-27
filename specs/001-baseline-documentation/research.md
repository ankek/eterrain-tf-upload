# Research: Terraform Backend Service - Technology Choices and Patterns

**Feature**: 001-baseline-documentation
**Date**: 2025-11-24
**Purpose**: Document technology choices, architectural patterns, and design decisions in the existing implementation

## Overview

This research document captures the technology choices and architectural patterns used in the Terraform Backend Service. Since this is a baseline documentation feature, this represents post-implementation analysis rather than pre-implementation research.

## Technology Stack Decisions

### Core Language: Go 1.25.0

**Decision**: Use Go (Golang) as the primary implementation language

**Rationale**:
- Native HTTP server capabilities in standard library (net/http)
- Excellent concurrency support for handling multiple organization requests
- Strong type safety for security-critical authentication and validation code
- Cross-platform compilation for deployment flexibility
- Minimal runtime dependencies for container deployment
- Fast startup time for containerized environments
- Built-in testing framework with benchmark support

**Alternatives Considered**:
- Python (FastAPI): Would require additional dependencies, slower performance for concurrent requests
- Node.js (Express): Less type safety, more complex dependency management
- Rust: Steeper learning curve, longer development time for MVP

### HTTP Router: chi v5.2.3

**Decision**: Use chi router for HTTP routing and middleware composition

**Rationale**:
- Lightweight and idiomatic Go HTTP routing
- Built-in middleware support for common patterns (logging, recovery, request ID)
- Context-based request handling aligns with Go best practices
- Standard library compatible (uses http.Handler interface)
- Active maintenance and stable API

**Alternatives Considered**:
- Standard library http.ServeMux: Limited routing capabilities (no path parameters)
- Gorilla Mux: Heavier weight, declining maintenance
- Gin: Opinionated framework with custom context, less standard library aligned

### Authentication: bcrypt + fsnotify

**Decision**: Use bcrypt for API key hashing and fsnotify for credential file watching

**Rationale**:
- Bcrypt provides adaptive cost factor for long-term security (golang.org/x/crypto/bcrypt)
- File-based credential storage (auth.cfg) enables hot-reload without service restart
- fsnotify provides cross-platform file system event monitoring with 500ms debounce
- Constant-time comparison prevents timing attacks (subtle.ConstantTimeCompare)
- No external database dependency for authentication simplifies deployment

**Alternatives Considered**:
- Plain text API keys: Unacceptable security risk if file is compromised
- Database-backed auth: Adds dependency and complexity for MVP
- JWT tokens: More complex, requires token lifecycle management

### Storage Layer: Multi-Mode Architecture

**Decision**: Support four storage modes: CSV, MySQL, dual (CSV+MySQL), and memory

**Rationale**:
- CSV mode: Simple file-based storage, no database required, easy backup/migration
- MySQL mode: Relational database for structured queries, organization isolation via tables
- Dual mode: Redundancy with graceful degradation, automatic backup
- Memory mode: Fast state backend for Terraform HTTP backend protocol
- Interface-driven design (Storage, DataStorage) enables switching without handler changes

**Alternatives Considered**:
- PostgreSQL: Would require additional driver dependency, similar capabilities to MySQL
- SQLite: File-based but no multi-tenant table isolation, locking concerns
- NoSQL (MongoDB): Overkill for append-only data structure
- S3/Object Storage: Network dependency, latency, complexity

### Database Driver: go-sql-driver/mysql v1.9.3

**Decision**: Use official MySQL driver for Go

**Rationale**:
- Most widely used MySQL driver in Go ecosystem
- Supports MySQL 8.4+ with modern authentication
- Compatible with database/sql standard library interface
- Active maintenance and security updates

**Alternatives Considered**:
- Pure Go MySQL libraries: Less battle-tested
- PostgreSQL drivers: Different SQL dialect, additional dependency

### Validation Strategy: Custom validation package

**Decision**: Implement custom JSON validation with size, depth, and complexity limits

**Rationale**:
- JSON size limit (10MB): Prevents DoS attacks via oversized payloads
- Depth limit (10 levels): Prevents stack overflow from deeply nested structures
- Complexity limit (1000 elements): Prevents resource exhaustion from large arrays/objects
- Semantic validation: Provider/category/resource_type pattern matching (alphanumeric + underscore/hyphen)
- Pre-storage validation: All checks pass before any storage operation

**Alternatives Considered**:
- JSON Schema validation: Heavier weight, external dependency, less control over limits
- No validation: Unacceptable security risk for multi-tenant service
- Minimal validation: Insufficient protection against malicious inputs

## Architectural Patterns

### Interface-Driven Storage Design

**Pattern**: Define Storage and DataStorage interfaces with multiple implementations

**Rationale**:
- Handlers depend on interfaces, not concrete implementations
- Easy to add new storage backends (PostgreSQL, S3, etc.)
- Testability: Mock implementations for unit tests
- Dual storage pattern wraps two implementations transparently

**Implementation**:
```go
type Storage interface {
    GetState(orgID uuid.UUID, name string) (*StateData, error)
    PutState(orgID uuid.UUID, name string, data []byte) error
    DeleteState(orgID uuid.UUID, name string) error
    LockState(orgID uuid.UUID, name string, lockInfo *LockInfo) error
    UnlockState(orgID uuid.UUID, name string, lockID string) error
    GetLock(orgID uuid.UUID, name string) (*LockInfo, error)
}

type DataStorage interface {
    AppendData(orgID uuid.UUID, data map[string]interface{}) error
    GetOrgData(orgID uuid.UUID) ([]DataUpload, error)
}
```

### Per-Organization Isolation

**Pattern**: Isolate storage per organization using UUID-based separation

**Rationale**:
- CSV mode: Separate file per organization (`{org-uuid}.csv`)
- MySQL mode: Separate table per organization (`org_{uuid_with_underscores}`)
- Memory mode: Keyed by organization UUID + state name
- Prevents cross-organization data leakage
- Enables per-organization backup and migration

**Implementation Details**:
- MySQL table names: Replace hyphens with underscores for SQL compatibility
- File system: Use UUID as filename with .csv extension
- Access control: Authentication middleware populates context with org ID

### Middleware Composition

**Pattern**: Compose authentication, rate limiting, and security middleware using chi router

**Rationale**:
- Separation of concerns: Each middleware has single responsibility
- Order matters: Auth first, then rate limiting (needs org ID from auth)
- Reusability: Middleware can be applied to different route groups
- Standard http.Handler interface compatibility

**Middleware Stack**:
1. Request ID (chi built-in)
2. Real IP (chi built-in)
3. Logger (chi built-in)
4. Recoverer (chi built-in)
5. Timeout (60s)
6. MaxBytesReader (10MB limit)
7. Throttle (100 concurrent requests)
8. Authentication (custom)
9. Per-Org Rate Limiting (custom, 60 req/min)

### Graceful Degradation in Dual Storage

**Pattern**: Dual storage continues operating if one backend fails

**Rationale**:
- High availability: Service continues with single backend
- Error logging: Failures are logged for operator awareness
- No transaction rollback: Append operations are idempotent
- Best-effort redundancy: Not ACID transactions

**Implementation**:
```go
func (d *DualStorage) AppendData(orgID uuid.UUID, data map[string]interface{}) error {
    err1 := d.csv.AppendData(orgID, data)
    err2 := d.mysql.AppendData(orgID, data)

    if err1 != nil && err2 != nil {
        return fmt.Errorf("both storage backends failed")
    }

    if err1 != nil {
        log.Printf("ERROR: CSV storage failed: %v", err1)
    }
    if err2 != nil {
        log.Printf("ERROR: MySQL storage failed: %v", err2)
    }

    return nil // Success if at least one succeeded
}
```

## Security Design Decisions

### Constant-Time Authentication

**Decision**: Use subtle.ConstantTimeCompare for API key validation

**Rationale**:
- Prevents timing attacks that could leak key information
- Ensures comparison time is independent of key match position
- Required for cryptographic operations per Go security guidelines

**Alternatives Considered**:
- String equality (==): Vulnerable to timing attacks
- bytes.Equal: Also vulnerable to timing attacks

### Per-Organization Rate Limiting

**Decision**: Rate limit per organization (60 req/min) rather than global

**Rationale**:
- Prevents single organization from affecting others
- Fair resource allocation across tenants
- Token bucket algorithm with 60-second window per organization
- Background cleanup of expired rate limit entries

**Alternatives Considered**:
- Global rate limiting: Unfair to well-behaved organizations
- IP-based rate limiting: Doesn't align with multi-tenant model
- No rate limiting: DoS vulnerability

### Request Validation Layers

**Decision**: Multiple validation layers before storage

**Rationale**:
1. HTTP layer: MaxBytesReader (10MB limit)
2. JSON layer: Size, depth, complexity validation
3. Semantic layer: Field format validation (alphanumeric + underscore/hyphen)
4. Business logic layer: Instance count (100 max), attribute count (100 max)

**Defense-in-Depth**: Multiple layers prevent bypass if one layer fails

## Terraform HTTP Backend Protocol

### State Backend Implementation

**Decision**: Implement Terraform HTTP backend protocol for state management

**Rationale**:
- Standard protocol: Compatible with Terraform CLI
- HTTP-based: No custom protocol implementation needed
- Locking support: Prevents concurrent state modifications
- Authentication: Uses same org ID + API key authentication

**Protocol Endpoints**:
- GET /api/v1/state/{name}: Retrieve state
- POST /api/v1/state/{name}: Update state
- DELETE /api/v1/state/{name}: Delete state
- POST /api/v1/state/{name}/lock: Acquire lock
- DELETE /api/v1/state/{name}/lock: Release lock

**Lock Implementation**:
- In-memory lock storage per organization + state name
- Lock ID must match for unlock operation
- Prevents race conditions in team collaboration

## Configuration Management

### Environment Variable-Based Configuration

**Decision**: All configuration via environment variables

**Rationale**:
- 12-factor app methodology compliance
- Container-friendly (Docker, Kubernetes)
- No configuration files to bundle in container
- Easy to override in different environments
- Explicit configuration (KISS principle)

**Configuration Variables**:
- HOST, PORT: Network binding
- STORAGE_TYPE: Storage mode selection (csv/mysql/dual/memory)
- STORAGE_PATH: CSV storage directory
- DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME: MySQL connection
- ENABLE_TLS, TLS_CERT_FILE, TLS_KEY_FILE: TLS configuration

**Default Values**: Defined in config/config.go with validation

## Testing Strategy

### Test Categories

**Decision**: Comprehensive test coverage across multiple dimensions

**Rationale**:
1. Unit tests: Package-level functionality
2. Integration tests: Cross-package interactions (auth/integration_test.go)
3. Edge case tests: Boundary conditions (auth/edge_cases_test.go)
4. Performance tests: Timing-attack resistance (auth/performance_test.go)

**Test Utilities**: auth/testutil package provides reusable helpers

### Performance Testing

**Decision**: Benchmark timing-attack resistance in authentication

**Rationale**:
- Validates constant-time comparison implementation
- Measures performance under load
- Ensures no performance regression from security measures

## Deployment Architecture

### Docker Containerization

**Decision**: Provide Dockerfile and docker-compose.yml for deployment

**Rationale**:
- Reproducible builds: Same environment everywhere
- Dependency isolation: No host-level Go installation required
- Health checks: Container orchestration integration
- Signal handling: Graceful shutdown on SIGTERM
- Volume mounts: Persistent storage for CSV and auth.cfg

**Multi-Stage Build**: Compile in Go image, run in minimal image (reduced size)

### Graceful Shutdown

**Decision**: Handle SIGINT/SIGTERM with 30-second timeout

**Rationale**:
- Complete in-flight requests before shutdown
- Prevents data loss or corruption
- Container orchestration compatibility (Kubernetes, Docker Compose)
- Cleanup resources (close database connections, file watchers)

## Documentation Strategy

### Multiple Documentation Layers

**Decision**: Provide README, QUICKSTART, deployment docs, and OpenAPI spec

**Rationale**:
- README: Overview and feature list
- QUICKSTART: Get started in 5 minutes
- DEPLOYMENT_NOTES: Production deployment guidance
- DOCKER.md: Container-specific documentation
- DUAL_STORAGE.md: Storage mode details
- openapi.yaml: Machine-readable API contract

**Audience Targeting**: Different docs for different users (developers, operators, integrators)

## Key Insights

### What Worked Well

1. **Interface-driven design**: Easy to add dual storage mode after initial CSV implementation
2. **File-based auth with hot-reload**: Operators can update credentials without restart
3. **Per-org rate limiting**: Fair resource allocation, prevents noisy neighbor problem
4. **Multiple validation layers**: Comprehensive defense against malicious inputs
5. **Standard library first**: Minimal dependencies reduce attack surface and maintenance burden

### Lessons Learned

1. **MySQL table naming**: UUIDs with hyphens require underscore conversion for SQL identifiers
2. **Graceful degradation**: Dual storage should log failures but continue operating
3. **Authentication timing**: Constant-time comparison is critical for security but requires testing
4. **Container health checks**: Essential for Docker Compose depends_on health condition
5. **Signal handling**: Proper shutdown requires context-aware server lifecycle management

## Future Considerations

While out of scope for this baseline documentation, the architecture supports these future enhancements:

1. **Additional storage backends**: PostgreSQL, S3, Azure Blob (interface already defined)
2. **Database-backed auth**: Replace file-based with MySQL/PostgreSQL credential store
3. **Metrics/monitoring**: Prometheus metrics endpoint for observability
4. **State versioning**: Track state history in database mode
5. **Multi-region**: Geographic distribution with eventual consistency
6. **Query capabilities**: Filter and search organization data beyond "get all"
7. **Audit logging**: Structured audit trail for compliance

## References

- Go HTTP server best practices: https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/
- Timing attack prevention: https://golang.org/pkg/crypto/subtle/
- Bcrypt in Go: https://pkg.go.dev/golang.org/x/crypto/bcrypt
- Chi router documentation: https://github.com/go-chi/chi
- Terraform HTTP backend protocol: https://www.terraform.io/docs/language/settings/backends/http.html
- MySQL 8.4 authentication: https://dev.mysql.com/doc/refman/8.4/en/caching-sha2-pluggable-authentication.html
- Docker multi-stage builds: https://docs.docker.com/build/building/multi-stage/
- 12-Factor App: https://12factor.net/config
