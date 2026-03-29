package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Default level (WarnLevel) is set in init() of colors.go to ensure it
	// takes effect before any package-level code emits log lines.

	var debugFlag bool
	var verboseFlag bool

	// Start background version check early. We collect the result channel
	// and print any notice after the command finishes.
	updateNoticeCh := make(chan string, 1)
	rewindDir := findRewindDir()
	if rewindDir != "" {
		startVersionCheck(rewindDir, updateNoticeCh)
	} else {
		updateNoticeCh <- ""
	}

	rootCmd := &cobra.Command{
		Use:          "rw",
		Short:        "Nimbi — a time-travel state engine for codebases",
		SilenceUsage: true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: false,
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if debugFlag || verboseFlag {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
			}
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			// Skip notice for upgrade itself to avoid confusion.
			if cmd.Name() == "upgrade" || cmd.Name() == "_shell_hook" {
				return
			}
			select {
			case notice := <-updateNoticeCh:
				if notice != "" {
					fmt.Println()
					fmt.Print(notice)
				}
			default:
				// goroutine not done yet — skip, don't block
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println()
			purpleBoldP.Println("  \u25c6  nimbi")
			dimP.Println("  time-travel for your codebase")
			fmt.Println()
			fmt.Printf("  %susage%s   rw <command> [flags]\n\n", colorPurpleDim, colorReset)
			dimP.Println("  run 'rw --help' for available commands")
			fmt.Println()
		},
	}

	rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "enable debug logging")
	rootCmd.PersistentFlags().BoolVar(&verboseFlag, "verbose", false, "enable verbose logging (alias for --debug)")

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
		upgradeCmd(),
		shellHookCmd(),
		shellSetupCmd(),
		listCmd(),
		branchesCmd(),
		diffCmd(),
		statusCmd(),
		gcCmd(),
		tagCmd(),
		healthCmd(),
		repairCmd(),
		timelineCmd(),
		annotateCmd(),
		protectCmd(),
		unprotectCmd(),
		listProtectedCmd(),
		stashCmd(),
		timeCmd(),
		heatmapCmd(),
		hooksCmd(),
		bundleCmd(),
		loadBundleCmd(),
		uiCmd(),
		shellInitCmd(),
		doctorCmd(),
		versionCmd(),
		completionCmd(rootCmd),
		manCmd(rootCmd),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// findRewindDir walks up from cwd looking for a .rewind directory.
// Returns the .rewind path if found, empty string otherwise.
func findRewindDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		candidate := filepath.Join(dir, ".rewind")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
