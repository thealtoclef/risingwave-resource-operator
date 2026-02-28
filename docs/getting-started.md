# Getting Started with RisingWave Resource Operator

This guide walks you through installing and using the RisingWave Resource Operator to manage users and privileges in RisingWave.

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

Verify the CRD is installed:

```bash
kubectl get crd risingwaveusers.risingwave.risingwavelabs.com
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

## Annotations Reference

| Annotation | Value | Effect |
|-----------|-------|--------|
| `risingwave.risingwavelabs.com/pause-reconcile` | `"true"` | Pause reconciliation for this resource |
| `risingwave.risingwavelabs.com/deletion-policy` | `"abandon"` | Skip `DROP USER` on resource deletion |
| `risingwave.risingwavelabs.com/rotate-password` | `"true"` | Trigger password rotation |

## Status Fields

| Field | Description |
|-------|-------------|
| `.status.phase` | `Ready` or `NotReady` |
| `.status.reason` | Reason for current phase |
| `.status.userCreated` | Whether the user was created in RisingWave |
| `.status.privilegesSynced` | Whether privileges have been synced |
| `.status.secretName` | Name of the secret with user password |
| `.status.observedGeneration` | Last observed generation |

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

## Uninstall

```bash
# Delete all RisingWaveUser resources first
kubectl delete risingwaveusers --all --all-namespaces

# Undeploy the operator
make undeploy

# Uninstall CRDs
make uninstall
```

## Troubleshooting

### Check Operator Logs

```bash
kubectl logs -n risingwave-resource-operator-system \
  deployment/risingwave-resource-operator-controller-manager -f
```

### View Resource Events

```bash
kubectl describe risingwaveuser <name>
```

### Common Issues

| Issue | Solution |
|-------|----------|
| User not created | Verify `connectionRef` host/port is correct, check RisingWave is reachable |
| Privileges not granted | Check database/schema exists, verify `status.phase` is `Ready` |
| Secret not created | Check operator RBAC permissions, verify operator can create secrets in the namespace |
| Password rotation not working | Verify annotation is exactly `risingwave.risingwavelabs.com/rotate-password: "true"` |
| Connection refused | Verify RisingWave service is accessible from operator pod |

### Debug Connection

Test connectivity from operator pod:

```bash
kubectl run -it --rm debug --image=postgres:latest --restart=Never -- \
  psql -h risingwave.example.com -p 4567 -U root -d dev
```

## Advanced Features

### WITH GRANT OPTION

Allow users to grant their privileges to others:

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
              withGrantOption: true
```

### ALL Privilege Shorthand

Grant all available privileges:

```yaml
grants:
  databases:
    - name: dev
      privileges:
        - ALL
      schemas:
        - name: public
          tables:
            - name: users
              privileges:
                - ALL
```

### Multiple Object Types

Grant privileges on different object types in one specification:

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
                - INSERT
          views:
            - name: user_summary
              privileges:
                - SELECT
          materializedViews:
            - name: user_metrics
              privileges:
                - SELECT
          sources:
            - name: events_source
              privileges:
                - SELECT
          functions:
            - name: process_data
              privileges:
                - EXECUTE
```

## See Also

- [Developer Guide](developer-guide.md) - For contributors
- [Local Testing Setup](local-testing-setup.md) - For local development
- [Implementation Review](implementation-review.md) - Technical details

---

Last updated: 2026-02-28
