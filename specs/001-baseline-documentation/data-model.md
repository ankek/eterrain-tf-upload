# Data Model: Terraform Backend Service

**Feature**: 001-baseline-documentation
**Date**: 2025-11-24
**Purpose**: Document the data entities, relationships, and validation rules in the existing implementation

## Overview

This document describes the data model used by the Terraform Backend Service. The service manages multiple entity types across different storage backends (CSV, MySQL, memory) with strict organization-based isolation and validation rules.

## Core Entities

### Organization

**Description**: A tenant entity representing a customer organization using the service. Each organization has isolated storage and independent rate limiting.

**Attributes**:
- `uuid` (UUID, required): Unique identifier for the organization (RFC 4122 format)

**Relationships**:
- Has many `Credential` (1:N): Organization can have multiple API keys
- Has many `ResourceUpload` (1:N): Organization owns all uploaded data
- Has many `StateFile` (1:N): Organization owns all Terraform state files
- Has many `StateLock` (1:N): Organization's state files can have locks

**Storage Implementation**:
- CSV mode: Organization UUID used as filename (`{org-uuid}.csv`)
- MySQL mode: Organization UUID used in table name (`org_{uuid_with_underscores}`)
- Memory mode: Organization UUID used as map key
- Authentication: Organization UUID stored in auth.cfg file

**Isolation**:
- Complete data isolation between organizations
- No cross-organization queries permitted
- Storage is organization-scoped (separate files/tables)

**Validation Rules**:
- UUID format: Must be valid RFC 4122 UUID (8-4-4-4-12 hex format)
- Example: `11111111-2222-3333-4444-555555555555`

---

### Credential

**Description**: API key credential associated with an organization for authentication.

**Attributes**:
- `org_id` (UUID, required): Organization UUID this credential belongs to
- `api_key_hash` (string, required): Bcrypt hash of the API key
- `created_at` (timestamp, implicit): When credential was created (file metadata)

**Relationships**:
- Belongs to `Organization` (N:1): Each credential is owned by exactly one organization

**Storage Implementation**:
- File-based: Stored in `auth.cfg` file in INI format
- Hot-reload: File watched via fsnotify with 500ms debounce
- Format: `[{org-uuid}]\napikey = $2a$10$...bcrypt_hash...`

**Authentication Flow**:
1. Client sends `X-Org-ID` (UUID) and `X-API-Key` (plaintext) headers
2. Middleware extracts headers and looks up organization in credential store
3. Bcrypt comparison: `bcrypt.CompareHashAndPassword(stored_hash, provided_key)`
4. Constant-time UUID comparison: `subtle.ConstantTimeCompare(uuid1, uuid2)`
5. On success: Organization UUID stored in request context

**Validation Rules**:
- API key: Must be non-empty string
- Org ID: Must be valid UUID
- Bcrypt hash: Must be valid bcrypt format (cost factor 10+)
- Comparison: Constant-time to prevent timing attacks

**Security Features**:
- Never stored in plaintext
- Bcrypt adaptive cost factor (default 10, configurable)
- Hot-reload on file changes (no service restart required)
- All auth failures logged with org ID and IP

---

### ResourceUpload

**Description**: A single resource instance uploaded by a Terraform provider, representing infrastructure resource state at a point in time.

**Attributes**:
- `timestamp` (RFC3339, required): When data was uploaded (server-side timestamp)
- `org_id` (UUID, required): Organization that uploaded the data
- `provider` (string, required): Terraform provider name (e.g., "aws", "azure", "gcp")
- `category` (string, required): Resource category (e.g., "compute", "network", "storage")
- `resource_type` (string, required): Specific resource type (e.g., "vm_instance", "subnet", "bucket")
- `resource_name` (string, required): Name/identifier of the resource instance
- `attributes` (JSON object, required): Key-value pairs of resource attributes

**Relationships**:
- Belongs to `Organization` (N:1): Each upload is owned by exactly one organization

**Storage Implementation**:

**CSV Mode**:
```csv
timestamp,org_id,provider,category,resource_type,resource_name,attributes
2025-11-24T10:30:00Z,11111111-2222-3333-4444-555555555555,aws,compute,vm_instance,web-server-01,"{""size"": ""t2.micro"", ""region"": ""us-east-1""}"
```

