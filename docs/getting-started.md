# Getting Started with RisingWave Resource Operator

This guide walks you through installing and using the RisingWave Resource Operator to manage your RisingWave resources including users, databases, schemas, and connections.

## Prerequisites

- A Kubernetes cluster (1.28+)
- `kubectl` configured to access your cluster
- A running RisingWave instance
- Admin credentials for RisingWave

## Installation

### Step 1: Install CRDs

```bash
make install
```

Verify all CRDs are installed:

```bash
kubectl get crd | grep risingwave.risingwavelabs.com
# risingwaveconnections.risingwave.risingwavelabs.com
# risingwavedatabases.risingwave.risingwavelabs.com
# risingwaveschemas.risingwave.risingwavelabs.com
# risingwaveusers.risingwave.risingwavelabs.com
```

### Step 2: Deploy the Operator

```bash
# Deploy with default image
make deploy

# Or with custom image
IMG=ghcr.io/risingwavelabs/risingwave-resource-operator:latest make deploy
```

Verify the operator is running:

```bash
kubectl get pods -n risingwave-resource-operator-system
```

## Quick Example

### Create a User with Table Privileges

Create a file `alice-user.yaml`:

```yaml
apiVersion: risingwave.risingwavelabs.com/v1alpha1
kind: RisingWaveUser
metadata:
  name: alice
  namespace: default
spec:
  name: alice
  connectionRef:
    host: risingwave.example.com
    port: 4567
    credentials:
      username: root
      password: ""
  password:
    generateRandomLength: 16
  grants:
    databases:
      - name: dev
        privileges:
          - CONNECT
        schemas:
          - name: public
            privileges:
              - USAGE
            tables:
              - name: users
                privileges:
                  - SELECT
                  - INSERT
```

Apply it:

```bash
kubectl apply -f alice-user.yaml
```

### Check Resource Status

```bash
kubectl get risingwaveusers
kubectl describe risingwaveuser alice
```

### Retrieve Generated Password

```bash
kubectl get secret risingwave-user-alice -o jsonpath='{.data.password}' | base64 -d
```

## Connection Reference

The `connectionRef` specifies how to connect to RisingWave:

```yaml
connectionRef:
  host: risingwave.example.com    # Required: RisingWave hostname or service
  port: 4567                      # Optional: defaults to 4567
  credentials:
    username: root                 # Optional: admin username
    password: ""                   # Admin password
    passwordSecretRef:             # Optional: reference to secret
      name: risingwave-admin
      namespace: default
      key: password
```

## Password Management

### Auto-Generated Password

The operator can generate a secure random password and store it in a Kubernetes secret:

```yaml
password:
  generateRandomLength: 16    # Length: 8-32 characters
```

The password is stored in a secret named `risingwave-user-{resource-name}`.

### External Secret

To manage the password externally:

```yaml
password:
  secretRef:
    name: my-secret
    key: password
```

### Password Rotation

Trigger password rotation by adding an annotation:

```bash
kubectl annotate risingwaveuser alice \
  risingwave.risingwavelabs.com/rotate-password="true"
```

The annotation is automatically removed after rotation completes.

## Privilege Management

### Database Privileges

```yaml
grants:
  databases:
    - name: dev
      privileges:
        - CONNECT    # Can connect to database
        - CREATE     # Can create schemas
        - ALL        # All database privileges
```

### Schema Privileges

```yaml
grants:
  databases:
    - name: dev
      privileges:
        - CONNECT
      schemas:
        - name: analytics
          privileges:
            - USAGE      # Can access schema
            - CREATE     # Can create objects in schema
            - ALL        # All schema privileges
```

### Table Privileges

```yaml
grants:
  databases:
    - name: dev
      privileges:
        - CONNECT
      schemas:
        - name: public
          tables:
            - name: users
              privileges:
                - SELECT     # Read data
                - INSERT     # Insert new rows
                - UPDATE     # Update existing rows
                - DELETE     # Delete rows
                - TRUNCATE    # Truncate table
                - REFERENCES # Create foreign keys
                - TRIGGER    # Create triggers
                - ALL        # All table privileges
```

### All Object Types

The operator supports privileges for:

| Object Type | Privileges |
|-------------|------------|
| **Database** | CONNECT, CREATE, ALL |
| **Schema** | USAGE, CREATE, ALL |
| **Table** | SELECT, INSERT, UPDATE, DELETE, TRUNCATE, REFERENCES, TRIGGER, ALL |
| **View** | SELECT, INSERT, DELETE, UPDATE, TRIGGER, ALL |
| **Materialized View** | SELECT, ALL |
| **Source** | SELECT, ALL |
| **Sink** | SELECT, ALL |
| **Connection** | USAGE, ALL |
| **Secret** | USAGE, ALL |
| **Function** | EXECUTE, ALL |

