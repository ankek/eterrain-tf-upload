# Docker Deployment Guide

This guide explains how to build and run the Terraform Backend Service using Docker and docker-compose.

## Prerequisites

- Docker Engine 20.10 or higher
- Docker Compose V2 or higher
- At least 512MB of available RAM

## Quick Start

### 1. Copy Environment File

```bash
cp .env.docker .env
```

Edit `.env` to customize your deployment if needed.

### 2. Build and Start

```bash
docker-compose up -d
```

This will:
- Build the Docker image
- Start the service in detached mode
- Map port 7777 to your host machine
- Mount the `data/` directory for persistent storage

### 3. Verify Service

```bash
# Check service health
curl http://localhost:7777/health

# View logs
docker-compose logs -f terraform-backend

# Check container status
docker-compose ps
```

## Configuration

### Environment Variables

Edit `.env` file to configure the service:

```bash
# Port mapping (host:container)
HOST_PORT=7777

# Storage type: csv (data upload) or memory (state backend)
STORAGE_TYPE=csv

# TLS Configuration
ENABLE_TLS=false
TLS_CERT_FILE=
TLS_KEY_FILE=
```

### Storage Modes

#### CSV Mode (Data Upload Service)
```bash
STORAGE_TYPE=csv
HOST_PORT=7777
```

#### Memory Mode (Terraform State Backend)
```bash
STORAGE_TYPE=memory
HOST_PORT=8080
```

You can run both modes simultaneously on different ports:

```yaml
# docker-compose.override.yml
version: '3.8'
services:
  terraform-backend-csv:
    ports:
      - "7777:7777"
    environment:
      - STORAGE_TYPE=csv

  terraform-backend-memory:
    ports:
      - "8080:7777"
    environment:
      - STORAGE_TYPE=memory
```

## Authentication

The service uses credentials from `auth.cfg` file. This file is mounted into the container as read-only.

To manage credentials:

1. Edit `auth.cfg` on the host
2. Restart the container: `docker-compose restart`

The service automatically reloads credentials when the file changes.

## Data Persistence

Data is persisted through Docker volumes:

- `./data:/app/data` - CSV storage directory
- `./auth.cfg:/app/auth.cfg:ro` - Authentication configuration (read-only)

Data persists across container restarts and rebuilds.

## TLS/HTTPS Support

To enable HTTPS:

1. Place your certificate files in a `certs/` directory:
   ```bash
   mkdir -p certs
   cp your-cert.pem certs/
   cp your-key.pem certs/
   ```

2. Update `.env`:
   ```bash
   ENABLE_TLS=true
   TLS_CERT_FILE=/app/certs/your-cert.pem
   TLS_KEY_FILE=/app/certs/your-key.pem
   ```

3. Uncomment the volume mount in `docker-compose.yml`:
   ```yaml
   volumes:
     - ./certs:/app/certs:ro
   ```

4. Restart the service:
   ```bash
   docker-compose restart
   ```

## Docker Commands

### Build and Start
```bash
# Build and start in background
docker-compose up -d

# Build with no cache
docker-compose build --no-cache

# Start without rebuilding
docker-compose up -d
```

### Monitor and Debug
```bash
# View logs (live)
docker-compose logs -f

# View logs for specific service
docker-compose logs -f terraform-backend

# Check container status
docker-compose ps

# Execute shell inside container
docker-compose exec terraform-backend sh

# View resource usage
docker stats terraform-backend
```

### Stop and Remove
```bash
# Stop services
docker-compose stop

# Stop and remove containers
docker-compose down

# Remove containers and volumes
docker-compose down -v

# Remove containers, volumes, and images
docker-compose down -v --rmi all
```

### Testing
```bash
# Health check
curl http://localhost:7777/health

# Upload test data (CSV mode)
curl -X POST "http://localhost:7777/api/v1/upload" \
  -H "X-Org-ID: 11111111-2222-3333-4444-555555555555" \
  -H "X-API-Key: demo-api-key-12345" \
  -H "Content-Type: application/json" \
  -d '{"resource_type": "vm_instance", "resource_name": "web-server-01", "status": "running"}'

# Retrieve data
curl "http://localhost:7777/api/v1/data" \
  -H "X-Org-ID: 11111111-2222-3333-4444-555555555555" \
  -H "X-API-Key: demo-api-key-12345"
```

## Production Deployment

### Security Recommendations

1. **Enable TLS**: Always use HTTPS in production
2. **Change Default Credentials**: Update `auth.cfg` with strong API keys
3. **Network Isolation**: Use Docker networks to isolate services
4. **Resource Limits**: Configure CPU and memory limits in docker-compose.yml
5. **Read-Only Filesystem**: The container uses a non-root user for security

### Monitoring

The service includes a health check endpoint:

```bash
# Manual health check
docker-compose exec terraform-backend wget -q -O- http://localhost:7777/health

# Docker will automatically check health every 30 seconds
docker inspect terraform-backend | grep -A 10 Health
```

### Scaling

To run multiple instances:

```bash
docker-compose up -d --scale terraform-backend=3
```

Note: You'll need to configure a load balancer (nginx, traefik, etc.) in front of the instances.

### Backup

To backup your data:

```bash
# Backup data directory
tar -czf backup-$(date +%Y%m%d).tar.gz data/

# Backup authentication config
cp auth.cfg auth.cfg.backup
```

## Troubleshooting

### Container Won't Start

Check logs:
```bash
docker-compose logs terraform-backend
```

Common issues:
- Port already in use: Change `HOST_PORT` in `.env`
- Missing auth.cfg: Create the file before starting
- Permission issues: Ensure `data/` directory is writable

### Service Not Responding

```bash
# Check if container is running
docker-compose ps

# Check container health
docker inspect terraform-backend | grep -A 10 Health

# Test from inside container
docker-compose exec terraform-backend wget -q -O- http://localhost:7777/health
```

### Data Not Persisting

Verify volume mounts:
```bash
docker inspect terraform-backend | grep -A 20 Mounts
```

Ensure the `data/` directory exists and has proper permissions.

### High Memory Usage

Adjust resource limits in `docker-compose.yml`:

```yaml
deploy:
  resources:
    limits:
      memory: 256M
```

## Network Configuration

The service uses a bridge network by default. To integrate with existing networks:

```yaml
networks:
  default:
    external:
      name: your-existing-network
```

## Integration with Terraform

### For CSV Mode (Data Upload):
```hcl
provider "your_provider" {
  url    = "http://localhost:7777"
  org_id = "11111111-2222-3333-4444-555555555555"
  apikey = "demo-api-key-12345"
}
```

### For Memory Mode (State Backend):
```hcl
terraform {
  backend "http" {
    address        = "http://localhost:8080/api/v1/state/my-infrastructure"
    lock_address   = "http://localhost:8080/api/v1/state/my-infrastructure/lock"
    unlock_address = "http://localhost:8080/api/v1/state/my-infrastructure/lock"
    lock_method    = "POST"
    unlock_method  = "DELETE"
    username       = "11111111-2222-3333-4444-555555555555"
    password       = "demo-api-key-12345"
  }
}
```

## Additional Resources

- [Main README](README.md) - General service documentation
- [Quick Start Guide](QUICKSTART.md) - Getting started without Docker
- [OpenAPI Specification](openapi.yaml) - API documentation

## Support

For issues or questions:
1. Check container logs: `docker-compose logs -f`
2. Verify configuration in `.env` and `docker-compose.yml`
3. Test connectivity: `curl http://localhost:7777/health`
