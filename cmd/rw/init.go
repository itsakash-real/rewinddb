package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/itsakash-real/nimbi/internal/timeline"
	"github.com/spf13/cobra"
)

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize a new Nimbi repository in the current directory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("cannot determine working directory: %w", err)
			}

			engine, err := timeline.Init(cwd)
			if err != nil {
				if errors.Is(err, timeline.ErrAlreadyInitialized) {
					return fmt.Errorf("already a Nimbi repository — .rewind/ already exists")
				}
				return fmt.Errorf("init failed: %w", err)
			}

			rewindDir := filepath.Dir(engine.IndexPath)

			sectionTitle("initialized")
			fmt.Println()
			kv("directory", rewindDir)
			kv("branch",    colorPurple+"main"+colorReset)
			fmt.Println()
			return nil
		},
	}
}