**MySQL Mode**:
```sql
CREATE TABLE IF NOT EXISTS org_11111111_2222_3333_4444_555555555555 (
    id INT AUTO_INCREMENT PRIMARY KEY,
    timestamp DATETIME NOT NULL,
    org_id VARCHAR(36) NOT NULL,
    provider VARCHAR(255) NOT NULL,
    category VARCHAR(255) NOT NULL,
    resource_type VARCHAR(255) NOT NULL,
    resource_name VARCHAR(255) NOT NULL,
    attributes JSON NOT NULL,
    INDEX idx_timestamp (timestamp),
    INDEX idx_resource_type (resource_type)
)
```

**Validation Rules**:

1. **Structural Validation**:
   - JSON payload size: Max 10MB (10485760 bytes)
   - JSON depth: Max 10 levels of nesting
   - JSON complexity: Max 1000 total elements (arrays + objects)

2. **Hierarchical Structure**:
   ```json
   {
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
   }
   ```

3. **Field Format Validation**:
   - `provider`: Alphanumeric + underscore/hyphen only (`^[a-zA-Z0-9_-]+$`)
   - `category`: Alphanumeric + underscore/hyphen only (`^[a-zA-Z0-9_-]+$`)
   - `resource_type`: Alphanumeric + underscore/hyphen only (`^[a-zA-Z0-9_-]+$`)
   - `resource_name`: Required, non-empty string
   - `attributes`: Must be valid JSON object (not array or primitive)

4. **Collection Limits**:
   - Max 100 instances per upload request
   - Max 100 attributes per instance
   - Attribute keys: Max 255 characters
   - Attribute values: Max 10000 characters (string values)

5. **Pre-Storage Validation**:
   - All validation must pass before any storage operation
   - Validation failures return 400 Bad Request with details
   - No partial writes: Either all instances stored or none

**Operations**:
- Append: Add new ResourceUpload to organization's storage
- Retrieve: Get all ResourceUploads for an organization (chronological order)
- No update or delete: Append-only historical tracking

---

### StateFile

**Description**: Terraform state data managed through the HTTP backend protocol. Contains infrastructure state as JSON blob.

**Attributes**:
- `org_id` (UUID, required): Organization that owns the state
- `name` (string, required): State name/identifier (from URL path)
- `data` ([]byte, required): Terraform state JSON blob
- `lock_id` (string, optional): Current lock ID if state is locked
- `version` (int64, required): State version number (incremented on update)

**Relationships**:
- Belongs to `Organization` (N:1): Each state is owned by exactly one organization
- Has zero or one `StateLock` (1:0..1): State can be locked during operations

**Storage Implementation**:
- Memory mode only: In-memory map keyed by org ID + state name
- Not persisted to disk (state backend is ephemeral)
- Concurrency: Protected by sync.RWMutex per state

**State Format**:
```json
{
  "version": 4,
  "terraform_version": "1.5.0",
  "serial": 1,
  "lineage": "abc123...",
  "outputs": {},
  "resources": []
}
```

**Validation Rules**:
- State name: Non-empty string, URL-safe characters
- Data: Valid JSON blob (Terraform enforces structure)
- Size: Subject to 10MB request body limit
- Version: Incremented on each successful update

**Operations**:
- Get: Retrieve current state data (200 OK or 404 Not Found)
- Put: Update state data (increment version, clear lock if lock_id matches)
- Delete: Remove state data (404 if not exists)
- Lock: Acquire exclusive lock (409 Conflict if already locked)
- Unlock: Release exclusive lock (requires matching lock_id)

**Concurrency Control**:
- State operations require lock acquisition first
- Lock prevents concurrent modifications
- Terraform CLI enforces lock protocol
- Server validates lock_id on unlock and state updates

---

### StateLock

**Description**: Exclusive lock on a Terraform state file during operations (apply, plan, etc.).

**Attributes**:
- `ID` (string, required): Unique lock identifier (client-generated UUID)
- `Operation` (string, required): Operation type (e.g., "OperationTypeApply", "OperationTypePlan")
- `Info` (string, optional): Additional information about the operation
- `Who` (string, required): User/host performing the operation (e.g., "user@hostname")
- `Version` (string, required): Terraform version (e.g., "1.5.0")
- `Created` (string, required): ISO8601 timestamp when lock was created
- `Path` (string, optional): Terraform working directory path

