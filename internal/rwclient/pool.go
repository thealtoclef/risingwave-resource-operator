/*
Copyright 2025 RisingWave Labs.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package rwclient

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

const (
	// Default connection settings.
	DefaultMaxOpenConns        = 5
	DefaultMaxIdleConns        = 2
	DefaultConnMaxLifetime     = 10 * time.Minute
	DefaultConnMaxIdleTime     = 5 * time.Minute
	DefaultHealthCheckInterval = 30 * time.Second
)

// ConnectionKey uniquely identifies a RisingWave cluster for connection pooling.
// Uses Namespace/Host:Port since this operator is standalone (no parent CR).
type ConnectionKey struct {
	Namespace string
	Host      string
	Port      int32
}

// String returns the string representation of the connection key.
func (k ConnectionKey) String() string {
	return fmt.Sprintf("%s/%s:%d", k.Namespace, k.Host, k.Port)
}

// ConnectionKeyFrom creates a ConnectionKey from namespace, host, and port.
func ConnectionKeyFrom(namespace, host string, port int32) ConnectionKey {
	return ConnectionKey{
		Namespace: namespace,
		Host:      host,
		Port:      port,
	}
}

// ConnectionInfo holds information needed to connect to a RisingWave cluster.
type ConnectionInfo struct {
	Host     string
	Port     int32
	Username string
	Password string
	Database string
	SSLMode  string
}

// DefaultConnectionInfo returns connection info with default values.
func DefaultConnectionInfo(host string, port int32, username, password string) *ConnectionInfo {
	return &ConnectionInfo{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		Database: "dev",
		SSLMode:  "disable",
	}
}

// DataSourceName returns the PostgreSQL data source name for the connection.
func (c *ConnectionInfo) DataSourceName() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.Username, c.Password, c.Database, c.SSLMode)
}

// PooledConnection wraps a database connection with metadata.
type PooledConnection struct {
	db             *sql.DB
	connectionInfo *ConnectionInfo
	lastUsed       time.Time
	mu             sync.RWMutex
}

// DB returns the underlying *sql.DB.
func (p *PooledConnection) DB() *sql.DB {
	return p.db
}

// ConnectionInfo returns the connection info.
func (p *PooledConnection) ConnectionInfo() *ConnectionInfo {
	return p.connectionInfo
}

// UpdateLastUsed updates the last used timestamp.
func (p *PooledConnection) UpdateLastUsed() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastUsed = time.Now()
}

// IsStale returns true if the connection hasn't been used for more than the specified duration.
func (p *PooledConnection) IsStale(idleTimeout time.Duration) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return time.Since(p.lastUsed) > idleTimeout
}

// Pool manages database connections to RisingWave clusters.
type Pool struct {
	connections sync.Map // ConnectionKey -> *PooledConnection

	// Configuration
	maxOpenConns        int
	maxIdleConns        int
	connMaxLifetime     time.Duration
	connMaxIdleTime     time.Duration
	idleConnectionTTL   time.Duration
	healthCheckInterval time.Duration

	// Health check
	stopHealthCheck chan struct{}
	once            sync.Once
}

// PoolConfig holds configuration for the connection pool.
type PoolConfig struct {
	MaxOpenConns        int
	MaxIdleConns        int
	ConnMaxLifetime     time.Duration
	ConnMaxIdleTime     time.Duration
	IdleConnectionTTL   time.Duration
	HealthCheckInterval time.Duration
}

// DefaultPoolConfig returns the default pool configuration.
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		MaxOpenConns:        DefaultMaxOpenConns,
		MaxIdleConns:        DefaultMaxIdleConns,
		ConnMaxLifetime:     DefaultConnMaxLifetime,
		ConnMaxIdleTime:     DefaultConnMaxIdleTime,
		IdleConnectionTTL:   10 * time.Minute,
		HealthCheckInterval: DefaultHealthCheckInterval,
	}
}

// NewPool creates a new connection pool.
func NewPool(config *PoolConfig) *Pool {
	if config == nil {
		config = DefaultPoolConfig()
	}

	pool := &Pool{
		maxOpenConns:        config.MaxOpenConns,
		maxIdleConns:        config.MaxIdleConns,
		connMaxLifetime:     config.ConnMaxLifetime,
		connMaxIdleTime:     config.ConnMaxIdleTime,
		idleConnectionTTL:   config.IdleConnectionTTL,
		healthCheckInterval: config.HealthCheckInterval,
		stopHealthCheck:     make(chan struct{}),
	}

	pool.startHealthChecker()

	return pool
}

// Get retrieves or creates a connection for the given key.
func (p *Pool) Get(ctx context.Context, key ConnectionKey, connInfo *ConnectionInfo) (*sql.DB, error) {
	// Check if connection exists
	if conn, ok := p.connections.Load(key); ok {
		pooledConn := conn.(*PooledConnection)
		pooledConn.UpdateLastUsed()

		// Verify connection is still alive
		if err := pooledConn.db.PingContext(ctx); err == nil {
			return pooledConn.db, nil
		}
		// Connection is dead, remove it
		p.connections.Delete(key)
	}

	// Create new connection
	db, err := sql.Open("postgres", connInfo.DataSourceName())
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(p.maxOpenConns)
	db.SetMaxIdleConns(p.maxIdleConns)
	db.SetConnMaxLifetime(p.connMaxLifetime)
	db.SetConnMaxIdleTime(p.connMaxIdleTime)

	// Verify connection works
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Store in pool
	pooledConn := &PooledConnection{
		db:             db,
		connectionInfo: connInfo,
		lastUsed:       time.Now(),
	}
	p.connections.Store(key, pooledConn)

	return db, nil
}

// Remove removes a connection from the pool.
func (p *Pool) Remove(key ConnectionKey) {
	if conn, ok := p.connections.Load(key); ok {
		pooledConn := conn.(*PooledConnection)
		_ = pooledConn.db.Close()
		p.connections.Delete(key)
	}
}

// Close closes all connections in the pool.
func (p *Pool) Close() {
	p.once.Do(func() {
		close(p.stopHealthCheck)

		p.connections.Range(func(key, value interface{}) bool {
			pooledConn := value.(*PooledConnection)
			_ = pooledConn.db.Close()
			p.connections.Delete(key)
			return true
		})
	})
}

// startHealthChecker starts a background goroutine to health check connections.
func (p *Pool) startHealthChecker() {
	go func() {
		ticker := time.NewTicker(p.healthCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				p.healthCheck(context.Background())
				p.cleanupStaleConnections()
			case <-p.stopHealthCheck:
				return
			}
		}
	}()
}

// healthCheck pings all connections in the pool.
func (p *Pool) healthCheck(ctx context.Context) {
	p.connections.Range(func(key, value interface{}) bool {
		pooledConn := value.(*PooledConnection)
		if err := pooledConn.db.PingContext(ctx); err != nil {
			// Connection is dead, remove it
			p.connections.Delete(key)
			_ = pooledConn.db.Close()
		}
		return true
	})
}

// cleanupStaleConnections removes connections that haven't been used recently.
func (p *Pool) cleanupStaleConnections() {
	p.connections.Range(func(key, value interface{}) bool {
		pooledConn := value.(*PooledConnection)
		if pooledConn.IsStale(p.idleConnectionTTL) {
			p.connections.Delete(key)
			_ = pooledConn.db.Close()
		}
		return true
	})
}

// Size returns the current number of connections in the pool.
func (p *Pool) Size() int {
	size := 0
	p.connections.Range(func(_, _ interface{}) bool {
		size++
		return true
	})
	return size
}
