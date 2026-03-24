package main

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Default: only show WARN and above. --debug flag overrides this.
	zerolog.SetGlobalLevel(zerolog.WarnLevel)

	var debugFlag bool

	rootCmd := &cobra.Command{
		Use:          "rw",
		Short:        "RewindDB — a time-travel state engine for codebases",
		SilenceUsage: true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: false, // expose `rw completion`
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if debugFlag {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println()
			purpleBoldP.Println("  \u25c6  rewinddb")
			dimP.Println("  time-travel for your codebase")
			fmt.Println()
			fmt.Printf("  %susage%s   rw <command> [flags]\n\n", colorPurpleDim, colorReset)
			dimP.Println("  run 'rw --help' for available commands")
			fmt.Println()
		},
	}

	rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "enable debug logging")

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
