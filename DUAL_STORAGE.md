# Dual Storage Feature (CSV + MySQL)

This document describes the dual storage feature that allows the Terraform Backend Service to store data in both CSV files and MySQL database simultaneously.

## Overview

The service now supports three storage modes for data uploads:

1. **CSV Storage** - Store data in CSV files only (original behavior)
2. **MySQL Storage** - Store data in MySQL database only
3. **Dual Storage** - Store data in both CSV and MySQL (recommended for production)

## Architecture

### Storage Modes

#### CSV Storage (`STORAGE_TYPE=csv`)
- Data stored in organization-specific CSV files
- File format: `{org_id}.csv`
- Location: Configured via `STORAGE_PATH` (default: `./data`)
- No database required

#### MySQL Storage (`STORAGE_TYPE=mysql`)
- Data stored in MySQL database
- One table per organization with name format: `org_{org_id_with_underscores}`
- Table structure: `id`, `timestamp`, `org_id`, `data` (JSON)
- Requires MySQL connection

#### Dual Storage (`STORAGE_TYPE=dual`)
- Data written to both CSV and MySQL simultaneously
- If one storage fails, data is still saved to the other
- Reads from CSV primarily, falls back to MySQL if needed
- **Recommended for production use**

## Configuration

### Environment Variables

Add the following to your `.env` file or docker-compose environment:

```bash
# Storage type selection
STORAGE_TYPE=dual  # Options: csv, mysql, dual

# CSV Storage (required for csv or dual)
STORAGE_PATH=/app/data

# MySQL Configuration (required for mysql or dual)
DB_HOST=db
DB_PORT=3306
DB_USER=exampleuser
DB_PASSWORD=your_secure_password
DB_NAME=data
```

### Docker Compose Setup

The provided `docker-compose.yml` includes:

1. **terraform-backend** service - The Go application
2. **db** service - MySQL 8.4 database
3. **db_data** volume - Persistent storage for MySQL

```bash
# Start services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down

# Stop and remove volumes
docker-compose down -v
```

## Database Schema

Each organization gets its own table with the following structure:

```sql
CREATE TABLE org_{org_id} (
    id INT AUTO_INCREMENT PRIMARY KEY,
    timestamp DATETIME(6) NOT NULL,
    org_id VARCHAR(36) NOT NULL,
    data JSON NOT NULL,
    INDEX idx_timestamp (timestamp),
    INDEX idx_org_id (org_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

### Example Table Names

- Organization ID: `11111111-2222-3333-4444-555555555555`
- Table name: `org_11111111_2222_3333_4444_555555555555`

Note: Hyphens are replaced with underscores because MySQL table names cannot contain hyphens.

## Benefits of Dual Storage

1. **Redundancy**: Data is stored in two places simultaneously
2. **Reliability**: If one storage fails, data is preserved in the other
3. **Flexibility**: Can query data from CSV files or MySQL database
4. **Migration Path**: Easy to migrate from CSV-only to MySQL-centric architecture
5. **Backup Strategy**: CSV files serve as automatic file-based backups

## Error Handling

The dual storage implementation is resilient:

- **Write Operations**:
  - Attempts to write to both storages
  - Logs errors but continues if one fails
  - Returns error only if BOTH storages fail

- **Read Operations**:
  - Reads from CSV first
  - Falls back to MySQL if CSV read fails
  - Returns error only if both fail

## Performance Considerations

### Write Performance
- Dual storage has slightly higher latency than single storage
- Writes happen sequentially: CSV first, then MySQL
- MySQL writes may be slower due to network and database overhead

### Read Performance
- CSV reads are fast for small to medium datasets
- MySQL provides better query capabilities for complex filtering
- Current implementation reads from CSV by default

## Monitoring

Check logs for storage-related messages:

```bash
# Success messages
"CSV storage initialized at: ./data"
"MySQL storage initialized at: localhost:3306/data"
"Using dual storage (CSV + MySQL)"

