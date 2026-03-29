package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	githubRepo     = "itsakash-real/nimbi"
	githubAPI      = "https://api.github.com/repos/" + githubRepo + "/releases/latest"
	versionCacheFile = "version-check.json"
	versionCheckTTL  = 24 * time.Hour
)

// githubRelease is the subset of the GitHub releases API response we need.
type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// versionCache is stored in .rewind/version-check.json to avoid hammering GitHub.
type versionCache struct {
	CheckedAt     time.Time `json:"checked_at"`
	LatestVersion string    `json:"latest_version"`
}

// ── rw upgrade ────────────────────────────────────────────────────────────────

func upgradeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade rw to the latest version",
		Args:  cobra.NoArgs,
		RunE:  runUpgrade,
	}
}

func runUpgrade(_ *cobra.Command, _ []string) error {
	sectionTitle("upgrade")
	fmt.Println()

	kv("current", colorCyan+Version+colorReset)

	// Fetch latest release info.
	kv("checking", "github.com/"+githubRepo)
	rel, err := fetchLatestRelease()
	if err != nil {
		return fmt.Errorf("could not reach GitHub: %w\n\n  Manual upgrade: go install github.com/%s/cmd/rw@latest", err, githubRepo)
	}

	latest := strings.TrimPrefix(rel.TagName, "v")
	kv("latest", colorCyan+rel.TagName+colorReset)
	fmt.Println()

	if Version != "dev" && !isNewer(latest, Version) {
		printSuccess("already on the latest version")
		fmt.Println()
		return nil
	}

	// Find the right asset for this OS/arch.
	assetName := platformAssetName(latest)
	downloadURL := ""
	for _, a := range rel.Assets {
		if strings.EqualFold(a.Name, assetName) {
			downloadURL = a.BrowserDownloadURL
			break
		}
	}
	if downloadURL == "" {
		// Fallback: tell user how to upgrade manually.
		fmt.Printf("  No pre-built binary found for %s/%s.\n\n", runtime.GOOS, runtime.GOARCH)
		fmt.Printf("  Upgrade manually:\n")
		fmt.Printf("  %sgo install github.com/%s/cmd/rw@latest%s\n\n", colorCyan, githubRepo, colorReset)
		return nil
	}

	// Download to a temp file.
	kv("downloading", assetName)
	tmpFile, err := downloadToTemp(downloadURL)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer os.Remove(tmpFile)

	// Extract the binary from the archive.
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot find current binary: %w", err)
	}
	exePath, _ = filepath.EvalSymlinks(exePath)

	newBin, err := extractBinary(tmpFile, assetName)
	if err != nil {
		return fmt.Errorf("extract failed: %w", err)
	}
	defer os.Remove(newBin)

	// Replace the current binary.
	if err := replaceBinary(exePath, newBin); err != nil {
		return fmt.Errorf("replace binary failed: %w\n\n  Try: go install github.com/%s/cmd/rw@latest", err, githubRepo)
	}

	// Update version cache.
	saveVersionCache(latest)

	fmt.Println()
	printSuccess("upgraded to %s", rel.TagName)
	fmt.Println()
	return nil
}

// ── background version check ─────────────────────────────────────────────────
// Called from main() as a goroutine. Checks cache first, hits GitHub at most
// once per 24 hours. Sends the notice string on ch (empty = up to date).

func startVersionCheck(rewindDir string, ch chan<- string) {
	go func() {
		notice := checkVersionNotice(rewindDir)
		ch <- notice
	}()
}

func checkVersionNotice(rewindDir string) string {
	if Version == "dev" {
		return "" // don't nag dev builds
	}

	cachePath := filepath.Join(rewindDir, versionCacheFile)
	cache := loadVersionCache(cachePath)

	var latest string
	if time.Since(cache.CheckedAt) < versionCheckTTL && cache.LatestVersion != "" {
		// Use cached value.
		latest = cache.LatestVersion
	} else {
		// Hit GitHub API.
		rel, err := fetchLatestRelease()
		if err != nil {
			return ""
		}
		latest = strings.TrimPrefix(rel.TagName, "v")
		saveVersionCacheAt(cachePath, latest)
	}

	if isNewer(latest, Version) {
		return fmt.Sprintf(
			"  %s⚡ rw %s is available (you have %s)%s  →  run %srw upgrade%s\n",
			colorYellow, latest, Version, colorReset,
			colorCyan, colorReset,
		)
	}
	return ""
}

// ── helpers ───────────────────────────────────────────────────────────────────