**Relationships**:
- Belongs to `StateFile` (N:1): Each lock is associated with exactly one state file
- Belongs to `Organization` (N:1): Indirectly through StateFile ownership

**Storage Implementation**:
- Memory mode only: Stored with StateFile in in-memory map
- Keyed by org ID + state name
- Protected by sync.RWMutex (same lock as StateFile)

**Lock Protocol**:

1. **Acquire Lock** (POST /api/v1/state/{name}/lock):
   ```json
   {
     "ID": "abc123-uuid",
     "Operation": "OperationTypeApply",
     "Who": "user@hostname",
     "Version": "1.5.0",
     "Created": "2025-11-24T10:30:00Z"
   }
   ```
   - Returns 200 OK if lock acquired
   - Returns 409 Conflict if already locked (with current lock info in body)

2. **Release Lock** (DELETE /api/v1/state/{name}/lock):
   ```json
   {
     "ID": "abc123-uuid"
   }
   ```
   - Returns 200 OK if lock released
   - Returns 400 Bad Request if lock_id doesn't match

**Validation Rules**:
- Lock ID: Non-empty string (typically UUID)
- Operation: Non-empty string
- Who: Non-empty string (user identification)
- Version: Non-empty string (Terraform version)
- Created: Valid ISO8601 timestamp

**Lifecycle**:
1. Lock acquired before state-modifying operations (apply, destroy)
2. Lock held during entire operation
3. Lock released after operation completes (success or failure)
4. Stale lock: If client crashes, manual intervention required (no timeout)

---

### DataUpload (Internal Type)

**Description**: Internal representation of uploaded data returned by storage layer.

**Attributes**:
- `Timestamp` (time.Time): When data was uploaded
- `OrgID` (uuid.UUID): Organization that uploaded the data
- `Data` (map[string]interface{}): Raw JSON data as key-value map

**Purpose**:
- Used by storage layer to return uploaded data to handlers
- Marshaled to JSON for API responses
- Not directly exposed in API contract (wrapped in response envelope)

**API Response Format**:
```json
{
  "org_id": "11111111-2222-3333-4444-555555555555",
  "count": 10,
  "data": [
    {
      "timestamp": "2025-11-24T10:30:00Z",
      "org_id": "11111111-2222-3333-4444-555555555555",
      "data": {
        "provider": "aws",
        "category": "compute",
        "resource_type": "vm_instance",
        "instances": [...]
      }
    }
  ]
}
```

---

## Entity Relationships Diagram

```
┌─────────────────┐
│  Organization   │
│  (UUID)         │
└────────┬────────┘
         │
         │ 1:N
         │
         ├─────────────────┬────────────────┬─────────────────┐
         │                 │                │                 │
         ▼                 ▼                ▼                 ▼
┌────────────────┐  ┌──────────────┐  ┌─────────────┐  ┌──────────────┐
│  Credential    │  │ ResourceUpload│  │  StateFile  │  │  StateLock   │
│  (auth.cfg)    │  │  (CSV/MySQL)  │  │  (memory)   │  │  (memory)    │
└────────────────┘  └──────────────┘  └──────┬──────┘  └──────────────┘
                                              │
                                              │ 1:0..1
                                              │
                                              ▼
                                       ┌──────────────┐
                                       │  StateLock   │
                                       │  (optional)  │
                                       └──────────────┘
```

## Storage Mode Comparison

### CSV Storage

**Use Case**: Simple file-based storage, easy backup, no database required

**Data Entities Supported**:
- ✅ ResourceUpload: Appended to organization-specific CSV file
- ❌ StateFile: Not supported (use memory mode)
- ❌ StateLock: Not supported (use memory mode)

**File Structure**:
```
data/
├── 11111111-2222-3333-4444-555555555555.csv
├── 22222222-3333-4444-5555-666666666666.csv
└── ...
```

**Advantages**:
- No database dependency
- Human-readable format
- Easy to backup (copy files)
- Simple migration (move files)

**Limitations**:
- No indexing (full file scan for queries)
- No query capabilities beyond "get all"
- Limited to append operations

---

### MySQL Storage

**Use Case**: Relational database, structured queries, organization isolation via tables

**Data Entities Supported**:
- ✅ ResourceUpload: Stored in organization-specific table
- ❌ StateFile: Not supported (use memory mode)
- ❌ StateLock: Not supported (use memory mode)

