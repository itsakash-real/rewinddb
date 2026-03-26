package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// bundleMeta is the metadata stored inside a .rwdb bundle.
type bundleMeta struct {
	CheckpointID string `json:"checkpoint_id"`
	Message      string `json:"message"`
	SnapshotRef  string `json:"snapshot_ref"`
	BranchName   string `json:"branch_name"`
}

func bundleCmd() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "bundle <checkpoint>",
		Short: "Package a checkpoint into a portable .rwdb file",
		Long: `Packages a checkpoint and all its referenced objects into a single
gzip-compressed tar archive that can be shared and loaded on another machine.

Example:
  rw bundle S3 --output auth-working.rwdb`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				return err
			}

			cp, err := resolveCheckpoint(r.engine, args[0])
			if err != nil {
				return err
			}

			if cp.SnapshotRef == "" {
				return fmt.Errorf("Root checkpoint has no files to compare. Choose a later checkpoint.")
			}

			if output == "" {
				output = fmt.Sprintf("checkpoint-%s.rwdb", shortID(cp.ID))
			}

			snap, err := r.scanner.Load(cp.SnapshotRef)
			if err != nil {
				return fmt.Errorf("load snapshot: %w", err)
			}

			branch, _ := r.engine.Index.Branches[cp.BranchID]

			// Create the output archive.
			f, err := os.Create(output)
			if err != nil {
				return fmt.Errorf("create output: %w", err)
			}
			defer f.Close()

			gw := gzip.NewWriter(f)
			defer gw.Close()
			tw := tar.NewWriter(gw)
			defer tw.Close()

			// Write metadata.
			meta := bundleMeta{
				CheckpointID: cp.ID,
				Message:      cp.Message,
				SnapshotRef:  cp.SnapshotRef,
				BranchName:   branch.Name,
			}
			metaData, err := json.MarshalIndent(meta, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal bundle metadata: %w", err)
			}
			if err := tarWriteFile(tw, "meta.json", metaData); err != nil {
				return err
			}

			// Write each file from the snapshot.
			for _, entry := range snap.Files {
				content, err := r.store.Read(entry.Hash)
				if err != nil {
					return fmt.Errorf("read object %s for %s: %w", entry.Hash[:8], entry.Path, err)
				}
				if err := tarWriteFile(tw, "files/"+entry.Path, content); err != nil {
					return fmt.Errorf("write %s to bundle: %w", entry.Path, err)
				}
			}

			// Include .rewindignore if it exists.
			ignPath := filepath.Join(parentDir(r.cfg.RewindDir), ".rewindignore")
			if data, err := os.ReadFile(ignPath); err == nil {
				_ = tarWriteFile(tw, ".rewindignore", data)
			}

			printSuccess("bundled checkpoint %s → %s (%d files)", shortID(cp.ID), output, len(snap.Files))
			return nil
		},
	}

	cmd.Flags().StringVar(&output, "output", "", "output file path (default: checkpoint-<id>.rwdb)")
	return cmd
}

func loadBundleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "load <file.rwdb>",
		Short: "Restore a checkpoint from a portable .rwdb bundle",
		Long: `Loads a previously bundled checkpoint, writing all files to the
current working directory.

Example:
  rw load auth-working.rwdb`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bundlePath := args[0]

			f, err := os.Open(bundlePath)
			if err != nil {
				return fmt.Errorf("open bundle: %w", err)
			}
			defer f.Close()

			gr, err := gzip.NewReader(f)
			if err != nil {
				return fmt.Errorf("gzip reader: %w", err)
			}
			defer gr.Close()

			tr := tar.NewReader(gr)
			var meta bundleMeta
			fileCount := 0

			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}

			for {
				hdr, err := tr.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					return fmt.Errorf("read bundle: %w", err)
				}

				if hdr.Name == "meta.json" {
					data, readErr := io.ReadAll(tr)
					if readErr != nil {
						return fmt.Errorf("read meta.json: %w", readErr)
					}
					if err := json.Unmarshal(data, &meta); err != nil {
						return fmt.Errorf("parse meta.json: %w", err)
					}
					continue
				}

				if hdr.Typeflag == tar.TypeReg {
					// Strip "files/" prefix.
					relPath := hdr.Name
					if len(relPath) > 6 && relPath[:6] == "files/" {
						relPath = relPath[6:]
					}

					// Sanitize path to prevent zip-slip / path traversal.
					relPath = filepath.FromSlash(relPath)
					absPath := filepath.Join(cwd, relPath)
					if !strings.HasPrefix(absPath, cwd+string(filepath.Separator)) && absPath != cwd {
						return fmt.Errorf("invalid path in bundle: %q escapes project directory", hdr.Name)
					}

					if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
						return err
					}

					content, readErr := io.ReadAll(tr)
					if readErr != nil {
						return fmt.Errorf("read %s: %w", relPath, readErr)
					}
					if err := os.WriteFile(absPath, content, os.FileMode(hdr.Mode)); err != nil {
						return fmt.Errorf("write %s: %w", relPath, err)
					}
					fileCount++
				}
			}

			printSuccess("loaded bundle: %q (%d files)", meta.Message, fileCount)
			return nil
		},
	}
}

func tarWriteFile(tw *tar.Writer, name string, data []byte) error {
	hdr := &tar.Header{
		Name: name,
		Mode: 0o644,
		Size: int64(len(data)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err := tw.Write(data)
	return err
}
