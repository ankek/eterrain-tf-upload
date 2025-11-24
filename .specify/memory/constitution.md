<!--
  ============================================================================
  SYNC IMPACT REPORT
  ============================================================================
  Version Change: 1.0.0 → 1.1.0

  Modified Principles:
  - [NEW] VII. DRY (Don't Repeat Yourself)
  - [NEW] VIII. KISS (Keep It Simple)

  Added Sections:
  - Two new core principles (DRY and KISS)

  Removed Sections: None

  Templates Status:
  ✅ spec-template.md - Aligned with simplicity requirements
  ✅ plan-template.md - Constitution Check will include new principles
  ✅ tasks-template.md - Task structure supports DRY/KISS validation
  ✅ command files - Generic references maintained

  Follow-up TODOs: None
  ============================================================================
-->

# Terraform Backend Service Constitution

## Core Principles

### I. Security-First Development

Security is NON-NEGOTIABLE and MUST be implemented at every layer:
- **Authentication**: MUST use organization ID (UUID) and API key validation with constant-time comparison
- **Input Validation**: ALL external inputs MUST be validated before processing (JSON size, depth, complexity, field formats)
- **Rate Limiting**: Per-organization rate limiting MUST be enforced to prevent abuse (60 req/min per org)
- **Secure Storage**: API keys MUST be stored as bcrypt hashes in auth.cfg with file-watching for live updates
- **Request Limits**: Body size MUST be limited (10MB max), concurrent requests throttled (100 max)
- **Security Logging**: All authentication failures and security violations MUST be logged with org ID and IP

**Rationale**: This service handles multi-tenant infrastructure data. A security breach in one organization could expose sensitive Terraform provider data across all tenants.

### II. Dual Storage Reliability

Data persistence MUST provide redundancy and reliability:
- **Multiple Backends**: System MUST support CSV, MySQL, or dual storage modes via STORAGE_TYPE configuration
- **Graceful Degradation**: Dual storage MUST continue on single storage failure and log errors
- **Per-Organization Isolation**: Each organization MUST have isolated storage (separate CSV files, separate MySQL tables)
- **Append-Only Pattern**: Historical data MUST be preserved through append operations, not overwrites
- **Table Naming**: MySQL tables MUST use format `org_{uuid_with_underscores}` to comply with SQL naming rules

**Rationale**: Infrastructure tracking data is critical. Dual storage provides automatic backup and enables migration strategies without data loss.

### III. Interface-Driven Design

System architecture MUST use interface abstractions for flexibility:
- **Storage Interface**: Storage and DataStorage interfaces MUST define contracts for state and data operations
- **Handler Separation**: Handlers (StateHandler, UploadHandler, HealthHandler) MUST be independent and composable
- **Authentication Abstraction**: CredentialStore interface MUST support multiple implementations (InMemoryStore, FileStore)
- **Middleware Composition**: Authentication, rate limiting, and security controls MUST be composable middleware
- **No Implementation Leakage**: Handlers MUST depend on interfaces, never concrete storage implementations

**Rationale**: The service supports multiple deployment modes (CSV-only, MySQL-only, dual, memory-based state backend). Interface-driven design enables this flexibility without code duplication.

### IV. Comprehensive Testing

Testing MUST cover correctness, security, integration, and performance:
- **Test Categories**: MUST include unit tests, integration tests, edge case tests, and performance tests
- **Security Testing**: Authentication, validation, and rate limiting MUST have dedicated test suites
- **Test Helpers**: Reusable test utilities MUST be provided (testutil package) to ensure consistency
- **Performance Benchmarks**: Load testing and timing-attack resistance MUST be validated
- **Edge Case Coverage**: Boundary conditions, malformed inputs, and failure scenarios MUST be tested

**Rationale**: Multi-tenant services require rigorous testing. A bug affecting authentication or storage could impact all organizations.

### V. Defensive Validation

ALL external inputs MUST be validated with multiple layers of defense:
- **Structural Validation**: JSON size (10MB), depth (10 levels), complexity (1000 elements)
- **Semantic Validation**: Provider, category, resource_type MUST use alphanumeric + underscore/hyphen only
- **Collection Limits**: Max 100 instances per request, max 100 attributes per instance
- **Attribute Validation**: Keys and values MUST be validated for type safety and reasonable sizes
- **Pre-Storage Validation**: ALL validation MUST pass before any storage operation begins

**Rationale**: Terraform providers can send arbitrary data structures. Without strict validation, malicious or malformed data could cause storage corruption, resource exhaustion, or injection attacks.

### VI. Production-Ready Observability

Operations MUST be observable and debuggable in production:
- **Structured Logging**: All operations MUST log with prefixes (DATA:, SECURITY:, ERROR:) and include org ID, IP, timestamps
- **Health Checks**: /health endpoint MUST be unauthenticated and return service version and status
- **Graceful Shutdown**: Server MUST handle SIGINT/SIGTERM with 30-second timeout for in-flight requests
- **File Watching**: auth.cfg MUST be monitored for changes and credentials auto-reloaded (500ms debounce)
- **Error Context**: Failures MUST log sufficient context to diagnose issues without exposing sensitive data

**Rationale**: Multi-tenant backend services run 24/7 with minimal access. Comprehensive logging and graceful operations are essential for debugging and maintenance.

### VII. DRY (Don't Repeat Yourself)

Code and configuration MUST eliminate duplication through abstraction and reuse:
- **Shared Logic**: Validation, authentication, and storage patterns MUST be centralized in reusable packages
- **Configuration**: Environment variables and defaults MUST be defined once in config package, not scattered
- **Test Utilities**: Common test setup, fixtures, and assertions MUST use testutil helpers
- **Storage Implementations**: Common operations (append, read, isolation) MUST share code via interfaces
- **Error Messages**: Standard error responses MUST use shared functions, not duplicate strings
- **No Copy-Paste**: Similar code blocks MUST be refactored into functions or methods

**Rationale**: Duplication leads to inconsistent behavior, harder maintenance, and bugs when one copy is fixed but others are not. DRY ensures changes are made once and applied everywhere.

### VIII. KISS (Keep It Simple)

Simplicity MUST be preferred over cleverness or premature optimization:
- **Standard Library First**: Use Go standard library before adding dependencies (http, encoding/json, sync)
- **Linear Logic**: Avoid nested abstractions; prefer straightforward control flow over complex patterns
- **Minimal Interfaces**: Interfaces MUST have 2-5 methods; larger interfaces should be split
- **Explicit Over Implicit**: Configuration via environment variables, not conventions or "magic"
- **No Premature Abstraction**: Create abstractions when third use case appears, not earlier
- **Reject Over-Engineering**: Features like caching, queuing, or service mesh MUST justify real need

**Rationale**: Simple code is easier to understand, debug, and maintain. The service has clear requirements (auth, storage, validation) that don't require complex patterns. Complexity should only be added when solving actual problems, not hypothetical ones.

## Architecture Constraints

### Technology Stack
- **Language**: Go 1.25+ (required for modern features and security updates)
- **HTTP Framework**: chi router v5+ for routing, standard library middleware for core functionality
- **Database**: MySQL 8.4+ (when using MySQL or dual storage mode)
- **Dependencies**: Minimal external dependencies, prefer standard library where possible
- **Authentication**: File-based credential store with bcrypt hashing and fsnotify watching

### Deployment Requirements
- **Container Support**: MUST provide Dockerfile with proper health checks and signal handling
- **Docker Compose**: MUST support multi-container deployment with depends_on health checks
- **Environment Configuration**: ALL runtime configuration MUST be via environment variables (no hardcoded values)
- **Port Binding**: MUST bind to 0.0.0.0 in container mode, configurable via HOST environment variable
- **Volume Persistence**: CSV data and auth.cfg MUST be mountable volumes for persistence

### API Contracts
- **Authentication Headers**: X-Org-ID (UUID format) and X-API-Key (string token) MUST be required for all protected endpoints
- **Data Upload**: POST /api/v1/upload MUST accept hierarchical JSON with provider, category, resource_type, instances
- **Data Retrieval**: GET /api/v1/data MUST return all uploads for authenticated organization
- **State Backend**: MUST support Terraform HTTP backend protocol for /api/v1/state/{name} endpoints
- **HTTP Status Codes**: MUST use appropriate codes (401 unauthorized, 400 validation, 500 server error, 200 success)

## Development Workflow

### Code Organization
- **Project Structure**: cmd/ for entry points, internal/ for application code, tests alongside implementation
- **Package Separation**: auth/, handlers/, storage/, config/, middleware/, validation/ MUST remain independent
- **Interface Files**: storage.go MUST define interfaces before implementations
- **Configuration**: config/ MUST centralize all environment variable loading with validation

### Security Review Requirements
- **Authentication Changes**: MUST be reviewed for timing attacks and constant-time comparison
- **Validation Changes**: MUST verify all input paths are covered and limits are enforced
- **Storage Changes**: MUST verify organization isolation is maintained
- **Rate Limiting**: MUST verify per-organization tracking and proper cleanup

### Documentation Standards
- **README**: MUST include API examples, configuration options, deployment instructions, and security considerations
- **Code Comments**: Complex security logic MUST have explanatory comments
- **Deployment Docs**: MUST provide docker-compose examples and troubleshooting guides
- **Architecture Docs**: Changes to storage modes or authentication MUST update relevant .md files

### DRY and KISS Review Checklist
- **Code Review**: Flag duplicate validation logic, repeated error handling, or copy-pasted functions
- **Complexity Check**: Question nested abstractions, unused flexibility, or speculative features
- **Dependency Review**: New dependencies MUST justify why standard library is insufficient
- **Refactoring**: When fixing bugs in similar code blocks, consolidate them first

## Governance

### Constitution Authority
This constitution supersedes all other development practices. All features, changes, and reviews MUST comply with these principles.

### Amendment Process
- **Proposal**: Changes to principles MUST be documented with rationale and impact analysis
- **Review**: Amendments MUST be reviewed for conflicts with existing architecture and security model
- **Migration**: Breaking changes MUST include migration plan and backward compatibility strategy
- **Version Bump**: MAJOR for principle removal/redefinition, MINOR for additions, PATCH for clarifications

### Compliance Verification
- **Pull Requests**: MUST verify compliance with Security-First, Validation, Testing, DRY, and KISS principles
- **Code Review**: MUST check for interface violations, hardcoded values, duplication, and unnecessary complexity
- **Testing**: New features MUST include tests covering security, edge cases, and integration scenarios
- **Documentation**: Changes affecting deployment or configuration MUST update relevant documentation

### Complexity Justification
Any deviation from simplicity MUST be explicitly justified (KISS enforcement):
- **New Storage Backend**: MUST justify why existing CSV/MySQL/dual modes are insufficient
- **Additional Dependencies**: MUST justify why standard library or existing dependencies cannot solve the problem
- **New Middleware**: MUST justify why existing auth/rate-limiting middleware cannot be extended
- **Additional Interfaces**: MUST justify why existing Storage/DataStorage interfaces are insufficient
- **Abstractions**: MUST show three concrete use cases before adding new abstraction layers

**Version**: 1.1.0 | **Ratified**: 2025-11-24 | **Last Amended**: 2025-11-24
