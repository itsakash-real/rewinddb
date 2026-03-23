package main

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func main() {
	// Pretty console logging for CLI output
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	rootCmd := &cobra.Command{
		Use:   "rw",
		Short: "RewindDB — a time-travel state engine for codebases",
		Long: `RewindDB lets you snapshot, diff, and restore the runtime state
of your codebase like Git, but for runtime state rather than source files.`,
		SilenceUsage: true,
	}

	rootCmd.AddCommand(
		saveCmd(),
		gotoCmd(),
		listCmd(),
		branchesCmd(),
		diffCmd(),
		gcCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal().Err(err).Msg("command failed")
		os.Exit(1)
	}
}

func saveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "save [message]",
		Short: "Save a snapshot of the current runtime state",
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Info().Msg("save: not implemented")
			return nil
		},
	}
}

func gotoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "goto <snapshot-id>",
		Short: "Restore state to a previous snapshot",
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Info().Msg("goto: not implemented")
			return nil
		},
	}
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all snapshots on the current branch",
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Info().Msg("list: not implemented")
			return nil
		},
	}
}

func branchesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "branches",
		Short: "List all branches",
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Info().Msg("branches: not implemented")
			return nil
		},
	}
}

func diffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diff <snapshot-a> <snapshot-b>",
		Short: "Show delta between two snapshots",
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Info().Msg("diff: not implemented")
			return nil
		},
	}
}

func gcCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "gc",
		Short: "Garbage-collect unreachable objects from the store",
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Info().Msg("gc: not implemented")
			return nil
		},
	}
}
