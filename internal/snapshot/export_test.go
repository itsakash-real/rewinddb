package snapshot

// StoreRead is a test helper that exposes ObjectStore.Read to the _test package.
func (sc *Scanner) StoreRead(hash string) ([]byte, error) {
	return sc.Store.Read(hash)
}
