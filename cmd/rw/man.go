package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func manCmd(rootCmd *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:    "man [output-dir]",
		Short:  "Generate man pages for rw and all subcommands",
		Hidden: true, // maintenance command; hide from normal help
		Args:   cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			outputDir := "./man/man1"
			if len(args) == 1 {
				outputDir = args[0]
			}

			if err := os.MkdirAll(outputDir, 0o755); err != nil {
				return fmt.Errorf("man: mkdir %s: %w", outputDir, err)
			}

			// GenManHeader provides metadata embedded in each .1 file [web:153].
			now := time.Now()
			header := &doc.GenManHeader{
				Title:   "RW",
				Section: "1",
				Date:    &now,
				Source:  fmt.Sprintf("RewindDB %s", Version),
				Manual:  "RewindDB Manual",
			}

			if err := doc.GenManTree(rootCmd, header, outputDir); err != nil {
				return fmt.Errorf("man: generate: %w", err)
			}

			fmt.Printf("✓ Man pages written to %s/\n", outputDir)
			fmt.Printf("  Install: sudo cp %s/*.1 /usr/local/share/man/man1/ && mandb\n", outputDir)
			return nil
		},
	}
}
