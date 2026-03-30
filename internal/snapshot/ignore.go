package snapshot

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/rs/zerolog/log"
)

// defaultIgnores are always excluded regardless of .rewindignore [file:139].
var defaultIgnores = []string{
	".rewind/**",
	".git/**",
	"node_modules/**",
	"vendor/**",
	"__pycache__/**",
	".DS_Store",
	"*.swp",
	"*.swo",
	"*~",
	".idea/**",
	".vscode/**",
	"*.pyc",
	"dist/**",
	"build/**",
	"target/**",
	"*.o",
	"*.a",
	"*.so",
	"*.dll",
	"*.exe",
	"**/*.pyc",
	"**/*.o",
	"**/*.a",
}

// ignoreList holds compiled patterns from defaultIgnores + .rewindignore.
type ignoreList struct {
	patterns []string
}

// loadIgnoreList reads .rewindignore from projectRoot and merges with defaults.
// Missing .rewindignore is silently ignored. [web:148] for doublestar matching.
func loadIgnoreList(projectRoot string) *ignoreList {
	patterns := make([]string, len(defaultIgnores))
	copy(patterns, defaultIgnores)

	ignoreFile := filepath.Join(projectRoot, ".rewindignore")
	f, err := os.Open(ignoreFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Warn().Err(err).Msg("ignore: could not read .rewindignore")
		}
		return &ignoreList{patterns: patterns}
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue // blank lines and comments
		}
		// Normalize negations (! prefix) — not yet supported, skip.
		if strings.HasPrefix(line, "!") {
			log.Debug().Str("pattern", line).Msg("ignore: negation patterns not supported, skipping")
			continue
		}
		// Directories without trailing slash: match as prefix glob.
		if !strings.Contains(line, "/") {
			patterns = append(patterns, "**/"+line)
			patterns = append(patterns, "**/"+line+"/**")
		} else {
			patterns = append(patterns, line)
		}
	}
	return &ignoreList{patterns: patterns}
}

// matches returns true if relPath (slash-separated, relative to project root)
// should be excluded. Uses doublestar for ** support [web:148].
func (il *ignoreList) matches(relPath string) bool {
	for _, pattern := range il.patterns {
		ok, err := doublestar.Match(pattern, relPath)
		if err == nil && ok {
			return true
		}
	}
	return false
}

// IgnoreList is the exported wrapper around ignoreList for use by cmd code.
type IgnoreList struct {
	inner *ignoreList
}

// LoadIgnoreListPublic returns an exported IgnoreList for the given project root.
func LoadIgnoreListPublic(projectRoot string) *IgnoreList {
	return &IgnoreList{inner: loadIgnoreList(projectRoot)}
}

// Matches returns true if the given relative path should be ignored.
func (il *IgnoreList) Matches(relPath string) bool {
	return il.inner.matches(relPath)
}