### Wildcard Privileges

Grant privileges on all objects in a schema:

```yaml
grants:
  databases:
    - name: dev
      privileges:
        - CONNECT
      schemas:
        - name: public
          tables:
            - name: "*"        # Wildcard: all tables in schema
              privileges:
                - SELECT
```

This generates: `GRANT SELECT ON ALL TABLES IN SCHEMA "public" TO "alice"`

### Multi-Database Support

The operator supports granting privileges on multiple databases:

```yaml
grants:
  databases:
    - name: dev
      privileges:
        - CONNECT
      schemas:
        - name: public
          tables:
            - name: users
              privileges:
                - SELECT
    - name: analytics
      privileges:
        - CONNECT
        - CREATE
      schemas:
        - name: reporting
          tables:
            - name: reports
              privileges:
                - SELECT
                - INSERT
```

Each database's statements are executed with proper `USE <database>` context switching.

### Privilege Reconciliation

The operator automatically:

- **Grants** new privileges specified in the spec
- **Revokes** privileges removed from the spec
- **Cleans up orphaned** privileges (objects removed from spec)

## Authentication Types

### Password Authentication (Default)

```yaml
spec:
  # No auth section or explicit password type
  name: myuser
  password:
    generateRandomLength: 16
```

### OAuth Authentication

```yaml
spec:
  name: oauth_user
  auth:
    type: oauth
    oauth:
      jwksUrl: "https://example.com/.well-known/jwks.json"
      issuer: "https://example.com"
  # No password needed for OAuth
```

### LDAP Authentication

```yaml
spec:
  name: ldap_user
  auth:
    type: ldap
    ldap:
      host: "ldap.example.com"
      port: 389
      baseDN: "dc=example,dc=com"
  # No password needed for LDAP
```

## Resource Types

The operator supports multiple CRDs for managing RisingWave resources at different scopes:

### List All CRDs

```bash
kubectl get crd | grep risingwave.risingwavelabs.com
# risingwaveconnections.risingwave.risingwavelabs.com
# risingwavedatabases.risingwave.risingwavelabs.com
# risingwaveschemas.risingwave.risingwavelabs.com
# risingwaveusers.risingwave.risingwavelabs.com
```

### RisingWaveDatabase

Database-level resource management with owner and deletion policy.

```yaml
apiVersion: risingwave.risingwavelabs.com/v1alpha1
kind: RisingWaveDatabase
metadata:
  name: analytics-db
spec:
  connectionRef:
    host: risingwave.example.com
  name: analytics             # Optional: RisingWave database name
  owner: "analytics_admin"    # Optional: Initial owner
```

### RisingWaveSchema

Schema-scoped resource management within a database.

```yaml
apiVersion: risingwave.risingwavelabs.com/v1alpha1
kind: RisingWaveSchema
metadata:
  name: reports-schema
spec:
  connectionRef:
    host: risingwave.example.com
  databaseRef:
    name: analytics           # Required: Target database name
  name: reports               # Optional: RisingWave schema name
```

### RisingWaveConnection

Reusable connection objects for sources, sinks, and tables with secret support.

```yaml
apiVersion: risingwave.risingwavelabs.com/v1alpha1
kind: RisingWaveConnection
metadata:
  name: kafka-connection
spec:
  connectionRef:
    host: risingwave.example.com
  databaseRef:
    name: analytics
  schemaRef:                  # Optional: Target schema name (defaults to "public")
    name: public
  name: kafka_prod            # Optional: Connection name in RisingWave
  type: kafka
  properties:
    properties.bootstrap.server: "kafka-broker-1:9092"
    properties.sasl.password: "SECRET kafka_credentials"  # Reference RisingWave secret
```

**Secret Reference**: Prefix value with `SECRET ` to reference a RisingWave secret.

## Annotations Reference

| Annotation | Description | Applies To |
|-----------|-------------|------------|
| `risingwave.risingwavelabs.com/pause-reconcile: "true"` | Pause reconciliation | All CRDs |
| `risingwave.risingwavelabs.com/deletion-policy: "abandon"` | Skip `DROP` on resource deletion | All CRDs |
| `risingwave.risingwavelabs.com/rotate-password: "true"` | Trigger password rotation | RisingWaveUser |

**Deletion Policy**: `abandon` (default) - Resource is retained. Use `delete` to remove it from RisingWave.

> **Note**: The `public` schema is protected from deletion. Even with `deletion-policy: "delete"`, the operator will retain the `public` schema because it is created by default in every RisingWave database.

## Status Fields