# Error messages (dual storage continues on single failure)
"ERROR: Failed to write to CSV storage for org {org_id}: {error}"
"ERROR: Failed to write to MySQL storage for org {org_id}: {error}"
"WARNING: Failed to read from CSV storage for org {org_id}: {error}, falling back to MySQL"
```

## Migration Guide

### From CSV to Dual Storage

1. **Update Configuration**:
   ```bash
   STORAGE_TYPE=dual
   DB_HOST=db
   DB_USER=exampleuser
   DB_PASSWORD=your_password
   DB_NAME=data
   ```

2. **Start MySQL Service**:
   ```bash
   docker-compose up -d db
   ```

3. **Restart Application**:
   ```bash
   docker-compose restart eterrain-tf-backend
   ```

4. **Verify**:
   ```bash
   docker-compose logs eterrain-tf-backend | grep "dual storage"
   ```

### From Dual to MySQL-Only

If you want to switch to MySQL-only after running dual storage:

1. Ensure all data has been written to both storages
2. Change `STORAGE_TYPE=mysql`
3. Restart the service
4. Optional: Archive CSV files for backup

## Testing

### Test CSV Storage
```bash
export STORAGE_TYPE=csv
export STORAGE_PATH=./data
./terraform-backend-service
```

### Test MySQL Storage
```bash
export STORAGE_TYPE=mysql
export DB_HOST=localhost
export DB_PORT=3306
export DB_USER=exampleuser
export DB_PASSWORD=your_password
export DB_NAME=data
./terraform-backend-service
```

### Test Dual Storage
```bash
export STORAGE_TYPE=dual
export STORAGE_PATH=./data
export DB_HOST=localhost
export DB_PORT=3306
export DB_USER=exampleuser
export DB_PASSWORD=your_password
export DB_NAME=data
./terraform-backend-service
```

### Verify Data Upload
```bash
# Upload test data
curl -X POST "http://localhost:7777/api/v1/upload" \
  -H "X-Org-ID: 11111111-2222-3333-4444-555555555555" \
  -H "X-API-Key: demo-api-key-12345" \
  -H "Content-Type: application/json" \
  -d '{"provider": "test", "category": "test", "resource_type": "test", "instances": [{"attributes": {"name": "test-1"}}]}'

# Check CSV file
cat data/11111111-2222-3333-4444-555555555555.csv

# Check MySQL database
docker-compose exec db mysql -u exampleuser -p data -e "SHOW TABLES;"
docker-compose exec db mysql -u exampleuser -p data -e "SELECT * FROM org_11111111_2222_3333_4444_555555555555 LIMIT 5;"
```

## Troubleshooting

### MySQL Connection Failed

**Error**: `Failed to initialize MySQL storage: failed to connect to MySQL`

**Solutions**:
1. Check MySQL service is running: `docker-compose ps`
2. Verify credentials in environment variables
3. Check network connectivity between containers
4. Review MySQL logs: `docker-compose logs db`

### Table Creation Failed

**Error**: `Failed to create table org_xxx: Access denied`

**Solutions**:
1. Verify user has CREATE TABLE permissions
2. Check database exists: `docker-compose exec db mysql -u root -p -e "SHOW DATABASES;"`
3. Grant permissions if needed

### Data Not Appearing in MySQL

**Possible Causes**:
1. Dual storage write to MySQL failed (check logs)
2. Table not created yet (first write creates table)
3. Wrong database selected

**Verify**:
```bash
# Check application logs
docker-compose logs eterrain-tf-backend | grep MySQL

# List tables in database
docker-compose exec db mysql -u exampleuser -p data -e "SHOW TABLES;"

# Check table contents
docker-compose exec db mysql -u exampleuser -p data -e "SELECT COUNT(*) FROM org_{your_org_id};"
```

## Security Considerations

1. **Database Credentials**: Use strong passwords and environment variables
2. **Network Isolation**: MySQL should not be exposed to the internet
3. **Access Control**: Database user should have minimal required permissions
4. **Encryption**: Enable TLS for MySQL connections in production
5. **Backup**: Regularly backup both CSV files and MySQL database

## Future Enhancements

Potential improvements for consideration:

1. Parallel writes to CSV and MySQL for better performance
2. Configurable read priority (MySQL-first vs CSV-first)
3. Data synchronization tools for CSV-to-MySQL migration
4. Query interface for complex MySQL queries
5. Data retention policies per storage type
6. Compression for CSV files
7. Read replicas for MySQL

## Support

For issues or questions:
1. Check application logs: `docker-compose logs -f eterrain-tf-backend`
2. Check MySQL logs: `docker-compose logs -f db`
3. Verify configuration in `.env` file
4. Test connectivity: `docker-compose exec eterrain-tf-backend ping db`
