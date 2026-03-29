package main

import (
	"fmt"
	"time"

	diffpkg "github.com/itsakash-real/nimbi/internal/diff"
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

			// ── Header box ────────────────────────────────────────────────────
			headLine := ""
			if hasHead {
				elapsed := int64(time.Now().Sub(headCP.CreatedAt.Local()).Seconds())
				headLine = fmt.Sprintf("%s%s%s  \u00b7  %s  \u00b7  %s",
					colorPurple, branch.Name, colorReset,
					cyanP.Sprint(shortID(headCP.ID)),
					dimP.Sprint(humanTime(elapsed)),
				)
			} else {
				headLine = dimP.Sprint("no checkpoints yet")
			}

			boxTop(52)
			boxLine("  "+purpleBoldP.Sprint("\u25c6  nimbi"), 52)
			boxLine("", 52)
			boxLine("  "+headLine, 52)
			boxBottom(52)
			fmt.Println()

			// ── Checkpoint counts ─────────────────────────────────────────────
			branchCPs, _ := r.engine.ListCheckpoints("")
			totalCPs := len(r.engine.Index.Checkpoints)
			_ = hasBranch
			kv("checkpoints", fmt.Sprintf("%d on branch  \u00b7  %d total", len(branchCPs), totalCPs))

			// ── Storage stats ──────────────────────────────────────────────────
			objectCount, totalBytes, err := r.store.Stats()
			if err != nil {
				return fmt.Errorf("storage stats: %w", err)
			}
			kv("storage", fmt.Sprintf("%d objects  \u00b7  %s", objectCount, formatBytes(totalBytes)))

			// ── Working directory diff ─────────────────────────────────────────
			fmt.Println()
			currentSnap, err := r.scanner.Scan()
			if err != nil {
				return fmt.Errorf("scan working directory: %w", err)
			}

			sectionTitle("working directory")
			fmt.Println()

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
						printSuccess("working directory is clean")
					} else {
						for _, f := range result.Added {
							fmt.Printf("  %s+%s  %s\n", colorGreen, colorReset, f.Path)
						}
						for _, f := range result.Removed {
							fmt.Printf("  %s-%s  %s\n", colorRed, colorReset, f.Path)
						}
						for _, fd := range result.Modified {
							fmt.Printf("  %s~%s  %s\n", colorYellow, colorReset, fd.Path)
						}
						fmt.Printf("\n  %s\u2192%s  run %srw save \"message\"%s to checkpoint\n",
							colorPurpleDim, colorReset, colorBold, colorReset)
					}
				}
			}

			// ── --verify: full object integrity check ─────────────────────────
			if verify {
				fmt.Printf("\n%s\u2500\u2500 Object integrity check \u2500\u2500%s\n", colorBold, colorReset)
				corrupt, checked, err := verifyAllObjects(r)
				if err != nil {
					return fmt.Errorf("verify: %w", err)
				}
				if corrupt == 0 {
					fmt.Printf("%s\u2713 All %d objects verified OK%s\n", colorGreen, checked, colorReset)
				} else {
					fmt.Printf("%s\u2717 %d/%d objects are CORRUPT%s\n",
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
				fmt.Printf("  %sCORRUPT%s  %s \u2192 %s  (%v)\n",
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
