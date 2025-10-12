# Quick Start Guide

This guide will help you get the Terraform Data Upload Backend Service running in minutes.

## Prerequisites

- Go 1.25 or higher installed
- Basic familiarity with Terraform

## Step 1: Build the Service

```bash
# Build the binary
make build

# Or manually
go build -o terraform-backend-service ./cmd/server
```

## Step 2: Start the Service

```bash
# Run the service (defaults to port 7777 with CSV storage)
./terraform-backend-service
```

You should see output like:
```
2025/10/12 14:45:11 Starting Terraform Backend Service v1.0.0
2025/10/12 14:45:11 Server will listen on 127.0.0.1:7777
2025/10/12 14:45:11 Using CSV storage at: ./data
2025/10/12 14:45:11 Demo credentials added: OrgID=11111111-2222-3333-4444-555555555555, APIKey=demo-api-key-12345
2025/10/12 14:45:11 Server started successfully
2025/10/12 14:45:11 Press Ctrl+C to stop
```

## Step 3: Test the Service

Open a new terminal and test the health endpoint:

```bash
curl http://127.0.0.1:7777/health
```

Expected output:
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "service": "terraform-backend-service"
}
```

Test data upload:

```bash
curl -X POST "http://127.0.0.1:7777/api/v1/upload" \
  -H "X-Org-ID: 11111111-2222-3333-4444-555555555555" \
  -H "X-API-Key: demo-api-key-12345" \
  -H "Content-Type: application/json" \
  -d '{"resource_type": "vm_instance", "resource_name": "web-server-01", "status": "running", "region": "us-east-1"}'
```

Expected output:
```json
{
  "status": "success",
  "message": "Data uploaded successfully",
  "org_id": "11111111-2222-3333-4444-555555555555"
}
```

Check the CSV file:

```bash
cat data/11111111-2222-3333-4444-555555555555.csv
```

Retrieve all uploaded data:

```bash
curl "http://127.0.0.1:7777/api/v1/data" \
  -H "X-Org-ID: 11111111-2222-3333-4444-555555555555" \
  -H "X-API-Key: demo-api-key-12345"
```

## Step 4: Use with Terraform Provider

In your Terraform provider configuration:

```hcl
provider "your_provider" {
  url    = "http://127.0.0.1:7777"
  org_id = "11111111-2222-3333-4444-555555555555"
  apikey = "demo-api-key-12345"
}
```

The provider should send POST requests to `http://127.0.0.1:7777/api/v1/upload` with:
- Header: `X-Org-ID: 11111111-2222-3333-4444-555555555555`
- Header: `X-API-Key: demo-api-key-12345`
- Body: JSON data to be stored

## Configuration Options

You can customize the service using environment variables:

```bash
# Change the port
export PORT=8080
./terraform-backend-service

# Change storage location
export STORAGE_PATH=/var/data/terraform
./terraform-backend-service

# Use memory storage for state backend (instead of CSV)
export STORAGE_TYPE=memory
export PORT=8080
./terraform-backend-service
```

See [README.md](README.md) for complete configuration options.

## Data Storage

Data is stored in CSV files with the following format:
- Filename: `{org_id}.csv`
- Location: `./data/` (configurable via `STORAGE_PATH`)
- Format: `timestamp,org_id,data` (JSON data column)
- All uploads are appended to maintain historical order

## Troubleshooting

### Connection Refused

Make sure the service is running:
```bash
curl http://127.0.0.1:7777/health
```

### Authentication Failed

Verify you're using the correct credentials:
- Org ID: `11111111-2222-3333-4444-555555555555`
- API Key: `demo-api-key-12345`

Check the headers:
```bash
curl -v \
  -H "X-Org-ID: 11111111-2222-3333-4444-555555555555" \
  -H "X-API-Key: demo-api-key-12345" \
  http://127.0.0.1:7777/api/v1/data
```

### CSV File Not Created

Ensure the data directory exists and is writable:
```bash
mkdir -p ./data
ls -la ./data
```

## Next Steps

- Read the [README.md](README.md) for detailed API documentation
- Add your own organization credentials in `cmd/server/main.go`
- Configure your Terraform provider to upload data
- Set up proper authentication in production

## Stopping the Service

Press `Ctrl+C` in the terminal where the service is running. The service will perform a graceful shutdown.
