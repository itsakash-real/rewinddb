package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	diffpkg "github.com/itsakash-real/nimbi/internal/diff"
	"github.com/itsakash-real/nimbi/internal/snapshot"
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
				headLine = fmt.Sprintf("%s%s%s  ·  %s  ·  %s",
					colorPurple, branch.Name, colorReset,
					cyanP.Sprint(shortID(headCP.ID)),
					dimP.Sprint(humanTime(elapsed)),
				)
			} else {
				headLine = dimP.Sprint("no checkpoints yet")
			}

			boxTop(52)
			boxLine("  "+purpleBoldP.Sprint("◆  nimbi"), 52)
			boxLine("", 52)
			boxLine("  "+headLine, 52)
			boxBottom(52)
			fmt.Println()
			_ = hasBranch

			// ── Checkpoint counts ─────────────────────────────────────────────
			branchCPs, _ := r.engine.ListCheckpoints("")
			totalCPs := len(r.engine.Index.Checkpoints)
			kv("checkpoints", fmt.Sprintf("%d on branch  ·  %d total", len(branchCPs), totalCPs))

			// ── Storage stats ──────────────────────────────────────────────────
			objectCount, totalBytes, err := r.store.Stats()
			if err != nil {
				return fmt.Errorf("storage stats: %w", err)
			}
			kv("storage", fmt.Sprintf("%d objects  ·  %s", objectCount, formatBytes(totalBytes)))

			// ── File tracking stats ───────────────────────────────────────────
			projectRoot := parentDir(r.cfg.RewindDir)
			tracked, ignored := countTrackedIgnored(projectRoot)
			fmt.Println()
			kv("tracking", fmt.Sprintf("%s%d files%s", colorGreen, tracked, colorReset))
			if ignored > 0 {
				ignoreNames := topIgnoredDirs(projectRoot)
				kv("ignoring", fmt.Sprintf("%s%d files%s (%s)",
					colorDim, ignored, colorReset, strings.Join(ignoreNames, ", ")))
			}

			// ── Worth saving: files git doesn't track ─────────────────────────
			worthSaving := findWorthSaving(projectRoot)
			if len(worthSaving) > 0 {
				fmt.Println()
				sectionTitle("worth saving")
				fmt.Println()
				for _, ws := range worthSaving {
					fmt.Printf("  %s%s%s  %s← %s%s\n",
						colorCyan, ws.path, colorReset,
						colorDim, ws.reason, colorReset)
				}
			}

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
						fmt.Printf("\n  %s→%s  run %srw save \"message\"%s to checkpoint\n",
							colorPurpleDim, colorReset, colorBold, colorReset)
					}
				}
			}

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

// worthSavingEntry describes a file worth highlighting.
type worthSavingEntry struct {
	path   string
	reason string
}

// findWorthSaving identifies files nimbi tracks that git typically ignores.
func findWorthSaving(projectRoot string) []worthSavingEntry {
	var results []worthSavingEntry

	// Check for common git-ignored files that nimbi would track.
	envFiles := []string{".env", ".env.local", ".env.development", ".env.production", ".env.staging"}
	for _, f := range envFiles {
		if fileExistsAt(projectRoot, f) {
			if isGitIgnored(projectRoot, f) {
				results = append(results, worthSavingEntry{f, "not in git"})
			}
		}
	}

	// Config files that are commonly gitignored.
	configFiles := []string{
		"config/local.json", "config/local.yaml", "config/local.yml",
		"config/secrets.json", ".secrets", "local.settings.json",
	}
	for _, f := range configFiles {
		if fileExistsAt(projectRoot, f) {
			if isGitIgnored(projectRoot, f) {
				results = append(results, worthSavingEntry{f, "not in git"})
			}
		}
	}

	// Check for compiled binaries in common locations.
	binaryDirs := []struct {
		dir    string
		reason string
	}{
		{"build", "compiled outputs"},
		{"out", "build outputs"},
		{"bin", "compiled binaries"},
	}
	for _, bd := range binaryDirs {
		dirPath := filepath.Join(projectRoot, bd.dir)
		if info, err := os.Stat(dirPath); err == nil && info.IsDir() {
			if isGitIgnored(projectRoot, bd.dir+"/") {
				results = append(results, worthSavingEntry{bd.dir + "/", bd.reason})
			}
		}
	}

	return results
}

// isGitIgnored heuristically checks if a file is in .gitignore.
func isGitIgnored(projectRoot, relPath string) bool {
	gitignorePath := filepath.Join(projectRoot, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		return false
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Simple matching: check if the gitignore pattern matches.
		clean := strings.TrimSuffix(strings.TrimPrefix(line, "/"), "/")
		cleanRel := strings.TrimSuffix(strings.TrimPrefix(relPath, "/"), "/")
		if clean == cleanRel || strings.HasPrefix(cleanRel, clean) {
			return true
		}
		// Wildcard prefix match.
		if strings.HasPrefix(line, "*") {
			suffix := strings.TrimPrefix(line, "*")
			if strings.HasSuffix(relPath, suffix) {
				return true
			}
		}
	}
	return false
}

// countTrackedIgnored walks the project and counts tracked vs ignored files.
func countTrackedIgnored(projectRoot string) (tracked, ignored int) {
	ignores := snapshot.LoadIgnoreListPublic(projectRoot)
	_ = filepath.WalkDir(projectRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}

		rel, relErr := filepath.Rel(projectRoot, path)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)

		if ignores.Matches(rel) {
			ignored++
		} else {
			tracked++
		}
		return nil
	})
	return
}

// topIgnoredDirs returns the names of major ignored directories for display.
func topIgnoredDirs(projectRoot string) []string {
	candidates := []string{"node_modules", ".git", "dist", "vendor", "__pycache__", "target", ".next", "build", ".venv"}
	var found []string
	for _, c := range candidates {
		if info, err := os.Stat(filepath.Join(projectRoot, c)); err == nil && info.IsDir() {
			found = append(found, c)
		}
	}
	if len(found) == 0 {
		return []string{"default patterns"}
	}
	return found
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
