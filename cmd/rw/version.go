package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// These variables are stamped at build time via -ldflags.
var (
	Version   = "dev"
	BuildTime = "unknown"
	GoVersion = runtime.Version()
)

func versionCmd() *cobra.Command {
	var short bool

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if short {
				fmt.Println(Version)
				return
			}
			fmt.Printf("Nimbi %s (%s/%s)\n", Version, runtime.GOOS, runtime.GOARCH)
			fmt.Printf("Built:    %s\n", BuildTime)
			fmt.Printf("Go:       %s\n", GoVersion)
		},
	}

	cmd.Flags().BoolVar(&short, "short", false, "print only the version string")
	return cmd
}
