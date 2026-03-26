package main

import (
	"fmt"
	"path/filepath"

	"github.com/itsakash-real/rewinddb/internal/timeline"
)

// depFiles maps dependency files to their install commands.
var depFiles = map[string]string{
	"package.json":      "npm install",
	"package-lock.json": "npm install",
	"go.mod":            "go mod download",
	"go.sum":            "go mod download",
	"requirements.txt":  "pip install -r requirements.txt",
	"Pipfile":           "pipenv install",
	"Cargo.toml":        "cargo build",
}

// checkDependencyChanges compares two snapshots and warns if any dependency
// manifest files changed, suggesting the appropriate install command.
func checkDependencyChanges(before, after *timeline.Snapshot) {
	beforeMap := make(map[string]string, len(before.Files))
	for _, f := range before.Files {
		beforeMap[f.Path] = f.Hash
	}

	warned := make(map[string]bool)
	for _, f := range after.Files {
		base := filepath.Base(f.Path)
		cmd, isDep := depFiles[base]
		if !isDep {
			continue
		}
		oldHash, existed := beforeMap[f.Path]
		if !existed || oldHash != f.Hash {
			if !warned[cmd] {
				warned[cmd] = true
				fmt.Printf("  %s\U0001F4E6 %s changed — run '%s' to sync dependencies%s\n",
					colorYellow, base, cmd, colorReset)
			}
		}
	}
}
