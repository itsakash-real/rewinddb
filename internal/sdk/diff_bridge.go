package sdk

import diffpkg "github.com/itsakash-real/nimbi/internal/diff"

// DiffEngine is a thin re-export of the internal diff engine methods so that
// SDK consumers don't need to import internal/diff directly.
type DiffEngine struct {
	inner *diffpkg.Engine
}

// NewDiffEngine returns a DiffEngine bound to client's object store.
func NewDiffEngine(c *Client) *DiffEngine {
	return &DiffEngine{inner: c.diffEngine}
}

// Summary returns a one-line human-readable diff summary.
func (d *DiffEngine) Summary(result *diffpkg.DiffResult) string {
	return d.inner.Summary(result)
}

// PrettyPrint returns a colour-coded terminal diff report.
func (d *DiffEngine) PrettyPrint(result *diffpkg.DiffResult) string {
	return d.inner.PrettyPrint(result)
}
