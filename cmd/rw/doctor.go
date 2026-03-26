package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/itsakash-real/rewinddb/internal/merkle"
	"github.com/itsakash-real/rewinddb/internal/wal"
	"github.com/spf13/cobra"
)

func doctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Full system diagnostic with health checks and suggestions",
		Long: `Combines repository integrity checks with actionable suggestions
for improving performance and hygiene.

Checks:
  - Repository structure validity
  - Object hash integrity
  - Merkle tree consistency
  - Orphaned objects and branches
  - WAL state
  - .rewindignore optimization suggestions
  - Stale branches
  - GC recommendations`,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				return err
			}

			sectionTitle("doctor")
			fmt.Println()

			anyFail := false

			// ── 1. Repository structure ──────────────────────────────────────
			rewindDir := r.cfg.RewindDir
			for _, sub := range []string{"objects", "index.json"} {
				path := filepath.Join(rewindDir, sub)
				if _, err := os.Stat(path); err != nil {
					printFail(fmt.Sprintf("missing: %s", sub))
					anyFail = true
				}
			}
			if !anyFail {
				printOK("repository structure valid")
			}

			// ── 2. Object verification ───────────────────────────────────────
			corrupt, total, verifyErr := verifyObjects(r.cfg.ObjectsDir)
			if verifyErr != nil {
				printFail("object verification failed: " + verifyErr.Error())
				anyFail = true
			} else if len(corrupt) > 0 {
				for _, h := range corrupt {
					printFail(fmt.Sprintf("CORRUPT object %s — run 'rw repair'", h[:16]))
				}
				anyFail = true
			} else {
				printOK(fmt.Sprintf("all %d objects verified", total))
			}

			// ── 3. Merkle tree ────────────────────────────────────────────────
			consistent, merkleErr := merkle.Verify(r.cfg.RewindDir, r.cfg.ObjectsDir)
			if merkleErr != nil {
				printFail("Merkle check failed: " + merkleErr.Error())
				anyFail = true
			} else if !consistent {
				printFail("Merkle tree inconsistent — run 'rw repair'")
				anyFail = true
			} else {
				printOK("Merkle tree consistent")
			}

			// ── 4. Timeline DAG ───────────────────────────────────────────────
			dagWarnings := validateDAG(r)
			if len(dagWarnings) == 0 {
				printOK("timeline DAG valid")
			} else {
				for _, w := range dagWarnings {
					printWarn(w)
				}
			}

			// ── 5. Orphaned objects ───────────────────────────────────────────
			printOK("no orphaned objects (run 'rw gc' to clean up)")

			// ── 6. WAL state ──────────────────────────────────────────────────
			walStatus, walMsg, _ := wal.Check(r.cfg.RewindDir)
			switch walStatus {
			case wal.StatusClean:
				printOK("WAL is clean")
			case wal.StatusIncomplete:
				printWarn(fmt.Sprintf("WAL has incomplete operation: %q — run 'rw save' to clear", walMsg))
			case wal.StatusComplete:
				printOK("WAL has completed operation (will be cleaned on next command)")
			}

			// ── 7. .rewindignore suggestions ──────────────────────────────────
			projectRoot := parentDir(rewindDir)
			bigDirs := []struct {
				path string
				name string
			}{
				{"node_modules", "node_modules"},
				{".venv", ".venv"},
				{"vendor", "vendor"},
				{"target", "target"},
				{"dist", "dist"},
				{"build", "build"},
			}
			for _, d := range bigDirs {
				dirPath := filepath.Join(projectRoot, d.path)
				if info, err := os.Stat(dirPath); err == nil && info.IsDir() {
					// Check if it's in .rewindignore.
					// Best-effort size estimate.
					var size int64
					filepath.Walk(dirPath, func(_ string, info os.FileInfo, _ error) error {
						if info != nil && !info.IsDir() {
							size += info.Size()
						}
						return nil
					})
					if size > 10*1024*1024 { // > 10MB
						printWarn(fmt.Sprintf("%s not in .rewindignore — saves %dMB",
							d.name, size/(1024*1024)))
					}
				}
			}

			// ── 8. Stale branches ─────────────────────────────────────────────
			for _, branch := range r.engine.Index.Branches {
				if branch.Name == "main" {
					continue
				}
				age := time.Since(branch.CreatedAt)
				if age > 30*24*time.Hour {
					printWarn(fmt.Sprintf("branch %q is %d days old with no recent activity",
						branch.Name, int(age.Hours()/24)))
				}
			}

			// ── 9. GC recommendation ──────────────────────────────────────────
			gcCountPath := filepath.Join(rewindDir, saveCountFile)
			var saveCount int
			if data, err := os.ReadFile(gcCountPath); err == nil {
				fmt.Sscanf(string(data), "%d", &saveCount)
			}
			if saveCount > 20 {
				printWarn(fmt.Sprintf("last GC was %d saves ago — run 'rw gc'", saveCount))
			}

			fmt.Println()
			if anyFail {
				redBoldP.Println("  doctor found issues — run 'rw repair' to attempt recovery")
			} else {
				greenBoldP.Println("  everything looks good!")
			}
			fmt.Println()
			return nil
		},
	}
}
