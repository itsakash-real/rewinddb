package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/itsakash-real/nimbi/internal/storage"
	"github.com/itsakash-real/nimbi/internal/timeline"
	"github.com/spf13/cobra"
)

// exportManifest is written inside the .rwdb archive as manifest.json.
type exportManifest struct {
	Version      int      `json:"version"`
	CheckpointID string   `json:"checkpoint_id"`
	Message      string   `json:"message"`
	CreatedAt    string   `json:"created_at"`
	FileCount    int      `json:"file_count"`
	Files        []string `json:"files"`
}

func exportCmd() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "export <checkpoint-id>",
		Short: "Export a checkpoint and its objects to a .rwdb archive",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				return err
			}

			// Resolve checkpoint.
			cp, err := resolveCheckpoint(r.engine, args[0])
			if err != nil {
				return err
			}
			if cp.SnapshotRef == "" {
				return fmt.Errorf("checkpoint %s has no snapshot", shortID(cp.ID))
			}

			// Determine output filename.
			outFile := output
			if outFile == "" {
				outFile = shortID(cp.ID) + ".rwdb"
			}

			// Load snapshot.
			snap, err := r.scanner.Load(cp.SnapshotRef)
			if err != nil {
				return fmt.Errorf("load snapshot: %w", err)
			}

			// Collect file hashes.
			var fileHashes []string
			for _, fe := range snap.Files {
				fileHashes = append(fileHashes, fe.Hash)
			}
			// Also include the snapshot JSON object hash.
			fileHashes = append(fileHashes, cp.SnapshotRef)

			// Build manifest.
			manifest := exportManifest{
				Version:      1,
				CheckpointID: cp.ID,
				Message:      cp.Message,
				CreatedAt:    cp.CreatedAt.UTC().Format(time.RFC3339),
				FileCount:    len(snap.Files),
				Files:        fileHashes,
			}

			// Write archive.
			f, err := os.Create(outFile)
			if err != nil {
				return fmt.Errorf("create output file: %w", err)
			}
			defer f.Close()

			gw := gzip.NewWriter(f)
			defer gw.Close()
			tw := tar.NewWriter(gw)
			defer tw.Close()

			// Write manifest.json.
			manifestData, err := json.MarshalIndent(manifest, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal manifest: %w", err)
			}
			if err := writeTarEntry(tw, "manifest.json", manifestData); err != nil {
				return fmt.Errorf("write manifest: %w", err)
			}

			// Write checkpoint JSON.
			cpData, err := json.MarshalIndent(cp, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal checkpoint: %w", err)
			}
			if err := writeTarEntry(tw, "checkpoint.json", cpData); err != nil {
				return fmt.Errorf("write checkpoint: %w", err)
			}

			// Write snapshot JSON.
			snapData, err := json.MarshalIndent(snap, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal snapshot: %w", err)
			}
			if err := writeTarEntry(tw, "snapshot.json", snapData); err != nil {
				return fmt.Errorf("write snapshot: %w", err)
			}

			// Write each object file.
			written := 0
			for _, fe := range snap.Files {
				data, readErr := r.store.Read(fe.Hash)
				if readErr != nil {
					return fmt.Errorf("read object %s: %w", fe.Hash[:8], readErr)
				}
				entryName := "objects/" + fe.Hash[:2] + "/" + fe.Hash[2:]
				if writeErr := writeTarEntry(tw, entryName, data); writeErr != nil {
					return fmt.Errorf("write object %s: %w", fe.Hash[:8], writeErr)
				}
				written++
			}

			fmt.Printf("%s✓ Exported checkpoint %s to %s%s\n",
				colorGreen, shortID(cp.ID), outFile, colorReset)
			fmt.Printf("  Message: %q\n", cp.Message)
			fmt.Printf("  Objects: %d\n", written)
			return nil
		},
	}

	cmd.Flags().StringVar(&output, "output", "", "output filename (default: <id-prefix>.rwdb)")
	return cmd
}

func importCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import <file.rwdb>",
		Short: "Import a .rwdb archive into the current repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				return err
			}

			inFile := args[0]
			f, err := os.Open(inFile)
			if err != nil {
				return fmt.Errorf("open archive: %w", err)
			}
			defer f.Close()

			gr, err := gzip.NewReader(f)
			if err != nil {
				return fmt.Errorf("read gzip: %w", err)
			}
			defer gr.Close()

			tr := tar.NewReader(gr)

			var manifest exportManifest
			var importedCP timeline.Checkpoint
			objectsWritten := 0

			for {
				hdr, err := tr.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					return fmt.Errorf("read archive: %w", err)
				}

				data, err := io.ReadAll(tr)
				if err != nil {
					return fmt.Errorf("read entry %s: %w", hdr.Name, err)
				}

				switch hdr.Name {
				case "manifest.json":
					if err := json.Unmarshal(data, &manifest); err != nil {
						return fmt.Errorf("parse manifest: %w", err)
					}
				case "checkpoint.json":
					if err := json.Unmarshal(data, &importedCP); err != nil {
						return fmt.Errorf("parse checkpoint: %w", err)
					}
				case "snapshot.json":
					// Written to object store below after we have the hash.
					if _, writeErr := r.store.Write(data); writeErr != nil {
						return fmt.Errorf("store snapshot: %w", writeErr)
					}
				default:
					// Object files are stored as objects/<hash[:2]>/<hash[2:]>.
					if len(data) > 0 {
						if _, writeErr := r.store.Write(data); writeErr != nil {
							return fmt.Errorf("store object from %s: %w", hdr.Name, writeErr)
						}
						objectsWritten++
					}
				}
			}

			// Create a new checkpoint in the current timeline referencing the imported snapshot.
			lockPath := filepath.Join(r.cfg.RewindDir, storage.LockFileName)
			fl := storage.NewFileLock(lockPath)
			var newCPID string
			err = fl.WithLock(func() error {
				newMsg := "imported: " + importedCP.Message
				newCP, saveErr := r.engine.SaveCheckpoint(newMsg, importedCP.SnapshotRef)
				if saveErr != nil {
					return fmt.Errorf("save imported checkpoint: %w", saveErr)
				}
				newCPID = newCP.ID
				return nil
			})
			if err != nil {
				return err
			}

			fmt.Printf("%s✓ Imported %s → new checkpoint %s%s\n",
				colorGreen, inFile, shortID(newCPID), colorReset)
			fmt.Printf("  Original message: %q\n", importedCP.Message)
			fmt.Printf("  Objects imported: %d\n", objectsWritten)
			return nil
		},
	}
}

// writeTarEntry writes a single file entry into a tar archive.
func writeTarEntry(tw *tar.Writer, name string, data []byte) error {
	hdr := &tar.Header{
		Name:    name,
		Mode:    0o644,
		Size:    int64(len(data)),
		ModTime: time.Now(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err := tw.Write(data)
	return err
}