func fetchLatestRelease() (*githubRelease, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest("GET", githubAPI, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "nimbi/"+Version)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("no releases found yet — this is the first version")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

// platformAssetName returns the archive filename for the current OS/arch
// using the target version. Matches goreleaser naming: rewinddb_1.0.0_Linux_x86_64.tar.gz
func platformAssetName(version string) string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// strings.Title is deprecated; capitalize manually.
	osName := strings.ToUpper(goos[:1]) + goos[1:]
	archName := goarch
	if goarch == "amd64" {
		archName = "x86_64"
	}

	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}

	return fmt.Sprintf("rewinddb_%s_%s_%s%s", version, osName, archName, ext)
}

// downloadToTemp downloads url to a temp file and returns the path.
func downloadToTemp(url string) (string, error) {
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}

	tmp, err := os.CreateTemp("", "rw-upgrade-*")
	if err != nil {
		return "", err
	}
	defer tmp.Close()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	return tmp.Name(), nil
}

// extractBinary extracts the `rw` (or `rw.exe`) binary from the downloaded archive.
// Returns path to the extracted binary in a temp file.
func extractBinary(archivePath, archiveName string) (string, error) {
	binName := "rw"
	if runtime.GOOS == "windows" {
		binName = "rw.exe"
	}

	tmp, err := os.CreateTemp("", "rw-new-*")
	if err != nil {
		return "", err
	}
	tmp.Close()

	if strings.HasSuffix(archiveName, ".zip") {
		return tmp.Name(), extractFromZip(archivePath, binName, tmp.Name())
	}
	return tmp.Name(), extractFromTarGz(archivePath, binName, tmp.Name())
}

func extractFromTarGz(archivePath, binName, destPath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if filepath.Base(hdr.Name) == binName {
			out, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
			if err != nil {
				return err
			}
			_, err = io.Copy(out, tr)
			out.Close()
			return err
		}
	}
	return fmt.Errorf("binary %q not found in archive", binName)
}

func extractFromZip(archivePath, binName, destPath string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if filepath.Base(f.Name) == binName {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			out, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
			if err != nil {
				rc.Close()
				return err
			}
			_, err = io.Copy(out, rc)
			rc.Close()
			out.Close()
			return err
		}
	}
	return fmt.Errorf("binary %q not found in zip", binName)
}

// replaceBinary atomically replaces exePath with newBinPath.
// On Windows: rename old → .old, copy new → exePath, delete .old.
// On Unix: write to tmp in same dir, rename over.
func replaceBinary(exePath, newBinPath string) error {
	if runtime.GOOS == "windows" {
		oldPath := exePath + ".old"
		os.Remove(oldPath) // clean up any leftover from previous upgrade
		if err := os.Rename(exePath, oldPath); err != nil {
			return fmt.Errorf("cannot rename current binary: %w", err)
		}
		if err := copyFile(newBinPath, exePath, 0o755); err != nil {
			// Restore on failure.
			os.Rename(oldPath, exePath)
			return err
		}
		os.Remove(oldPath)
		return nil
	}

	// Unix: write to a tmp file in the same directory, then rename over.
	dir := filepath.Dir(exePath)
	tmp, err := os.CreateTemp(dir, ".rw-upgrade-*")
	if err != nil {
		return err
	}
	tmp.Close()
	os.Remove(tmp.Name())

	if err := copyFile(newBinPath, tmp.Name(), 0o755); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), exePath)
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// isNewer returns true if candidate is strictly newer than current.
// Both should be semver without the leading 'v'.
func isNewer(candidate, current string) bool {
	c := semverParts(candidate)
	cur := semverParts(current)
	for i := 0; i < 3; i++ {
		if c[i] > cur[i] {
			return true
		}
		if c[i] < cur[i] {
			return false
		}
	}
	return false
}

func semverParts(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	var out [3]int
	for i, p := range parts {
		if i >= 3 {
			break
		}
		fmt.Sscanf(p, "%d", &out[i])
	}
	return out
}

// ── version cache ─────────────────────────────────────────────────────────────

func loadVersionCache(path string) versionCache {
	data, err := os.ReadFile(path)
	if err != nil {
		return versionCache{}
	}
	var c versionCache
	json.Unmarshal(data, &c) //nolint:errcheck
	return c
}

func saveVersionCache(latest string) {
	// Try loadRepo first; fall back to findRewindDir so the cache
	// is saved even when run outside an initialized project.
	rewindDir := ""
	if r, err := loadRepo(); err == nil {
		rewindDir = r.cfg.RewindDir
	} else {
		rewindDir = findRewindDir()
	}
	if rewindDir == "" {
		return
	}
	saveVersionCacheAt(filepath.Join(rewindDir, versionCacheFile), latest)
}

func saveVersionCacheAt(path, latest string) {
	c := versionCache{CheckedAt: time.Now(), LatestVersion: latest}
	data, _ := json.Marshal(c)
	os.WriteFile(path, data, 0o644) //nolint:errcheck
}
