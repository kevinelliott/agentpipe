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
	if Version == "dev" || Version == "" || strings.Contains(Version, "dirty") {
		return false, "", nil
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Don't follow redirects, we just want the Location header
			return http.ErrUseLastResponse
		},
	}

	// Use the latest release redirect URL which doesn't count against rate limits
	// This returns a 302 redirect to the actual release page
	resp, err := client.Head("https://github.com/kevinelliott/agentpipe/releases/latest")
	if err != nil {
		return false, "", fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	// Check if we got a redirect (302 or 301)
	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusMovedPermanently {
		// Try the API as a fallback (will hit rate limits but worth a try)
		return checkViaAPI()
	}

	// Extract version from the redirect URL
	// The Location header will be something like:
	// https://github.com/kevinelliott/agentpipe/releases/tag/v1.0.0
	location := resp.Header.Get("Location")
	if location == "" {
		return false, "", fmt.Errorf("no redirect location found")
	}

	// Extract version from URL
	parts := strings.Split(location, "/")
	if len(parts) == 0 {
		return false, "", fmt.Errorf("invalid redirect URL")
	}
	
	latestTag := parts[len(parts)-1]
	latestVersion := strings.TrimPrefix(latestTag, "v")
	currentVersion := strings.TrimPrefix(Version, "v")

	// Simple version comparison (works for semantic versions)
	if compareVersions(latestVersion, currentVersion) > 0 {
		return true, latestTag, nil
	}

	// Return false with the latest version so we know the check succeeded
	return false, latestTag, nil
}

// checkViaAPI is a fallback that uses the GitHub API (subject to rate limits)
func checkViaAPI() (bool, string, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Add a user agent to be a good citizen
	req, err := http.NewRequest("GET", "https://api.github.com/repos/kevinelliott/agentpipe/releases/latest", nil)
	if err != nil {
		return false, "", err
	}
	req.Header.Set("User-Agent", "agentpipe/"+Version)

	resp, err := client.Do(req)
	if err != nil {
		return false, "", fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	// Check for rate limiting
	if resp.StatusCode == http.StatusForbidden {
		remaining := resp.Header.Get("X-RateLimit-Remaining")
		if remaining == "0" {
			// Silently ignore rate limit errors
			return false, "", nil
		}
	}

	if resp.StatusCode != http.StatusOK {
		// Silently ignore other API errors
		return false, "", nil
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