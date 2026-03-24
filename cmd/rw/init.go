package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/itsakash-real/rewinddb/internal/timeline"
	"github.com/spf13/cobra"
)

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize a new RewindDB repository in the current directory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("cannot determine working directory: %w", err)
			}

			engine, err := timeline.Init(cwd)
			if err != nil {
				if errors.Is(err, timeline.ErrAlreadyInitialized) {
					return fmt.Errorf("already a RewindDB repository — .rewind/ already exists")
				}
				return fmt.Errorf("init failed: %w", err)
			}

			branch, _ := engine.Index.CurrentBranch()
			fmt.Printf("✓ Initialized RewindDB in .rewind/\n")
			fmt.Printf("  Branch:  %s\n", branch.Name)
			fmt.Printf("  Index:   %s\n", engine.IndexPath)
			return nil
		},
	}
}
