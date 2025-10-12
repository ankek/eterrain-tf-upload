package storage

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
)

// MemoryStorage provides an in-memory implementation of Storage
type MemoryStorage struct {
	mu     sync.RWMutex
	states map[string]*StateData // key: "orgID:name"
	locks  map[string]*LockInfo  // key: "orgID:name"
}

// NewMemoryStorage creates a new in-memory storage
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		states: make(map[string]*StateData),
		locks:  make(map[string]*LockInfo),
	}
}

func (m *MemoryStorage) stateKey(orgID uuid.UUID, name string) string {
	return fmt.Sprintf("%s:%s", orgID.String(), name)
}

// GetState retrieves state data for an organization
func (m *MemoryStorage) GetState(orgID uuid.UUID, name string) (*StateData, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.stateKey(orgID, name)
	state, exists := m.states[key]
	if !exists {
		return nil, ErrNotFound
	}

	// Return a copy to prevent external modifications
	stateCopy := *state
	dataCopy := make([]byte, len(state.Data))
	copy(dataCopy, state.Data)
	stateCopy.Data = dataCopy

	return &stateCopy, nil
}

// PutState stores state data for an organization
func (m *MemoryStorage) PutState(orgID uuid.UUID, name string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.stateKey(orgID, name)

	// Make a copy of the data
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	existing, exists := m.states[key]
	version := int64(1)
	if exists {
		version = existing.Version + 1
	}

	m.states[key] = &StateData{
		OrgID:   orgID,
		Name:    name,
		Data:    dataCopy,
		Version: version,
	}

	return nil
}

// DeleteState deletes state data for an organization
func (m *MemoryStorage) DeleteState(orgID uuid.UUID, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.stateKey(orgID, name)

	// Check if state exists
	if _, exists := m.states[key]; !exists {
		return ErrNotFound
	}

	// Check if state is locked
	if _, locked := m.locks[key]; locked {
		return ErrAlreadyLocked
	}

	delete(m.states, key)
	return nil
}

// LockState locks the state for an organization
func (m *MemoryStorage) LockState(orgID uuid.UUID, name string, lockInfo *LockInfo) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.stateKey(orgID, name)

	// Check if already locked
	if _, locked := m.locks[key]; locked {
		return ErrAlreadyLocked
	}

	// Make a copy of lock info
	lockCopy := *lockInfo
	m.locks[key] = &lockCopy

	return nil
}

// UnlockState unlocks the state for an organization
func (m *MemoryStorage) UnlockState(orgID uuid.UUID, name string, lockID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.stateKey(orgID, name)

	// Check if locked
	lock, locked := m.locks[key]
	if !locked {
		return ErrNotLocked
	}

	// Verify lock ID matches
	if lock.ID != lockID {
		return fmt.Errorf("lock ID mismatch: expected %s, got %s", lock.ID, lockID)
	}

	delete(m.locks, key)
	return nil
}

// GetLock retrieves lock information
func (m *MemoryStorage) GetLock(orgID uuid.UUID, name string) (*LockInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.stateKey(orgID, name)

	lock, exists := m.locks[key]
	if !exists {
		return nil, ErrNotFound
	}

	// Return a copy
	lockCopy := *lock
	return &lockCopy, nil
}
