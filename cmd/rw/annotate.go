package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

const notesFile = "notes.json"

func annotateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "annotate <checkpoint> <note>",
		Short: "Add a note to a checkpoint after the fact",
		Long: `Attach a text annotation to any existing checkpoint.
Notes are stored in .rewind/notes.json and shown by 'rw list' and 'rw show'.

Examples:
  rw annotate S3 "this is where auth broke"
  rw annotate d0d2536c "stable build"`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				return err
			}

			cp, err := resolveCheckpoint(r.engine, args[0])
			if err != nil {
				return err
			}

			note := args[1]
			notes, err := loadNotes(r.cfg.RewindDir)
			if err != nil {
				return err
			}

			notes[cp.ID] = note
			if err := saveNotes(r.cfg.RewindDir, notes); err != nil {
				return err
			}

			sNum := r.engine.Index.SNumberFor(cp.ID)
			label := shortID(cp.ID)
			if sNum != "" {
				label = sNum + " " + label
			}
			printSuccess("annotated %s: %q", label, note)
			return nil
		},
	}
}

// loadNotes reads .rewind/notes.json, returning an empty map if the file
// doesn't exist yet.
func loadNotes(rewindDir string) (map[string]string, error) {
	path := filepath.Join(rewindDir, notesFile)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return make(map[string]string), nil
	}
	if err != nil {
		return nil, fmt.Errorf("load notes: %w", err)
	}
	var notes map[string]string
	if err := json.Unmarshal(data, &notes); err != nil {
		return nil, fmt.Errorf("parse notes: %w", err)
	}
	return notes, nil
}

// saveNotes atomically writes notes to .rewind/notes.json.
func saveNotes(rewindDir string, notes map[string]string) error {
	path := filepath.Join(rewindDir, notesFile)
	data, err := json.MarshalIndent(notes, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal notes: %w", err)
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".notes-*.tmp")
	if err != nil {
		return fmt.Errorf("save notes: create temp: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}
