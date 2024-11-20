package db

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ConnectionManager struct {
	pools map[string]*pgxpool.Pool
	mu    sync.RWMutex
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		pools: make(map[string]*pgxpool.Pool),
	}
}

func (cm *ConnectionManager) AddPool(ctx context.Context, role, connStr string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.pools[role]; exists {
		return fmt.Errorf("pool for role %s already exists", role)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return fmt.Errorf("unable to create pool for role %s: %w", role, err)
	}

	cm.pools[role] = pool
	return nil
}

func (cm *ConnectionManager) GetPool(role string) *pgxpool.Pool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	pool, exists := cm.pools[role]
	if !exists {
		panic("Pool doesnt exist")
	}

	return pool
}

func (cm *ConnectionManager) CloseAll() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for role, pool := range cm.pools {
		if pool != nil {
			pool.Close()
			fmt.Printf("Closed pool for role: %s\n", role)
		}
	}
}
