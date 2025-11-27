# Quickstart Guide: Terraform Backend Service

**Feature**: 001-baseline-documentation
**Date**: 2025-11-24
**Audience**: Developers, DevOps Engineers, Infrastructure Teams

## Overview

This quickstart guide helps you get the Terraform Backend Service running in minutes. The service provides two key capabilities:

1. **Data Upload API**: Accept and store infrastructure resource data from Terraform providers
2. **State Backend**: Terraform HTTP backend for state management with locking

## Prerequisites

- **Go 1.25+**: [Install Go](https://go.dev/doc/install) (or use Docker to skip this)
- **Docker & Docker Compose** (optional, for containerized deployment)
- **MySQL 8.4+** (optional, only if using MySQL or dual storage mode)

## Quick Start Options

Choose your preferred deployment method:

- [Option 1: Local Development (CSV mode)](#option-1-local-development-csv-mode) - Fastest, no database
- [Option 2: Docker Deployment (CSV mode)](#option-2-docker-deployment-csv-mode) - Containerized, no database
- [Option 3: Production Setup (Dual storage)](#option-3-production-setup-dual-storage) - MySQL + CSV redundancy

---

## Option 1: Local Development (CSV Mode)

**Time**: ~2 minutes | **Complexity**: Beginner

### 1. Clone and Build

```bash
# Clone repository
git clone <repository-url>
cd eterrain-tf-upload

# Download dependencies
go mod download

# Build the server
go build -o terraform-backend-service ./cmd/server

# Build keygen utility (for generating API keys)
go build -o keygen ./cmd/keygen
```

### 2. Generate API Key

```bash
# Generate bcrypt hash for your API key
./keygen
# Enter your desired API key when prompted
# Example output: $2a$10$abcdef123456...
```

### 3. Configure Authentication

Edit `auth.cfg` and add your organization:

```ini
[11111111-2222-3333-4444-555555555555]
apikey = $2a$10$abcdef123456...  # Paste the hash from step 2
```

### 4. Start the Service

```bash
# Create data directory
mkdir -p data

# Start server (CSV storage mode is default)
./terraform-backend-service
```

You should see:
```
Starting Terraform Backend Service v1.0.0
Using CSV storage at: ./data
Authentication credentials loaded from ./auth.cfg
Server starting on 127.0.0.1:7777
Server started successfully
```

### 5. Test the Service

```bash
# Health check (no auth required)
curl http://127.0.0.1:7777/health

# Upload sample data
curl -X POST "http://127.0.0.1:7777/api/v1/upload" \
  -H "X-Org-ID: 11111111-2222-3333-4444-555555555555" \
  -H "X-API-Key: your-actual-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "aws",
    "category": "compute",
    "resource_type": "vm_instance",
    "instances": [
      {
        "resource_name": "web-server-01",
        "attributes": {
          "size": "t2.micro",
          "region": "us-east-1",
          "status": "running"
        }
      }
    ]
  }'

# Retrieve organization data
curl "http://127.0.0.1:7777/api/v1/data" \
  -H "X-Org-ID: 11111111-2222-3333-4444-555555555555" \
  -H "X-API-Key: your-actual-api-key"

# Check CSV file
cat data/11111111-2222-3333-4444-555555555555.csv
```

**Success!** Your service is running and storing data in CSV files.

---

## Option 2: Docker Deployment (CSV Mode)

**Time**: ~3 minutes | **Complexity**: Intermediate

### 1. Clone Repository

```bash
git clone <repository-url>
cd eterrain-tf-upload
```

### 2. Configure Environment

Create `.env` file:

```bash
# Server configuration
HOST=0.0.0.0
PORT=7777
STORAGE_TYPE=csv
STORAGE_PATH=/app/data

# Optional: Enable TLS
# ENABLE_TLS=true
# TLS_CERT_FILE=/app/certs/server.crt
# TLS_KEY_FILE=/app/certs/server.key
```

### 3. Configure Authentication

Edit `auth.cfg`:

```ini
[11111111-2222-3333-4444-555555555555]
apikey = $2a$10$abcdef...  # Use keygen to generate this

# Add more organizations as needed
```

### 4. Build and Run Container

```bash
# Build Docker image
docker build -t terraform-backend-service:latest .

# Run container
docker run -d \
  --name tf-backend \
  -p 7777:7777 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/auth.cfg:/app/auth.cfg \
  --env-file .env \
  terraform-backend-service:latest

# Check logs
docker logs tf-backend

# Health check
curl http://localhost:7777/health
```

### 5. Test Data Upload

```bash
curl -X POST "http://localhost:7777/api/v1/upload" \
  -H "X-Org-ID: 11111111-2222-3333-4444-555555555555" \
  -H "X-API-Key: your-actual-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "digitalocean",
    "category": "compute",
    "resource_type": "droplet",
    "instances": [
      {
        "resource_name": "web-1",
        "attributes": {
          "size": "s-1vcpu-1gb",
          "region": "nyc1"
        }
      }
    ]
  }'
```

**Success!** Your service is running in Docker and storing data in mounted CSV directory.

---

## Option 3: Production Setup (Dual Storage)

**Time**: ~5 minutes | **Complexity**: Advanced

This setup uses both CSV and MySQL storage for redundancy with graceful degradation.

### 1. Prepare MySQL Database

```bash
# Start MySQL (using Docker Compose)
docker-compose up -d mysql

# Wait for MySQL to be ready
docker-compose exec mysql mysql -u root -prootpassword -e "CREATE DATABASE IF NOT EXISTS terraform_backend;"
```

### 2. Configure Environment

Create `.env` file:

```bash
# Server configuration
HOST=0.0.0.0
PORT=7777
STORAGE_TYPE=dual  # Use both CSV and MySQL

# CSV storage
STORAGE_PATH=/app/data

# MySQL configuration
DB_HOST=mysql
DB_PORT=3306
DB_USER=tfbackend
DB_PASSWORD=securepassword
DB_NAME=terraform_backend

# TLS (recommended for production)
ENABLE_TLS=true
TLS_CERT_FILE=/app/certs/server.crt
TLS_KEY_FILE=/app/certs/server.key
```

### 3. Generate Strong API Keys

```bash
# Generate strong random API key
openssl rand -base64 32

# Generate bcrypt hash
go run ./cmd/keygen/main.go
# Enter the random key from above
```

### 4. Configure Authentication

Edit `auth.cfg`:

```ini
# Production organization
[aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee]
apikey = $2a$10$...production_hash...

# Staging organization (separate isolation)
[11111111-2222-3333-4444-555555555555]
apikey = $2a$10$...staging_hash...
```

### 5. Deploy with Docker Compose

```bash
# Start all services (MySQL + Backend)
docker-compose up -d

# Check service health
curl http://localhost:7777/health

# View logs
docker-compose logs -f terraform-backend-service
```

### 6. Verify Dual Storage

```bash
# Upload data (will write to both CSV and MySQL)
curl -X POST "http://localhost:7777/api/v1/upload" \
  -H "X-Org-ID: aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" \
  -H "X-API-Key: your-production-key" \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "aws",
    "category": "database",
    "resource_type": "rds_instance",
    "instances": [
      {
        "resource_name": "postgres-prod",
        "attributes": {
          "engine": "postgres",
          "version": "15.4",
          "size": "db.t3.medium",
          "region": "us-east-1"
        }
      }
    ]
  }'

# Verify CSV storage
cat data/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee.csv

# Verify MySQL storage
docker-compose exec mysql mysql -u tfbackend -psecurepassword terraform_backend \
  -e "SELECT * FROM org_aaaaaaaa_bbbb_cccc_dddd_eeeeeeeeeeee;"
```

### 7. Test Graceful Degradation

```bash
# Stop MySQL
docker-compose stop mysql

# Service continues with CSV storage
curl -X POST "http://localhost:7777/api/v1/upload" \
  -H "X-Org-ID: aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" \
  -H "X-API-Key: your-production-key" \
  -H "Content-Type: application/json" \
  -d '{"provider": "test", "category": "test", "resource_type": "test", "instances": [{"resource_name": "test", "attributes": {}}]}'

# Check logs - you'll see MySQL error logged but request succeeds
docker-compose logs terraform-backend-service | grep ERROR

# Restart MySQL
docker-compose start mysql
```

**Success!** Your production service is running with dual storage redundancy.

---

## Using as Terraform State Backend

### Configure Terraform

Add to your `terraform.tf`:

```hcl
terraform {
  backend "http" {
    address        = "http://localhost:7777/api/v1/state/my-infrastructure"
    lock_address   = "http://localhost:7777/api/v1/state/my-infrastructure/lock"
    unlock_address = "http://localhost:7777/api/v1/state/my-infrastructure/lock"
    lock_method    = "POST"
    unlock_method  = "DELETE"
    username       = "11111111-2222-3333-4444-555555555555"  # Your Org ID
    password       = "your-actual-api-key"
  }
}
```

### Initialize Terraform

```bash
# Initialize backend
terraform init

# Terraform will now use your backend for state storage
terraform plan
terraform apply
```

### Test State Locking

```bash
# In terminal 1: Start long-running operation
terraform apply

# In terminal 2: Try concurrent operation (will wait for lock)
terraform plan  # Will wait until terminal 1's apply completes
```

---

## Common Configuration Options

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `HOST` | `127.0.0.1` | Server bind address (use `0.0.0.0` in Docker) |
| `PORT` | `7777` | Server port |
| `STORAGE_TYPE` | `csv` | Storage mode: `csv`, `mysql`, `dual`, `memory` |
| `STORAGE_PATH` | `./data` | CSV storage directory |
| `DB_HOST` | `localhost` | MySQL hostname |
| `DB_PORT` | `3306` | MySQL port |
| `DB_USER` | `root` | MySQL username |
| `DB_PASSWORD` | `` | MySQL password |
| `DB_NAME` | `terraform_backend` | MySQL database name |
| `ENABLE_TLS` | `false` | Enable HTTPS |
| `TLS_CERT_FILE` | `` | TLS certificate path |
| `TLS_KEY_FILE` | `` | TLS private key path |

### Storage Modes

**CSV Mode** (`STORAGE_TYPE=csv`):
- File-based storage: `data/{org-uuid}.csv`
- No database required
- Human-readable format
- Easy backup (copy files)
- Best for: Development, small deployments

**MySQL Mode** (`STORAGE_TYPE=mysql`):
- Database storage: `org_{org_uuid}` tables
- Requires MySQL 8.4+
- Indexed queries
- Scalable to millions of rows
- Best for: Production, large datasets, query requirements

**Dual Mode** (`STORAGE_TYPE=dual`):
- Writes to both CSV and MySQL
- Reads from CSV only
- Graceful degradation on single backend failure
- Automatic redundancy
- Best for: Production with high reliability requirements

**Memory Mode** (`STORAGE_TYPE=memory`):
- In-memory state storage
- For Terraform state backend only
- Fast, no persistence
- Best for: Development, temporary state

---

## Adding Organizations

### 1. Generate Organization UUID

```bash
# Use uuidgen (Linux/macOS)
uuidgen

# Or online: https://www.uuidgenerator.net/
# Or in Go: github.com/google/uuid
```

### 2. Generate API Key

```bash
# Generate strong random key
openssl rand -base64 32

# Create bcrypt hash
./keygen
# Enter the random key
```

### 3. Add to auth.cfg

```ini
[aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee]
apikey = $2a$10$...hash_for_org_A...

[11111111-2222-3333-4444-555555555555]
apikey = $2a$10$...hash_for_org_B...
```

### 4. Reload Credentials

No restart needed! The service watches `auth.cfg` and reloads automatically (500ms debounce).

```bash
# Check logs to confirm reload
docker-compose logs -f terraform-backend-service | grep "Credentials reloaded"
```

---

## Security Best Practices

### Production Checklist

- ✅ **Use strong random API keys**: `openssl rand -base64 32`
- ✅ **Enable TLS/HTTPS**: Set `ENABLE_TLS=true` with valid certificates
- ✅ **Restrict network access**: Deploy behind firewall or API gateway
- ✅ **Use bcrypt cost factor 10+**: Default is 10, can increase for keygen
- ✅ **Separate organizations**: Use different UUIDs for prod/staging/dev
- ✅ **Monitor auth failures**: Check logs for `SECURITY:` prefix
- ✅ **Regular key rotation**: Update auth.cfg with new keys periodically
- ✅ **Backup auth.cfg**: Store securely in secrets management system
- ✅ **Volume mounts**: Use persistent volumes for data and auth.cfg
- ✅ **Resource limits**: Set Docker memory/CPU limits

### TLS Configuration

```bash
# Generate self-signed certificate (development)
openssl req -x509 -newkey rsa:4096 -keyout server.key -out server.crt -days 365 -nodes

# Production: Use Let's Encrypt or your CA certificates
```

Update `.env`:
```bash
ENABLE_TLS=true
TLS_CERT_FILE=/app/certs/server.crt
TLS_KEY_FILE=/app/certs/server.key
```

Mount certificates:
```bash
docker run -d \
  -v $(pwd)/certs:/app/certs \
  -v $(pwd)/auth.cfg:/app/auth.cfg \
  ...
```

---

## Troubleshooting

### Service Won't Start

**Symptom**: Server exits immediately

**Check**:
```bash
# View logs
docker-compose logs terraform-backend-service

# Common issues:
# 1. Port already in use
sudo lsof -i :7777

# 2. Missing auth.cfg
ls -la auth.cfg

# 3. Invalid auth.cfg format
cat auth.cfg  # Check INI format

# 4. MySQL connection failed (dual/mysql mode)
docker-compose exec mysql mysqladmin ping -u root -p
```

### Authentication Fails

**Symptom**: `401 Unauthorized` responses

**Check**:
```bash
# 1. Verify credentials in auth.cfg
cat auth.cfg

# 2. Check headers format
curl -v http://localhost:7777/api/v1/data \
  -H "X-Org-ID: your-uuid" \
  -H "X-API-Key: your-key"

# 3. Verify bcrypt hash
./keygen
# Re-enter key and compare hash

# 4. Check logs for auth failures
docker-compose logs | grep "SECURITY:"
```

### Validation Errors

**Symptom**: `400 Bad Request` with validation message

**Check**:
```bash
# Common validation issues:

# 1. Missing required fields
# Must have: provider, category, resource_type, instances

# 2. Invalid field format
# provider/category/resource_type: alphanumeric + underscore/hyphen only

# 3. Too many instances
# Max 100 instances per request

# 4. Too many attributes
# Max 100 attributes per instance

# 5. JSON too large
# Max 10MB request body

# 6. JSON too deep
# Max 10 levels of nesting
```

### Dual Storage Issues

**Symptom**: MySQL errors but service continues

**Check**:
```bash
# 1. View error logs
docker-compose logs | grep "ERROR: MySQL"

# 2. Verify MySQL is running
docker-compose ps mysql

# 3. Check MySQL connection
docker-compose exec terraform-backend-service ping mysql

# 4. Verify CSV writes still work
cat data/{org-uuid}.csv

# 5. Restart MySQL
docker-compose restart mysql
```

---

## Next Steps

### Development

- Read [data-model.md](./data-model.md) for entity details
- Review [contracts/openapi.yaml](./contracts/openapi.yaml) for API specification
- Check [research.md](./research.md) for architecture decisions

### Operations

- Set up monitoring (health checks, log aggregation)
- Configure backups (CSV files, MySQL dumps, auth.cfg)
- Implement rate limiting at API gateway level
- Set up log rotation for container logs

### Integration

- Configure Terraform providers to use upload API
- Set up Terraform state backend for teams
- Create organization-specific credentials
- Document internal API usage patterns

---

## Quick Reference

### Common Commands

```bash
# Build
go build -o terraform-backend-service ./cmd/server

# Run (CSV mode)
./terraform-backend-service

# Run (MySQL mode)
export STORAGE_TYPE=mysql
export DB_HOST=localhost
export DB_USER=tfbackend
export DB_PASSWORD=password
./terraform-backend-service

# Run (Dual mode)
export STORAGE_TYPE=dual
./terraform-backend-service

# Generate API key
./keygen

# Docker build
docker build -t terraform-backend-service:latest .

# Docker run
docker run -d -p 7777:7777 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/auth.cfg:/app/auth.cfg \
  terraform-backend-service:latest

# Docker Compose
docker-compose up -d
docker-compose logs -f
docker-compose stop
docker-compose down
```

### API Examples

```bash
# Health check
curl http://localhost:7777/health

# Upload data
curl -X POST http://localhost:7777/api/v1/upload \
  -H "X-Org-ID: {uuid}" \
  -H "X-API-Key: {key}" \
  -H "Content-Type: application/json" \
  -d '{"provider": "aws", "category": "compute", "resource_type": "instance", "instances": [{"resource_name": "test", "attributes": {}}]}'

# Get organization data
curl http://localhost:7777/api/v1/data \
  -H "X-Org-ID: {uuid}" \
  -H "X-API-Key: {key}"

# Get Terraform state
curl http://localhost:7777/api/v1/state/my-state \
  -H "X-Org-ID: {uuid}" \
  -H "X-API-Key: {key}"

# Put Terraform state
curl -X POST http://localhost:7777/api/v1/state/my-state \
  -H "X-Org-ID: {uuid}" \
  -H "X-API-Key: {key}" \
  -H "Content-Type: application/json" \
  -d '{"version": 4, "terraform_version": "1.5.0", "resources": []}'

# Lock state
curl -X POST http://localhost:7777/api/v1/state/my-state/lock \
  -H "X-Org-ID: {uuid}" \
  -H "X-API-Key: {key}" \
  -H "Content-Type: application/json" \
  -d '{"ID": "lock-123", "Operation": "OperationTypeApply", "Who": "user@host", "Version": "1.5.0", "Created": "2025-11-24T10:00:00Z"}'

# Unlock state
curl -X DELETE http://localhost:7777/api/v1/state/my-state/lock \
  -H "X-Org-ID: {uuid}" \
  -H "X-API-Key: {key}" \
  -H "Content-Type: application/json" \
  -d '{"ID": "lock-123"}'
```

---

## Support

- **Documentation**: See repo docs (README.md, DEPLOYMENT_NOTES.md, DOCKER.md, DUAL_STORAGE.md)
- **API Reference**: [contracts/openapi.yaml](./contracts/openapi.yaml)
- **Issues**: Report bugs and request features via repository issues
- **Logs**: Check `docker-compose logs` or stdout for detailed error messages

---

**You're ready to use the Terraform Backend Service!** Choose your deployment option above and follow the steps to get started.
