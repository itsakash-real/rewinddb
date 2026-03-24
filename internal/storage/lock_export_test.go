package storage

// Path exposes the lock file path for tests.
func (fl *FileLock) Path() string { return fl.path }
