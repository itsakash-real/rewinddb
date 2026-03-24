package main

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	rootCmd := &cobra.Command{
		Use:          "rw",
		Short:        "RewindDB — a time-travel state engine for codebases",
		SilenceUsage: true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: false, // expose `rw completion`
		},
	}

	rootCmd.AddCommand(
		initCmd(),
		saveCmd(),
		gotoCmd(),
		undoCmd(),
		runCmd(),
		watchCmd(),
		bisectCmd(),
		statsCmd(),
		searchCmd(),
		sessionCmd(),
		ignoreCmd(),
		exportCmd(),
		importCmd(),
		shellHookCmd(),
		shellSetupCmd(),
		listCmd(),
		branchesCmd(),
		diffCmd(),
		statusCmd(),
		gcCmd(),
		tagCmd(),
		versionCmd(),
		completionCmd(rootCmd),
		manCmd(rootCmd),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
