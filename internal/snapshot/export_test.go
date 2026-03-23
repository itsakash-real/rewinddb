package snapshot

import "github.com/itsakash-real/rewinddb/internal/storage"

// StoreRead is a test helper that exposes ObjectStore.Read to the _test package.
func (sc *Scanner) StoreRead(hash string) ([]byte, error) {
	return sc.Store.Read(hash)
}

// ensure *ObjectStore is used so the compiler won't complain.
var _ *storage.ObjectStore = (*Scanner)(nil).Store