| Field | Description | Applies To |
|-------|-------------|------------|
| `.status.phase` | `Ready` or `NotReady` | All CRDs |
| `.status.reason` | Human-readable reason for current phase | All CRDs |
| `.status.conditions[]` | Detailed status conditions | All CRDs |
| `.status.observedGeneration` | Last observed generation | All CRDs |
| `.status.userCreated` | Whether the user was created | RisingWaveUser |
| `.status.privilegesSynced` | Whether privileges have been synced | RisingWaveUser |
| `.status.databaseCreated` | Whether the database exists | RisingWaveDatabase |
| `.status.schemaCreated` | Whether the schema exists | RisingWaveSchema |
| `.status.connectionCreated`| Whether the connection exists | RisingWaveConnection |

## Examples

### Example 1: Read-Only User

```yaml
apiVersion: risingwave.risingwavelabs.com/v1alpha1
kind: RisingWaveUser
metadata:
  name: readonly-user
spec:
  name: readonly_user
  connectionRef:
    host: risingwave.example.com
    port: 4567
    credentials:
      username: root
      password: ""
  password:
    generateRandomLength: 16
  grants:
    databases:
      - name: dev
        privileges:
          - CONNECT
        schemas:
          - name: public
            tables:
              - name: "*"
                privileges:
                  - SELECT
```

### Example 2: Data Analyst with Wildcard Access

```yaml
apiVersion: risingwave.risingwavelabs.com/v1alpha1
kind: RisingWaveUser
metadata:
  name: data-analyst
spec:
  name: data_analyst
  connectionRef:
    host: risingwave.example.com
    port: 4567
    credentials:
      username: root
      password: ""
  password:
    generateRandomLength: 32
  grants:
    databases:
      - name: analytics
        privileges:
          - CONNECT
        schemas:
          - name: public
            tables:
              - name: "*"
                privileges:
                  - SELECT
            views:
              - name: "*"
                privileges:
                  - SELECT
            materializedViews:
              - name: "*"
                privileges:
                  - SELECT
```

### Example 3: Application User with External Secret

```yaml
apiVersion: risingwave.risingwavelabs.com/v1alpha1
kind: RisingWaveUser
metadata:
  name: app-user
spec:
  name: app_user
  connectionRef:
    host: risingwave.example.com
    port: 4567
    credentials:
      username: root
      password: ""
  password:
    secretRef:
      name: app-user-password
      key: password
  grants:
    databases:
      - name: app_db
        privileges:
          - CONNECT
        schemas:
          - name: app_schema
            tables:
              - name: transactions
                privileges:
                  - SELECT
                  - INSERT
                  - UPDATE
              - name: events
                privileges:
                  - SELECT
                  - INSERT
```

## API Reference

### ConnectionRef (Shared)

All CRDs use a `connectionRef` to specify the target RisingWave cluster.

```yaml
connectionRef:
  host: string                 # Required: RisingWave hostname/service
  port: int32                  # Optional: defaults to 4567
  credentials:                 # Optional admin credentials
    username: string           # defaults to "root"
    passwordSecretRef:         # Recommended for admin password
      name: string
      key: string
```

### RisingWaveDatabase Spec

```yaml
spec:
  connectionRef: ConnectionRef # Required
  name: string                 # Optional: defaults to metadata.name
  owner: string                # Optional: Initial owner
```

### RisingWaveSchema Spec

```yaml
spec:
  connectionRef: ConnectionRef # Required
  databaseRef:                 # Required
    name: string
  name: string                 # Optional: defaults to metadata.name
```

### RisingWaveConnection Spec

```yaml
spec:
  connectionRef: ConnectionRef # Required
  databaseRef:                 # Required
    name: string
  schemaRef:                   # Optional: defaults to "public"
    name: string
  name: string                 # Optional: defaults to metadata.name
  type: string                 # e.g., "kafka", "iceberg"
  properties: map[string]string # Key-value pairs for WITH clause
```

### RisingWaveUser Spec

```yaml
spec:
  connectionRef: ConnectionRef # Required
  name: string                 # Optional: defaults to metadata.name
  password:                    # Optional password config
    generateRandomLength: int  # 8-32
    secretRef:                 # Or reference existing secret
      name: string
      key: string
  auth:                        # Optional auth method
    type: "password"|"oauth"|"ldap"
  grants:                      # Hierarchical privilege grants
    databases:
      - name: string
        privileges: [string]
        schemas:
          - name: string
            privileges: [string]
            tables:
              - name: string   # Use "*" for all tables
                privileges: [string]
```

## Uninstall

```bash
# 1. Delete all resources
kubectl delete risingwaveusers --all -A
kubectl delete risingwaveconnections --all -A
kubectl delete risingwaveschemas --all -A
kubectl delete risingwavedatabases --all -A

# 2. Undeploy
make undeploy && make uninstall
```

## See Also

- [Developer Guide](developer-guide.md) - For contributors
- [Local Testing Setup](local-testing-setup.md) - For local development
- [Project Readme](../README.md) - Overview and architecture

---

Last updated: 2026-03-07
