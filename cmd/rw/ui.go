package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/itsakash-real/nimbi/internal/storage"
	"github.com/itsakash-real/nimbi/internal/timeline"
	"github.com/spf13/cobra"
)

// ── Styles ───────────────────────────────────────────────────────────────────

var (
	uiTitle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	uiSelected   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	uiNormal     = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	uiDim        = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	uiCyan       = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	uiStatusBar  = lipgloss.NewStyle().Background(lipgloss.Color("235")).Foreground(lipgloss.Color("250")).Padding(0, 1)
	uiBorder     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("99")).Padding(1, 2)
)

// ── Model ────────────────────────────────────────────────────────────────────

type uiModel struct {
	r           *repo
	checkpoints []*timeline.Checkpoint
	cursor      int
	headID      string
	branchName  string
	width       int
	height      int
	message     string // status message
	quitting    bool
}

func initialUIModel(r *repo) uiModel {
	cps, _ := r.engine.ListCheckpoints("")
	branch, _ := r.engine.Index.CurrentBranch()

	return uiModel{
		r:           r,
		checkpoints: cps,
		cursor:      0,
		headID:      r.engine.Index.CurrentCheckpointID,
		branchName:  branch.Name,
		width:       80,
		height:      24,
	}
}

func (m uiModel) Init() tea.Cmd {
	return nil
}

func (m uiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.checkpoints)-1 {
				m.cursor++
			}

		case "r", "R":
			// Restore selected checkpoint.
			if m.cursor < len(m.checkpoints) {
				cp := m.checkpoints[m.cursor]
				if cp.SnapshotRef == "" {
					m.message = "Cannot restore root checkpoint"
					return m, nil
				}
				lockPath := filepath.Join(m.r.cfg.RewindDir, storage.LockFileName)
				fl := storage.NewFileLock(lockPath)
				err := fl.WithLock(func() error {
					snap, err := m.r.scanner.Load(cp.SnapshotRef)
					if err != nil {
						return err
					}
					if _, err := m.r.engine.GotoCheckpoint(cp.ID); err != nil {
						return err
					}
					return m.r.scanner.Restore(snap)
				})
				if err != nil {
					m.message = fmt.Sprintf("Restore failed: %v", err)
				} else {
					m.headID = cp.ID
					m.message = fmt.Sprintf("Restored to %s", shortID(cp.ID))
				}
			}

		case "t", "T":
			// Tag the selected checkpoint (simple inline).
			if m.cursor < len(m.checkpoints) {
				cp := m.checkpoints[m.cursor]
				m.message = fmt.Sprintf("Tag %s (use 'rw tag %s <name>' from terminal)", shortID(cp.ID), shortID(cp.ID))
			}

		case "a", "A":
			// Annotate shortcut hint.
			if m.cursor < len(m.checkpoints) {
				cp := m.checkpoints[m.cursor]
				m.message = fmt.Sprintf("Annotate: run 'rw annotate %s \"note\"' from terminal", shortID(cp.ID))
			}

		case "d", "D":
			// Diff selected vs HEAD hint.
			if m.cursor < len(m.checkpoints) {
				cp := m.checkpoints[m.cursor]
				m.message = fmt.Sprintf("Diff: run 'rw diff %s' from terminal", shortID(cp.ID))
			}
		}
	}

	return m, nil
}

func (m uiModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Title bar.
	title := uiTitle.Render(fmt.Sprintf(" Nimbi  ─  %s branch ", m.branchName))
	b.WriteString(title + "\n\n")

	// Checkpoint list.
	listHeight := m.height - 8
	if listHeight < 5 {
		listHeight = 5
	}

	// Viewport windowing.
	start := 0
	if m.cursor >= listHeight {
		start = m.cursor - listHeight + 1
	}
	end := start + listHeight
	if end > len(m.checkpoints) {
		end = len(m.checkpoints)
	}

	for i := start; i < end; i++ {
		cp := m.checkpoints[i]
		elapsed := int64(time.Since(cp.CreatedAt).Seconds())
		timeStr := humanTime(elapsed)

		sNum := m.r.engine.Index.SNumberFor(cp.ID)
		idLabel := shortID(cp.ID)
		if sNum != "" {
			idLabel = fmt.Sprintf("%-4s %s", sNum, shortID(cp.ID))
		}

		msg := truncate(cp.Message, 36)
		isHead := cp.ID == m.headID

		marker := "○"
		if isHead {
			marker = "◆"
		}

		line := fmt.Sprintf("  %s  %s  %-38s  %s", marker, idLabel, msg, timeStr)

		if i == m.cursor {
			b.WriteString(uiSelected.Render(line) + "\n")
		} else if isHead {
			b.WriteString(uiCyan.Render(line) + "\n")
		} else {
			b.WriteString(uiDim.Render(line) + "\n")
		}
	}

	// Pad remaining space.
	for i := end - start; i < listHeight; i++ {
		b.WriteString("\n")
	}

	// Status bar.
	b.WriteString("\n")
	keys := uiStatusBar.Render("  [R] Restore  [D] Diff  [T] Tag  [A] Annotate  [Q] Quit  ")
	b.WriteString(keys + "\n")

	if m.message != "" {
		b.WriteString("\n  " + m.message + "\n")
	}

	return b.String()
}

// ── Command ──────────────────────────────────────────────────────────────────

func uiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ui",
		Short: "Open the interactive TUI checkpoint browser",
		Long: `Launches a full-screen terminal UI for browsing checkpoints.

Navigate with arrow keys or j/k. Press R to restore, D for diff info,
T for tag, A for annotate, Q or Esc to quit.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				return err
			}

			p := tea.NewProgram(initialUIModel(r), tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				return fmt.Errorf("ui: %w", err)
			}
			return nil
		},
	}
}
