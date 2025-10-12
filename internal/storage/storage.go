package storage

import (
	"errors"

	"github.com/google/uuid"
)

var (
	ErrNotFound      = errors.New("state not found")
	ErrAlreadyLocked = errors.New("state already locked")
	ErrNotLocked     = errors.New("state is not locked")
)

// StateData represents Terraform state data
type StateData struct {
	OrgID   uuid.UUID
	Name    string
	Data    []byte
	LockID  string
	Version int64
}

// LockInfo represents Terraform lock information
type LockInfo struct {
	ID        string
	Operation string
	Info      string
	Who       string
	Version   string
	Created   string
	Path      string
}

// Storage defines the interface for storing Terraform state
type Storage interface {
	// GetState retrieves state data for an organization
	GetState(orgID uuid.UUID, name string) (*StateData, error)

	// PutState stores state data for an organization
	PutState(orgID uuid.UUID, name string, data []byte) error

	// DeleteState deletes state data for an organization
	DeleteState(orgID uuid.UUID, name string) error

	// LockState locks the state for an organization
	LockState(orgID uuid.UUID, name string, lockInfo *LockInfo) error

	// UnlockState unlocks the state for an organization
	UnlockState(orgID uuid.UUID, name string, lockID string) error

	// GetLock retrieves lock information
	GetLock(orgID uuid.UUID, name string) (*LockInfo, error)
}
