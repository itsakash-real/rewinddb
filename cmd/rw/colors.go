package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
)

// ─── ANSI codes ───────────────────────────────────────────────────────────────
// Primary brand color: purple (bright magenta).
// All vars are zeroed when color output is disabled.
var (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"

	// Purple / violet — primary RewindDB brand color.
	colorPurple     = "\033[95m"       // bright magenta
	colorPurpleDim  = "\033[38;5;141m" // soft lavender
	colorPurpleBold = "\033[1;95m"     // bold bright magenta
)

func init() {
	// Suppress debug/info logs early — main() may override via --debug flag.
	zerolog.SetGlobalLevel(zerolog.WarnLevel)

	noColor := false
	if os.Getenv("NO_COLOR") != "" {
		noColor = true
	} else if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		noColor = true
	} else if runtime.GOOS == "windows" {
		// Detect legacy Windows CMD (conhost) which doesn't support ANSI.
		// Windows Terminal and modern terminals set WT_SESSION or TERM.
		if os.Getenv("WT_SESSION") == "" && os.Getenv("TERM") == "" && os.Getenv("TERM_PROGRAM") == "" {
			noColor = true
		} else {
			noColor = color.NoColor
		}
	}

	if noColor {
		color.NoColor = true
		colorReset = ""
		colorRed = ""
		colorGreen = ""
		colorYellow = ""
		colorCyan = ""
		colorBold = ""
		colorDim = ""
		colorPurple = ""
		colorPurpleDim = ""
		colorPurpleBold = ""
	}
}

// ─── fatih/color printers ─────────────────────────────────────────────────────

var (
	purpleP     = color.New(color.FgHiMagenta)
	purpleBoldP = color.New(color.FgHiMagenta, color.Bold)
	cyanP       = color.New(color.FgCyan)
	greenBoldP  = color.New(color.FgGreen, color.Bold)
	redBoldP    = color.New(color.FgRed, color.Bold)
	yellowP     = color.New(color.FgYellow)
	dimP        = color.New(color.Faint)
	boldP       = color.New(color.Bold)
)

func printSuccess(format string, args ...interface{}) {
	greenBoldP.Printf("  ✓  "+format+"\n", args...)
}

func printError(format string, args ...interface{}) {
	redBoldP.Printf("  ✗  "+format+"\n", args...)
}

func printWarning(format string, args ...interface{}) {
	yellowP.Printf("  ⚠  "+format+"\n", args...)
}

func printInfo(format string, args ...interface{}) {
	purpleP.Printf("  "+format+"\n", args...)
}

func printDim(format string, args ...interface{}) {
	dimP.Printf("  "+format+"\n", args...)
}

// ─── Layout helpers ───────────────────────────────────────────────────────────

// sectionTitle prints a purple ◆ header with a dim trailing rule.
//
//	  ◆  title  ──────────────────────
func sectionTitle(title string) {
	label := purpleBoldP.Sprint("◆  " + title)
	ruleLen := 44 - len(title) - 3
	if ruleLen < 4 {
		ruleLen = 4
	}
	fmt.Printf("\n  %s  %s\n", label, dimP.Sprint(strings.Repeat("─", ruleLen)))
}

// hrule prints a dim horizontal rule at the given width.
func hrule(width int) {
	fmt.Printf("  %s%s%s\n", colorDim, strings.Repeat("─", width), colorReset)
}

// kv prints a key-value row: purple-dim key, normal value.
func kv(key, val string) {
	fmt.Printf("     %s%-14s%s %s\n", colorPurpleDim, key, colorReset, val)
}

// boxTop / boxLine / boxBottom render a rounded box with purple-dim borders.
func boxTop(width int) {
	fmt.Printf("  %s╭%s╮%s\n", colorPurpleDim, strings.Repeat("─", width), colorReset)
}

func boxLine(content string, width int) {
	visible := len([]rune(stripANSI(content)))
	pad := width - visible - 2
	if pad < 0 {
		pad = 0
	}
	fmt.Printf("  %s│%s %s%s %s│%s\n",
		colorPurpleDim, colorReset,
		content,
		strings.Repeat(" ", pad),
		colorPurpleDim, colorReset,
	)
}

func boxBottom(width int) {
	fmt.Printf("  %s╰%s╯%s\n", colorPurpleDim, strings.Repeat("─", width), colorReset)
}

// stripANSI strips escape sequences for visible-length measurement.
func stripANSI(s string) string {
	var out []rune
	inEsc := false
	for _, r := range s {
		if r == '\033' {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEsc = false
			}
			continue
		}
		out = append(out, r)
	}
	return string(out)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ─── humanTime ────────────────────────────────────────────────────────────────

func humanTime(sec int64) string {
	switch {
	case sec < 5:
		return "just now"
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