**Table Structure**:
- One table per organization: `org_{uuid_with_underscores}`
- Dynamic table creation on first upload per organization
- Indexed by timestamp and resource_type

**Advantages**:
- Fast queries with indexes
- Supports filtering and aggregation (future enhancement)
- Relational integrity constraints
- Scalable to large datasets

**Limitations**:
- Requires MySQL 8.4+ server
- Additional operational complexity
- Connection management required

---

### Dual Storage

**Use Case**: Redundancy with graceful degradation, automatic backup

**Data Entities Supported**:
- ✅ ResourceUpload: Written to both CSV and MySQL simultaneously
- ❌ StateFile: Not supported (use memory mode)
- ❌ StateLock: Not supported (use memory mode)

**Behavior**:
- Both storage backends called for each operation
- Success if at least one backend succeeds
- Failures logged but don't prevent operation
- Reads from CSV only (primary storage)

**Advantages**:
- Automatic redundancy
- Graceful degradation on single backend failure
- Backup without additional tooling
- Migration path between storage modes

**Limitations**:
- Writes to both (performance impact)
- Eventual consistency (not ACID transactions)
- Reads favor CSV (MySQL used as backup only)

---

### Memory Storage

**Use Case**: Terraform HTTP state backend, fast ephemeral storage

**Data Entities Supported**:
- ❌ ResourceUpload: Not supported (use CSV or MySQL)
- ✅ StateFile: Stored in-memory map
- ✅ StateLock: Stored with StateFile

**Structure**:
```go
map[string]*StateData{
    "org-uuid:state-name": &StateData{
        OrgID: uuid.UUID,
        Name: "state-name",
        Data: []byte("...json..."),
        LockID: "lock-uuid",
        Version: 3,
    },
}
```

**Advantages**:
- Fast (no disk I/O)
- Simple implementation (no persistence layer)
- Suitable for temporary state storage

**Limitations**:
- Data lost on service restart
- No persistence or backup
- Memory usage grows with state count
- Not recommended for production Terraform state

---

## Validation Rules Summary

### Request-Level Validation

| Rule | Limit | Rationale |
|------|-------|-----------|
| Request body size | 10MB | Prevent DoS attacks via oversized payloads |
| JSON depth | 10 levels | Prevent stack overflow from deeply nested structures |
| JSON complexity | 1000 elements | Prevent resource exhaustion from large arrays/objects |
| Concurrent requests | 100 | Prevent resource exhaustion |
| Rate limit per org | 60 req/min | Fair resource allocation, prevent noisy neighbor |

### Field-Level Validation

| Field | Pattern | Max Length | Required |
|-------|---------|------------|----------|
| Organization UUID | RFC 4122 UUID | 36 chars | ✅ |
| Provider | `^[a-zA-Z0-9_-]+$` | 255 chars | ✅ |
| Category | `^[a-zA-Z0-9_-]+$` | 255 chars | ✅ |
| Resource Type | `^[a-zA-Z0-9_-]+$` | 255 chars | ✅ |
| Resource Name | Non-empty string | 255 chars | ✅ |
| Attribute Key | Non-empty string | 255 chars | ✅ |
| Attribute Value | Any JSON type | 10000 chars (strings) | ✅ |
| State Name | URL-safe string | 255 chars | ✅ |
| Lock ID | Non-empty string | 255 chars | ✅ |

### Collection-Level Validation

| Collection | Limit | Rationale |
|------------|-------|-----------|
| Instances per upload | 100 | Prevent oversized single requests |
| Attributes per instance | 100 | Prevent resource exhaustion from large objects |
| States per organization | Unlimited | No artificial limit (memory-based) |
| Uploads per organization | Unlimited | Historical tracking (append-only) |

---

## Data Lifecycle

### ResourceUpload Lifecycle

1. **Creation**:
   - Client sends POST /api/v1/upload with JSON payload
   - Authentication middleware validates org ID + API key
   - Rate limiting middleware checks request quota
   - Validation middleware checks JSON size/depth/complexity
   - Handler validates field formats and collection limits
   - Storage layer appends to organization's storage (CSV/MySQL/dual)
   - Server returns 200 OK with success message

2. **Storage**:
   - CSV: Appended to `{org-uuid}.csv` file
   - MySQL: Inserted into `org_{uuid}` table
   - Dual: Written to both CSV and MySQL

