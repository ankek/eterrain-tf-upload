# Terraform Backend Service

A Go-based HTTP backend service for Terraform provider data uploads with organization-based authentication. This service accepts data from Terraform providers and stores it in CSV files for historical tracking.

## Features

- **Secure Authentication**: Uses organization ID (UUID) and API key for request authentication
- **CSV Data Storage**: Stores uploaded data in organization-specific CSV files
- **Historical Tracking**: Appends all uploads to the same file with timestamps
- **RESTful API**: Clean HTTP API for data upload and retrieval
- **Health Checks**: Built-in health check endpoint for monitoring
- **Graceful Shutdown**: Proper handling of shutdown signals
- **Dual Mode**: Supports both CSV storage (data upload) and memory storage (Terraform state backend)

## Requirements

- Go 1.25 or higher
- No external dependencies for basic operation

## Installation

```bash
# Clone the repository
git clone <repository-url>
cd eterrain-tf-upload

# Download dependencies
go mod download

# Build the server
go build -o terraform-backend-service ./cmd/server

# Run the server
./terraform-backend-service
```

## Configuration

The service is configured via environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `HOST` | Server bind address | `127.0.0.1` |
| `PORT` | Server port | `7777` |
| `STORAGE_TYPE` | Storage backend type (`csv` or `memory`) | `csv` |
| `STORAGE_PATH` | Path for CSV file storage | `./data` |
| `ENABLE_TLS` | Enable HTTPS | `false` |
| `TLS_CERT_FILE` | TLS certificate file | `` |
| `TLS_KEY_FILE` | TLS key file | `` |

### Example - Data Upload Mode (CSV)

```bash
export PORT=7777
export STORAGE_TYPE=csv
export STORAGE_PATH=./data
./terraform-backend-service
```

### Example - State Backend Mode (Memory)

```bash
export PORT=8080
export STORAGE_TYPE=memory
./terraform-backend-service
```

## Authentication

The service uses header-based authentication:

- `X-Org-ID`: Organization ID (UUID format)
- `X-API-Key`: API key (string token)

### Demo Credentials

For testing, the service includes demo credentials:
- **Org ID**: `11111111-2222-3333-4444-555555555555`
- **API Key**: `demo-api-key-12345`

## API Endpoints

### Health Check

```
GET /health
```

Returns service health status (no authentication required).

**Response:**
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "service": "terraform-backend-service"
}
```

### Data Upload Operations (CSV Storage Mode)

#### Upload Data

```
POST /api/v1/upload
Headers:
  X-Org-ID: <org-uuid>
  X-API-Key: <api-key>
  Content-Type: application/json
Body: <any-json-data>
```

Uploads data from Terraform provider and appends it to the organization's CSV file.

**Example:**
```bash
curl -X POST "http://127.0.0.1:7777/api/v1/upload" \
  -H "X-Org-ID: 11111111-2222-3333-4444-555555555555" \
  -H "X-API-Key: demo-api-key-12345" \
  -H "Content-Type: application/json" \
  -d '{"resource_type": "vm_instance", "resource_name": "web-server-01", "status": "running"}'
```

**Response:**
```json
{
  "status": "success",
  "message": "Data uploaded successfully",
  "org_id": "11111111-2222-3333-4444-555555555555"
}
```

#### Get Organization Data

```
GET /api/v1/data
Headers:
  X-Org-ID: <org-uuid>
  X-API-Key: <api-key>
```

Retrieves all uploaded data for the organization.

**Response:**
```json
{
  "org_id": "11111111-2222-3333-4444-555555555555",
  "count": 2,
  "data": [
    {
      "timestamp": "2025-10-12T11:45:52Z",
      "org_id": "11111111-2222-3333-4444-555555555555",
      "data": {
        "resource_type": "vm_instance",
        "resource_name": "web-server-01",
        "status": "running"
      }
    }
  ]
}
```

### State Operations (Memory Storage Mode)

All state endpoints require authentication headers.

#### Get State

```
GET /api/v1/state/{name}
Headers:
  X-Org-ID: <org-uuid>
  X-API-Key: <api-key>
```

Returns the Terraform state data.

#### Update State

```
POST /api/v1/state/{name}
Headers:
  X-Org-ID: <org-uuid>
  X-API-Key: <api-key>
  Content-Type: application/json
Body: <terraform-state-json>
```

Updates the Terraform state.

#### Delete State

```
DELETE /api/v1/state/{name}
Headers:
  X-Org-ID: <org-uuid>
  X-API-Key: <api-key>
```

Deletes the Terraform state.

#### Lock State

```
POST /api/v1/state/{name}/lock
Headers:
  X-Org-ID: <org-uuid>
  X-API-Key: <api-key>
  Content-Type: application/json
Body:
{
  "ID": "<lock-id>",
  "Operation": "OperationTypeApply",
  "Info": "",
  "Who": "user@host",
  "Version": "1.5.0",
  "Created": "2025-10-12T10:00:00Z",
  "Path": ""
}
```

Locks the state for exclusive access.

#### Unlock State

```
DELETE /api/v1/state/{name}/lock
Headers:
  X-Org-ID: <org-uuid>
  X-API-Key: <api-key>
  Content-Type: application/json
