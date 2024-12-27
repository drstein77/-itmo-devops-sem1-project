package storage

import (
	"context"
	"errors"
	"sync"

	"go.uber.org/zap"
)

// ErrConflict indicates a data conflict in the store.
var (
	ErrConflict = errors.New("data conflict")
	ErrNotFound = errors.New("not found")
)

type Log interface {
	Info(string, ...zap.Field)
}

// MemoryStorage represents an in-memory storage with locking mechanisms
type MemoryStorage struct {
	ctx context.Context
	mx  sync.RWMutex

	keeper Keeper
	log    Log
}

// Keeper interface for database operations
type Keeper interface {
	Ping(context.Context) bool
	Close() bool
}

// NewMemoryStorage creates a new MemoryStorage instance
func NewMemoryStorage(ctx context.Context, keeper Keeper, log Log) *MemoryStorage {

	if keeper != nil {
		var err error
		// Load messages

		if err != nil {
			log.Info("cannot load user data: ", zap.Error(err))
		}
	}

	return &MemoryStorage{
		ctx: ctx,

		keeper: keeper,
		log:    log,
	}
}