3. **Retrieval**:
   - Client sends GET /api/v1/data
   - Authentication validates org ID + API key
   - Storage layer reads all records for organization
   - Handler returns records in chronological order

4. **Retention**:
   - No automatic deletion or archival
   - Unlimited historical retention
   - Operator responsible for backup and cleanup

### StateFile Lifecycle

1. **Creation**:
   - Client sends POST /api/v1/state/{name}
   - Authentication validates org ID + API key
   - Handler creates state entry in memory map
   - Version initialized to 0

2. **Update**:
   - Client must acquire lock first (POST /api/v1/state/{name}/lock)
   - Client sends POST /api/v1/state/{name} with state JSON
   - Handler increments version
   - Lock cleared if lock_id matches request

3. **Locking**:
   - Client sends POST /api/v1/state/{name}/lock with lock info
   - Handler checks if state already locked
   - Returns 409 Conflict if locked, 200 OK if lock acquired
   - Lock persists until explicitly released

4. **Unlocking**:
   - Client sends DELETE /api/v1/state/{name}/lock with lock ID
   - Handler validates lock ID matches current lock
   - Lock cleared on success

5. **Deletion**:
   - Client sends DELETE /api/v1/state/{name}
   - Handler removes state from memory map
   - Returns 200 OK (or 404 if not exists)

6. **Data Loss**:
   - All state data lost on service restart (memory-only)
   - No persistence or backup mechanism
   - Clients must handle re-initialization

---

## Security Considerations

### Data Isolation

- **Organization Boundary**: All entities scoped by organization UUID
- **No Cross-Org Queries**: Authentication middleware enforces org context
- **Storage Isolation**: Separate files/tables per organization
- **Lock Isolation**: State locks are per-organization + state name

### Authentication Data

- **API Keys**: Never stored in plaintext, bcrypt hashed in auth.cfg
- **Constant-Time Comparison**: Prevents timing attacks on key validation
- **Hot-Reload**: Credential changes applied without service restart
- **Failed Auth Logging**: All failures logged with org ID and IP

### Input Validation

- **Defense-in-Depth**: Multiple validation layers
- **Pre-Storage Validation**: All checks pass before any storage operation
- **Error Messages**: Informative but don't leak sensitive data
- **Rate Limiting**: Per-organization to prevent abuse

### Data Integrity

- **Append-Only**: ResourceUploads never updated or deleted (historical integrity)
- **Version Tracking**: StateFiles have version numbers
- **Lock Protocol**: Prevents concurrent state modifications
- **Dual Storage**: Provides redundancy (but not ACID)

---

## Performance Characteristics

### CSV Storage

- **Write**: O(1) append to file
- **Read**: O(n) full file scan
- **Scalability**: Limited by file size (OS limits)
- **Indexing**: None (sequential access only)

### MySQL Storage

- **Write**: O(log n) with B-tree index
- **Read**: O(log n) with index, O(n) full table scan
- **Scalability**: Scales to millions of rows per table
- **Indexing**: timestamp, resource_type indexes

### Memory Storage

- **Write**: O(1) map insertion
- **Read**: O(1) map lookup
- **Scalability**: Limited by RAM
- **Concurrency**: Protected by RWMutex per state

---

## Migration Paths

### CSV to MySQL

1. Read all CSV files
2. Create MySQL tables per organization
3. Insert all records with original timestamps
4. Validate record counts match
5. Update configuration to MySQL mode

### CSV to Dual

1. Configure MySQL connection
2. Update STORAGE_TYPE to "dual"
3. Service creates MySQL tables on first write
4. Existing CSV data remains, new writes go to both

### Dual to MySQL

1. Verify MySQL has all data
2. Update STORAGE_TYPE to "mysql"
3. Stop writing to CSV
4. Archive CSV files for backup

---

## Future Enhancements

While out of scope for this baseline, the data model supports:

1. **Query Capabilities**: Add filters to GET /api/v1/data (by resource_type, date range, etc.)
2. **Pagination**: Return large result sets in pages
3. **State Persistence**: Add MySQL/PostgreSQL backend for StateFile
4. **State History**: Track all state versions with rollback capability
5. **Audit Trail**: Comprehensive logging of all data operations
6. **Data Retention Policies**: Automatic archival and cleanup
7. **Multi-Region**: Geographic distribution with replication
8. **Search Indexing**: Full-text search on resource attributes
