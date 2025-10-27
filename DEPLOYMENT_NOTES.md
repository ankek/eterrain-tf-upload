# Deployment Notes - Dual Storage Implementation

## Issue Resolution

### Original Problem
The backend service was failing to connect to MySQL with the error:
```
Failed to initialize MySQL storage: failed to ping MySQL: dial tcp 172.18.0.2:3306: connect: connection refused
```

This was caused by a race condition where the backend tried to connect to MySQL before MySQL was fully initialized and ready to accept connections.

### Solutions Implemented

1. **Docker Compose Health Check**
   - Added `healthcheck` to MySQL service in docker-compose.yml
   - Backend now waits for MySQL to be healthy before starting
   - Health check tests MySQL connection every 5 seconds with 20 retries

2. **Connection Retry Logic**
   - Added retry logic in MySQL storage initialization (30 attempts, 1 second delay)
   - Provides additional resilience if MySQL restarts
   - Logs connection attempts for visibility

3. **Dockerfile Update**
   - Set `HOST=0.0.0.0` by default for Docker deployments
   - Fixes the original issue where service bound to `127.0.0.1` and was unreachable

## Current Status

✅ **WORKING** - Dual storage successfully operational

### Verification Results

#### 1. Service Status
```bash
$ docker compose ps
NAME                   STATUS
terraform-backend      Up (healthy)
terraform-backend-db   Up (healthy)
```

#### 2. Service Logs
```
CSV storage initialized at: /app/data
MySQL storage initialized at: db:3306/data
Using dual storage (CSV + MySQL)
Server started successfully
```

#### 3. Data Upload Test
```bash
# Test upload
Successfully uploaded 1 instance(s)

# CSV verification
✅ Data written to: /app/data/11111111-2222-3333-4444-555555555555.csv

# MySQL verification
✅ Table created: org_11111111_2222_3333_4444_555555555555
✅ Data stored with full JSON structure
```

## Deployment Instructions

### 1. Build New Image
```bash
docker build -t terraform-backend:latest .
```

### 2. Update docker-compose.yml
Ensure the following configuration:
```yaml
services:
  eterrain-tf-backend:
    image: terraform-backend:latest
    depends_on:
      db:
        condition: service_healthy
    environment:
      - STORAGE_TYPE=dual
      - DB_HOST=db
      - DB_PORT=3306
      - DB_USER=exampleuser
      - DB_PASSWORD=your_password
      - DB_NAME=data

  db:
    image: mysql:8.4
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-u", "root", "-p$$MYSQL_ROOT_PASSWORD"]
      interval: 5s
      timeout: 5s
      retries: 20
      start_period: 10s
```

### 3. Deploy
```bash
# Stop old containers
docker compose down

# Start new containers
docker compose up -d

# View logs
docker compose logs -f eterrain-tf-backend

# Verify health
docker compose ps
```

## Storage Modes

### CSV Only (`STORAGE_TYPE=csv`)
- Data stored in CSV files only
- No database required
- Good for development/testing

### MySQL Only (`STORAGE_TYPE=mysql`)
- Data stored in MySQL database only
- Requires MySQL connection
- Better query capabilities

### Dual Storage (`STORAGE_TYPE=dual`) - **RECOMMENDED**
- Data written to both CSV and MySQL
- Provides redundancy and reliability
- CSV serves as automatic file backup
- Continues on single storage failure

## Monitoring

### Check Service Health
```bash
docker compose exec eterrain-tf-backend wget -q -O- http://localhost:7777/health
```

### View Logs
```bash
# Backend logs
docker compose logs -f eterrain-tf-backend

# MySQL logs
docker compose logs -f db
```

### Verify Data Storage

#### CSV
```bash
docker compose exec eterrain-tf-backend cat /app/data/{org-id}.csv
```

#### MySQL
```bash
# List tables
docker compose exec db mysql -u exampleuser -p data -e "SHOW TABLES;"

# View data
docker compose exec db mysql -u exampleuser -p data -e "SELECT * FROM org_{org_id} LIMIT 5;"
```

## Data Upload Example

```bash
curl -X POST "http://localhost:7777/api/v1/upload" \
  -H "X-Org-ID: 11111111-2222-3333-4444-555555555555" \
  -H "X-API-Key: demo-api-key-12345" \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "aws",
    "category": "compute",
    "resource_type": "ec2_instance",
    "instances": [
      {
        "attributes": {
          "name": "web-server-01",
          "instance_type": "t2.micro",
          "region": "us-east-1"
        }
      }
    ]
  }'
```

Expected response:
```json
{
  "status": "success",
  "message": "Successfully uploaded 1 instance(s)",
  "org_id": "11111111-2222-3333-4444-555555555555",
  "instances_count": 1
}
```

## Troubleshooting

### MySQL Connection Issues
If MySQL connection fails:
1. Check health status: `docker compose ps`
2. View MySQL logs: `docker compose logs db`
3. Verify credentials in environment variables
4. Wait for health check to pass (can take 10-20 seconds)

### Data Not Appearing
1. Check backend logs for errors
2. Verify authentication headers (X-Org-ID, X-API-Key)
3. Check CSV file exists: `ls -la data/`
4. Check MySQL table exists: `docker compose exec db mysql -u exampleuser -p data -e "SHOW TABLES;"`

### Port Not Accessible
If using nginx-proxy network:
- Service is only accessible through nginx-proxy
- No direct port access from host

To expose port directly, uncomment in docker-compose.yml:
```yaml
ports:
  - "7777:7777"
```

## Security Notes

1. **Change Default Password**: Update `DB_PASSWORD` in production
2. **Secure Credentials**: Use Docker secrets or environment files
3. **Network Isolation**: MySQL not exposed outside Docker network
4. **TLS**: Enable TLS for production deployments
5. **Auth Config**: Keep auth.cfg secure with proper permissions

## Performance Considerations

- **Dual Storage**: Adds ~50-100ms latency per write
- **CSV**: Fast for small to medium datasets
- **MySQL**: Better for large datasets and complex queries
- **Connection Pool**: 25 max connections, 5 idle connections
- **Connection Lifetime**: 5 minutes

## Backup Strategy

### CSV Backup
```bash
tar -czf backup-$(date +%Y%m%d).tar.gz data/
```

### MySQL Backup
```bash
docker compose exec db mysqldump -u exampleuser -p data > backup-$(date +%Y%m%d).sql
```

## Next Steps

1. ✅ Dual storage implemented and tested
2. ✅ Docker health checks configured
3. ✅ Connection retry logic added
4. ✅ Documentation completed

Recommended future enhancements:
- Implement read replicas for MySQL
- Add data retention policies
- Implement automated backups
- Add monitoring and alerting
- Consider data archival strategy

## Support

For issues or questions, check:
1. Application logs: `docker compose logs -f eterrain-tf-backend`
2. Database logs: `docker compose logs -f db`
3. Health endpoint: http://localhost:7777/health
4. Documentation: DUAL_STORAGE.md

---
**Last Updated**: 2025-10-27
**Version**: 1.0.0
**Status**: Production Ready ✅