Body:
{
  "ID": "<lock-id>"
}
```

Unlocks the state.

## Terraform Provider Configuration

For data upload service (CSV mode), configure your Terraform provider:

```hcl
provider "your_provider" {
  url    = "http://127.0.0.1:7777"
  org_id = "11111111-2222-3333-4444-555555555555"
  apikey = "demo-api-key-12345"
}
```

Your provider should make POST requests to `/api/v1/upload` endpoint with the appropriate headers.

## Terraform State Backend Configuration

To use this as a Terraform state backend (memory mode), configure your backend as follows:

```hcl
terraform {
  backend "http" {
    address        = "http://localhost:8080/api/v1/state/my-infrastructure"
    lock_address   = "http://localhost:8080/api/v1/state/my-infrastructure/lock"
    unlock_address = "http://localhost:8080/api/v1/state/my-infrastructure/lock"
    lock_method    = "POST"
    unlock_method  = "DELETE"
    username       = "00000000-0000-0000-0000-000000000001"
    password       = "demo-api-key-12345"
  }
}
```

The service will automatically map the username to `X-Org-ID` and password to `X-API-Key` headers through HTTP basic authentication.

### Alternative: Using HTTP Headers Directly

If your Terraform setup supports custom headers:

```hcl
terraform {
  backend "http" {
    address        = "http://localhost:8080/api/v1/state/my-infrastructure"
    lock_address   = "http://localhost:8080/api/v1/state/my-infrastructure/lock"
    unlock_address = "http://localhost:8080/api/v1/state/my-infrastructure/lock"
    lock_method    = "POST"
    unlock_method  = "DELETE"

    headers = {
      X-Org-ID  = "00000000-0000-0000-0000-000000000001"
      X-API-Key = "demo-api-key-12345"
    }
  }
}
```

## Testing

### Test Data Upload Service (CSV Mode)

```bash
# Health check
curl http://127.0.0.1:7777/health

# Upload data
curl -X POST "http://127.0.0.1:7777/api/v1/upload" \
     -H "X-Org-ID: 11111111-2222-3333-4444-555555555555" \
     -H "X-API-Key: demo-api-key-12345" \
     -H "Content-Type: application/json" \
     -d '{"resource_type": "vm_instance", "resource_name": "web-server-01", "status": "running"}'

# Get all data for organization
curl "http://127.0.0.1:7777/api/v1/data" \
     -H "X-Org-ID: 11111111-2222-3333-4444-555555555555" \
     -H "X-API-Key: demo-api-key-12345"

# Check CSV file
cat data/11111111-2222-3333-4444-555555555555.csv
```

### Test State Backend (Memory Mode)

First, start the service in memory mode:
```bash
export STORAGE_TYPE=memory
export PORT=8080
./terraform-backend-service
```

Then test:
```bash
# Health check
curl http://localhost:8080/health

# Get state (will return 404 if not exists)
curl -H "X-Org-ID: 11111111-2222-3333-4444-555555555555" \
     -H "X-API-Key: demo-api-key-12345" \
     http://localhost:8080/api/v1/state/test

# Create/Update state
curl -X POST \
     -H "X-Org-ID: 11111111-2222-3333-4444-555555555555" \
     -H "X-API-Key: demo-api-key-12345" \
     -H "Content-Type: application/json" \
     -d '{"version": 4, "terraform_version": "1.5.0", "serial": 1}' \
     http://localhost:8080/api/v1/state/test

# Lock state
curl -X POST \
     -H "X-Org-ID: 11111111-2222-3333-4444-555555555555" \
     -H "X-API-Key: demo-api-key-12345" \
     -H "Content-Type: application/json" \
     -d '{"ID": "test-lock-123", "Operation": "OperationTypeApply", "Who": "test@localhost"}' \
     http://localhost:8080/api/v1/state/test/lock

# Unlock state
curl -X DELETE \
     -H "X-Org-ID: 11111111-2222-3333-4444-555555555555" \
     -H "X-API-Key: demo-api-key-12345" \
     -H "Content-Type: application/json" \
     -d '{"ID": "test-lock-123"}' \
     http://localhost:8080/api/v1/state/test/lock
```

## Project Structure

```
.
├── cmd/
│   └── server/          # Main application entry point
│       └── main.go
├── internal/
│   ├── auth/            # Authentication middleware and credential management
│   │   ├── middleware.go
│   │   └── store.go
│   ├── config/          # Configuration management
│   │   └── config.go
│   ├── handlers/        # HTTP request handlers
│   │   ├── health.go
│   │   └── state.go
│   └── storage/         # State storage implementations
│       ├── storage.go
│       └── memory.go
├── go.mod
├── go.sum
└── README.md
```

## Security Considerations

1. **HTTPS**: In production, always enable TLS by setting `ENABLE_TLS=true` and providing certificate files
2. **API Keys**: Use strong, randomly generated API keys in production
3. **Credential Storage**: The in-memory credential store is for demo purposes. In production, use a secure database or secrets management system
4. **Rate Limiting**: Consider adding rate limiting middleware for production use
5. **Network Security**: Deploy behind a firewall or API gateway with proper access controls

## Future Enhancements

- [ ] Persistent storage backends (PostgreSQL, S3, etc.)
- [ ] Database-backed credential management
- [ ] Rate limiting
- [ ] Metrics and monitoring (Prometheus)
- [ ] State versioning and history
- [ ] Multi-region support
- [ ] Audit logging

## License

[Specify your license here]

## Contributing

[Contribution guidelines]
