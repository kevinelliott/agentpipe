package version

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var (
	// Version is the current version of agentpipe
	// This will be set at build time using -ldflags
	Version = "dev"
	
	// CommitHash is the git commit hash
	CommitHash = "unknown"
	
	// BuildDate is the build date
	BuildDate = "unknown"
)

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	HTMLURL string `json:"html_url"`
}

// CheckForUpdate checks if there's a newer version available on GitHub
func CheckForUpdate() (bool, string, error) {
	// Skip update check for dev builds
	if Version == "dev" || Version == "" {
		return false, "", nil
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get("https://api.github.com/repos/kevinelliott/agentpipe/releases/latest")
	if err != nil {
		return false, "", fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return false, "", fmt.Errorf("failed to parse release info: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	currentVersion := strings.TrimPrefix(Version, "v")

	// Simple version comparison (works for semantic versions)
	if compareVersions(latestVersion, currentVersion) > 0 {
		return true, release.TagName, nil
	}

	return false, "", nil
}

// compareVersions compares two semantic versions
// Returns: 1 if v1 > v2, -1 if v1 < v2, 0 if equal
func compareVersions(v1, v2 string) int {
	// Remove 'v' prefix if present
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	// Split versions into parts
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	// Ensure both have at least 3 parts (major.minor.patch)
	for len(parts1) < 3 {
		parts1 = append(parts1, "0")
	}
	for len(parts2) < 3 {
		parts2 = append(parts2, "0")
	}

	// Compare each part
	for i := 0; i < 3; i++ {
		var n1, n2 int
		fmt.Sscanf(parts1[i], "%d", &n1)
		fmt.Sscanf(parts2[i], "%d", &n2)

		if n1 > n2 {
			return 1
		}
		if n1 < n2 {
			return -1
		}
	}

	return 0
}

// GetVersionString returns the full version string
func GetVersionString() string {
	if Version == "dev" {
		return fmt.Sprintf("agentpipe version: dev (commit: %s, built: %s)", CommitHash, BuildDate)
	}
	return fmt.Sprintf("agentpipe version: %s (commit: %s, built: %s)", Version, CommitHash, BuildDate)
}

// GetShortVersion returns just the version number
func GetShortVersion() string {
	return Version
}