package main

import (
	"fmt"
	"strings"

	diffpkg "github.com/itsakash-real/rewinddb/internal/diff"
	"github.com/spf13/cobra"
)

func statusCmd() *cobra.Command {
	var verify bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current repository state and uncommitted changes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				return err
			}

			branch, hasBranch := r.engine.Index.CurrentBranch()
			headCP, hasHead := r.engine.Index.CurrentCheckpoint()

			// ── Header ────────────────────────────────────────────────────────
			fmt.Printf("%sRewindDB Status%s\n", colorBold, colorReset)
			fmt.Println(strings.Repeat("─", 40))

			if hasBranch {
				fmt.Printf("%-14s %s%s%s\n", "Branch:", colorGreen+colorBold, branch.Name, colorReset)
			} else {
				fmt.Printf("%-14s %s(none)%s\n", "Branch:", colorDim, colorReset)
			}

			if hasHead {
				fmt.Printf("%-14s %s%s%s  %q\n",
					"HEAD:",
					colorCyan, shortID(headCP.ID), colorReset,
					headCP.Message,
				)
				if len(headCP.Tags) > 0 && !(len(headCP.Tags) == 1 && headCP.Tags[0] == "root") {
					fmt.Printf("%-14s %s\n", "Tags:", strings.Join(headCP.Tags, ", "))
				}
			} else {
				fmt.Printf("%-14s %s(no checkpoint yet)%s\n", "HEAD:", colorDim, colorReset)
			}

			// ── Checkpoint counts ─────────────────────────────────────────────
			branchCPs, _ := r.engine.ListCheckpoints("")
			totalCPs := len(r.engine.Index.Checkpoints)
			fmt.Printf("%-14s %d on this branch, %d total\n", "Checkpoints:", len(branchCPs), totalCPs)

			// ── Working directory diff ─────────────────────────────────────────
			fmt.Println()
			currentSnap, err := r.scanner.Scan()
			if err != nil {
				return fmt.Errorf("scan working directory: %w", err)
			}

			if !hasHead || headCP.SnapshotRef == "" {
				fmt.Printf("%sModified since last checkpoint:%s all files (no previous snapshot)\n",
					colorBold, colorReset)
			} else {
				prevSnap, err := r.scanner.Load(headCP.SnapshotRef)
				if err != nil {
					fmt.Printf("%sModified since last checkpoint:%s (could not load previous snapshot)\n",
						colorBold, colorReset)
				} else {
					diffEng := diffpkg.New(r.store)
					result, err := diffEng.Compare(prevSnap, currentSnap)
					if err != nil {
						return fmt.Errorf("diff: %w", err)
					}

					if result.TotalChanges() == 0 {
						fmt.Printf("%sWorking directory is clean%s — nothing to save\n",
							colorGreen, colorReset)
					} else {
						fmt.Printf("%sModified since last checkpoint:%s\n", colorBold, colorReset)
						for _, f := range result.Added {
							fmt.Printf("  %s[+] %s%s\n", colorGreen, f.Path, colorReset)
						}
						for _, f := range result.Removed {
							fmt.Printf("  %s[-] %s%s\n", colorRed, f.Path, colorReset)
						}
						for _, fd := range result.Modified {
							fmt.Printf("  %s[~] %s%s\n", colorYellow, fd.Path, colorReset)
						}
						fmt.Printf("\n  → run %srw save \"message\"%s to checkpoint these changes\n",
							colorBold, colorReset)
					}
				}
			}

			// ── Storage stats ──────────────────────────────────────────────────
			fmt.Println()
			objectCount, totalBytes, err := r.store.Stats()
			if err != nil {
				return fmt.Errorf("storage stats: %w", err)
			}
			fmt.Printf("%-14s %d objects, %s\n", "Storage:", objectCount, formatBytes(totalBytes))

			// ── --verify: full object integrity check ─────────────────────────
			if verify {
				fmt.Printf("\n%s── Object integrity check ──%s\n", colorBold, colorReset)
				corrupt, checked, err := verifyAllObjects(r)
				if err != nil {
					return fmt.Errorf("verify: %w", err)
				}
				if corrupt == 0 {
					fmt.Printf("%s✓ All %d objects verified OK%s\n", colorGreen, checked, colorReset)
				} else {
					fmt.Printf("%s✗ %d/%d objects are CORRUPT%s\n",
						colorRed, corrupt, checked, colorReset)
					return fmt.Errorf("repository has %d corrupt object(s)", corrupt)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&verify, "verify", false,
		"validate SHA-256 checksums of all referenced objects")
	return cmd
}

// verifyAllObjects walks every object referenced by the current index and
// re-reads it through ObjectStore.Read (which validates the checksum).
func verifyAllObjects(r *repo) (corrupt, checked int, err error) {
	seen := make(map[string]struct{})

	for _, cp := range r.engine.Index.Checkpoints {
		if cp.SnapshotRef == "" {
			continue
		}
		// Validate the snapshot JSON object.
		if _, ok := seen[cp.SnapshotRef]; !ok {
			seen[cp.SnapshotRef] = struct{}{}
			checked++
			if _, readErr := r.store.Read(cp.SnapshotRef); readErr != nil {
				fmt.Printf("  %sCORRUPT%s  snapshot %s  (%v)\n",
					colorRed, colorReset, shortID(cp.SnapshotRef), readErr)
				corrupt++
			}
		}

		// Load snapshot to get file hashes.
		snap, loadErr := r.scanner.Load(cp.SnapshotRef)
		if loadErr != nil {
			continue
		}
		for _, fe := range snap.Files {
			if _, ok := seen[fe.Hash]; ok {
				continue
			}
			seen[fe.Hash] = struct{}{}
			checked++
			if _, readErr := r.store.Read(fe.Hash); readErr != nil {
				fmt.Printf("  %sCORRUPT%s  %s → %s  (%v)\n",
					colorRed, colorReset, fe.Path, shortID(fe.Hash), readErr)
				corrupt++
			}
		}
	}
	return corrupt, checked, nil
}

// formatBytes converts bytes to a human-readable string (B / KB / MB / GB).
func formatBytes(b int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	switch {
	case b >= GB:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
