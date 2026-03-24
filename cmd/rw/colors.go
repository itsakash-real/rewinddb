package main

import (
	"fmt"

	"github.com/fatih/color"
)

// ANSI terminal colour/style codes.
// All commands in this package share these constants.
// callers that don't want colour should strip \033[...m sequences before writing.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

// Convenience print helpers using fatih/color for semantic colouring.

var (
	successPrinter = color.New(color.FgGreen, color.Bold)
	errorPrinter   = color.New(color.FgRed, color.Bold)
	warningPrinter = color.New(color.FgYellow)
	infoPrinter    = color.New(color.FgCyan)
	dimPrinter     = color.New(color.Faint)
)

func printSuccess(format string, args ...interface{}) {
	successPrinter.Printf(format+"\n", args...)
}

func printError(format string, args ...interface{}) {
	errorPrinter.Printf(format+"\n", args...)
}

func printWarning(format string, args ...interface{}) {
	warningPrinter.Printf(format+"\n", args...)
}

func printInfo(format string, args ...interface{}) {
	infoPrinter.Printf(format+"\n", args...)
}

func printDim(format string, args ...interface{}) {
	dimPrinter.Printf(format+"\n", args...)
}

// humanTime returns a human-readable relative time string using time.Time.
func humanTime(sec int64) string {
	switch {
	case sec < 60:
		return fmt.Sprintf("%ds ago", sec)
	case sec < 3600:
		m := sec / 60
		if m == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	case sec < 86400:
		h := sec / 3600
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	default:
		d := sec / 86400
		if d == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", d)
	}
}
