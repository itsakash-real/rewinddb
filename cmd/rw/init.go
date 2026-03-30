package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/itsakash-real/nimbi/internal/timeline"
	"github.com/spf13/cobra"
)

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize a new Nimbi repository in the current directory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("cannot determine working directory: %w", err)
			}

			engine, err := timeline.Init(cwd)
			if err != nil {
				if errors.Is(err, timeline.ErrAlreadyInitialized) {
					return fmt.Errorf("already a Nimbi repository — .rewind/ already exists")
				}
				return fmt.Errorf("init failed: %w", err)
			}

			rewindDir := filepath.Dir(engine.IndexPath)

			sectionTitle("initialized")
			fmt.Println()
			kv("directory", rewindDir)
			kv("branch", colorPurple+"main"+colorReset)
			fmt.Println()

			// ── Smart ignore: auto-detect project type and apply patterns ───
			detected := detectProjectTypes(cwd)
			if len(detected) > 0 {
				fmt.Printf("  %s✓ Detected:%s %s\n", colorGreen, colorReset, strings.Join(detected, ", "))

				candidates := smartIgnorePatterns(cwd)
				added := 0
				ignored := []string{}
				for _, pat := range candidates {
					ok, addErr := addIgnorePattern(cwd, pat)
					if addErr != nil {
						continue
					}
					if ok {
						ignored = append(ignored, pat)
						added++
					}
				}

				if added > 0 {
					fmt.Printf("  %sAuto-ignoring:%s %s\n", colorDim, colorReset, strings.Join(ignored, ", "))
				}

				tracking := smartTrackingHighlights(cwd)
				if len(tracking) > 0 {
					fmt.Printf("  %sTracking:%s %s\n", colorDim, colorReset, strings.Join(tracking, ", "))
				}
				fmt.Println()
			}

			dimP.Println("  run 'rw save \"first checkpoint\"' to get started")
			fmt.Println()
			return nil
		},
	}
}

// detectProjectTypes checks for common project markers and returns human-readable names.
func detectProjectTypes(root string) []string {
	var types []string

	if fileExistsAt(root, "package.json") {
		label := "Node.js project"
		if fileExistsAt(root, "next.config.js") || fileExistsAt(root, "next.config.mjs") || fileExistsAt(root, "next.config.ts") {
			label = "Next.js project"
		} else if fileExistsAt(root, "vite.config.js") || fileExistsAt(root, "vite.config.ts") {
			label = "Vite project"
		} else if fileExistsAt(root, "nuxt.config.ts") || fileExistsAt(root, "nuxt.config.js") {
			label = "Nuxt project"
		}
		types = append(types, label)
	}
	if fileExistsAt(root, "go.mod") {
		types = append(types, "Go project")
	}
	if fileExistsAt(root, "requirements.txt") || fileExistsAt(root, "pyproject.toml") || fileExistsAt(root, "setup.py") {
		types = append(types, "Python project")
	}
	if fileExistsAt(root, "Cargo.toml") {
		types = append(types, "Rust project")
	}
	if fileExistsAt(root, "pom.xml") || fileExistsAt(root, "build.gradle") || fileExistsAt(root, "build.gradle.kts") {
		types = append(types, "Java/Kotlin project")
	}
	if fileExistsAt(root, "Gemfile") {
		types = append(types, "Ruby project")
	}
	if fileExistsAt(root, "composer.json") {
		types = append(types, "PHP project")
	}
	if fileExistsAt(root, "mix.exs") {
		types = append(types, "Elixir project")
	}
	if fileExistsAt(root, "pubspec.yaml") {
		types = append(types, "Dart/Flutter project")
	}

	return types
}

// smartIgnorePatterns returns ignore patterns based on detected project type.
func smartIgnorePatterns(root string) []string {
	patterns := []string{}

	if fileExistsAt(root, "package.json") {
		patterns = append(patterns, "node_modules/", "dist/", ".next/", ".nuxt/", "coverage/", ".cache/", ".turbo/", ".parcel-cache/")
	}
	if fileExistsAt(root, "go.mod") {
		patterns = append(patterns, "vendor/", "*.test", "*.exe")
	}
	if fileExistsAt(root, "requirements.txt") || fileExistsAt(root, "pyproject.toml") || fileExistsAt(root, "setup.py") {
		patterns = append(patterns, "__pycache__/", ".venv/", "*.pyc", ".tox/", ".mypy_cache/", "*.egg-info/")
	}
	if fileExistsAt(root, "Cargo.toml") {
		patterns = append(patterns, "target/")
	}
	if fileExistsAt(root, "pom.xml") || fileExistsAt(root, "build.gradle") || fileExistsAt(root, "build.gradle.kts") {
		patterns = append(patterns, "target/", "build/", ".gradle/")
	}
	if fileExistsAt(root, "Gemfile") {
		patterns = append(patterns, "vendor/bundle/", ".bundle/")
	}
	if fileExistsAt(root, "composer.json") {
		patterns = append(patterns, "vendor/")
	}
	if fileExistsAt(root, "pubspec.yaml") {
		patterns = append(patterns, ".dart_tool/", "build/")
	}

	// Always add common temp/log patterns.
	patterns = append(patterns, "*.log", "tmp/", "temp/")

	return patterns
}

// smartTrackingHighlights returns files worth highlighting that nimbi tracks but git doesn't.
func smartTrackingHighlights(root string) []string {
	var highlights []string

	worthTracking := []string{
		".env", ".env.local", ".env.development", ".env.production",
		"config/local.json", "config/local.yaml", "config/local.yml",
	}

	for _, f := range worthTracking {
		if fileExistsAt(root, f) {
			highlights = append(highlights, f)
		}
	}

	// Check for compiled outputs / build artifacts that git typically ignores.
	gitignorePaths := []string{"build/", "out/", "compiled/"}
	for _, p := range gitignorePaths {
		full := filepath.Join(root, p)
		if info, err := os.Stat(full); err == nil && info.IsDir() {
			highlights = append(highlights, p+" (compiled outputs)")
			break
		}
	}

	return highlights
}
