// Package store manages the global GraphStore / IDStore instances.
//
// After the entry point (cmd/hstl-oss/cmd/) calls db.InitDB() it must call
// store.Init(sqliteStore). After that, every pkg/* package accesses the
// singletons via store.Get() / store.GetIDStore().
//
// pkg/db handles DB file path resolution and InitDB(); pkg/store acts as
// a registry holding the initialised GraphStore / IDStore instances.
package store

import (
	"fmt"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
)

var (
	_store   ports.GraphStore
	_idStore ports.IDStore
)

// Init sets the global GraphStore and IDStore.
// It must be called by the entry point immediately after db.InitDB().
func Init(gs ports.GraphStore, ids ports.IDStore) {
	_store = gs
	_idStore = ids
}

// Get returns the global GraphStore.
// Returns nil rather than panicking when Init() has not been called.
func Get() ports.GraphStore {
	return _store
}

// GetIDStore returns the global IDStore.
func GetIDStore() ports.IDStore {
	return _idStore
}

// MustGet returns the GraphStore or an error when it is nil.
func MustGet() (ports.GraphStore, error) {
	if _store == nil {
		return nil, fmt.Errorf("store not initialised (call db.InitDB() + store.Init() first)")
	}
	return _store, nil
}

// MustGetIDStore returns the IDStore or an error when it is nil.
func MustGetIDStore() (ports.IDStore, error) {
	if _idStore == nil {
		return nil, fmt.Errorf("IDStore not initialised (call db.InitDB() + store.Init() first)")
	}
	return _idStore, nil
}

// Reset resets the global instances to nil (for test cleanup).
func Reset() {
	_store = nil
	_idStore = nil
}
