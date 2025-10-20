package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// VersionInfo contains version information for an agent
type VersionInfo struct {
	Installed string // Version currently installed (empty if not installed)
	Latest    string // Latest version available
	HasUpdate bool   // True if installed version is older than latest
}

// GetLatestVersion fetches the latest version for an agent from its package manager
func (a *AgentDefinition) GetLatestVersion() (string, error) {
	switch a.PackageManager {
	case "npm":
		return getNPMLatestVersion(a.PackageName)
	case "homebrew":
		return getHomebrewLatestVersion(a.PackageName)
	case "github":
		return getGitHubLatestRelease(a.PackageName)
	case "script":
		return getScriptVersion(a.PackageName)
	default:
		return "", fmt.Errorf("no package manager configured for %s", a.Name)
	}
}

// getNPMLatestVersion fetches the latest version of an npm package
func getNPMLatestVersion(packageName string) (string, error) {
	// Use npm registry API
	url := fmt.Sprintf("https://registry.npmjs.org/%s/latest", packageName)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch npm package info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("npm registry returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read npm response: %w", err)
	}

	var data struct {
		Version string `json:"version"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return "", fmt.Errorf("failed to parse npm response: %w", err)
	}

	if data.Version == "" {
		return "", fmt.Errorf("no version found in npm registry response")
	}

	return data.Version, nil
}

// getHomebrewLatestVersion fetches the latest version of a homebrew formula
func getHomebrewLatestVersion(formulaName string) (string, error) {
	// Use Homebrew Formulae API
	url := fmt.Sprintf("https://formulae.brew.sh/api/formula/%s.json", formulaName)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch homebrew formula info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("homebrew api returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read homebrew response: %w", err)
	}

	var data struct {
		Versions struct {
			Stable string `json:"stable"`
		} `json:"versions"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return "", fmt.Errorf("failed to parse homebrew response: %w", err)
	}

	if data.Versions.Stable == "" {
		return "", fmt.Errorf("no stable version found in homebrew api response")
	}

	return data.Versions.Stable, nil
}

// getGitHubLatestRelease fetches the latest release version from GitHub
func getGitHubLatestRelease(repoName string) (string, error) {
	// Use GitHub API to get latest release
	// repoName should be in format "owner/repo"
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repoName)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent header (required by GitHub API)
	req.Header.Set("User-Agent", "agentpipe-cli")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch github release info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read github response: %w", err)
	}

	var data struct {
		TagName string `json:"tag_name"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return "", fmt.Errorf("failed to parse github response: %w", err)
	}

	if data.TagName == "" {
		return "", fmt.Errorf("no tag found in github release response")
	}

	// Remove 'v' prefix if present
	version := strings.TrimPrefix(data.TagName, "v")
	return version, nil
}

// getScriptVersion fetches version from a shell script that contains VER= definition
func getScriptVersion(scriptURL string) (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(scriptURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch script: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("script fetch returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read script: %w", err)
	}

	// Look for VER="x.y.z" pattern in the script
	scriptContent := string(body)
	lines := strings.Split(scriptContent, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "VER=") {
			// Extract version from VER="x.y.z" or VER=x.y.z
			version := strings.TrimPrefix(line, "VER=")
			version = strings.Trim(version, "\"'")
			if version != "" {
				return version, nil
			}
		}
	}

	return "", fmt.Errorf("no VER definition found in script")
}

// GetInstalledVersion gets the currently installed version of an agent
func GetInstalledVersion(command string) string {
	// Try --version first
	cmd := exec.Command(command, "--version")
	output, err := cmd.CombinedOutput()
	if err == nil || len(output) > 0 {
		// Even if command exits with error, we might have version info in output
		version := strings.TrimSpace(string(output))

		// Handle multi-line output - look for version in all lines
		lines := strings.Split(version, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			// Skip empty lines
			if line == "" {
				continue
			}
			// Look for lines that contain version info (even in warnings)
			if containsVersion(line) {
				version = extractVersionNumber(line)
				if version != "" && version != line {
					// Successfully extracted a version that's different from the whole line
					return version
				}
			}
		}

		// Fallback: try first non-warning line
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "Warning:") && !strings.HasPrefix(line, "Error:") {
				version = extractVersionNumber(line)
				if version != "" {
					return version
				}
			}
		}
	}

	// Try version subcommand
	cmd = exec.Command(command, "version")
	if output, err := cmd.CombinedOutput(); err == nil {
		version := strings.TrimSpace(string(output))
		if lines := strings.Split(version, "\n"); len(lines) > 0 {
			version = strings.TrimSpace(lines[0])
		}
		version = extractVersionNumber(version)
		return version
	}

	return ""
}

// containsVersion checks if a string appears to contain version information
func containsVersion(s string) bool {
	// Look for version keywords or version-like patterns
	lower := strings.ToLower(s)
	return strings.Contains(lower, "version") ||
		strings.Contains(lower, "client version") ||
		strings.Contains(s, ".") && containsDigit(s)
}

// extractVersionNumber extracts a semantic version number from a string
func extractVersionNumber(s string) string {
	// Common patterns:
	// "command v1.2.3"
	// "command 1.2.3"
	// "v1.2.3"
	// "1.2.3"

	// Remove common prefixes
	s = strings.TrimSpace(s)

	// Split on whitespace and look for version-like strings
	parts := strings.Fields(s)
	for _, part := range parts {
		// Remove leading 'v' or 'V'
		part = strings.TrimPrefix(part, "v")
		part = strings.TrimPrefix(part, "V")

		// Check if it looks like a version (contains dots and numbers)
		if strings.Contains(part, ".") && containsDigit(part) {
			return part
		}
	}

	// If we didn't find anything, return the whole string
	return s
}

// containsDigit checks if a string contains at least one digit
func containsDigit(s string) bool {
	for _, c := range s {
		if c >= '0' && c <= '9' {
			return true
		}
	}
	return false
}

// CompareVersions compares two semantic version strings
// Returns:
//
//	-1 if v1 < v2
//	 0 if v1 == v2
//	 1 if v1 > v2
//	error if versions cannot be parsed
func CompareVersions(v1, v2 string) (int, error) {
	// Simple semantic version comparison
	// Split on dots and compare each part

	// Clean versions
	v1 = strings.TrimPrefix(v1, "v")
	v1 = strings.TrimPrefix(v1, "V")
	v2 = strings.TrimPrefix(v2, "v")
	v2 = strings.TrimPrefix(v2, "V")

	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	// Compare each part
	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var p1, p2 int

		if i < len(parts1) {
			// Extract numeric part only
			numStr := extractNumericPrefix(parts1[i])
			fmt.Sscanf(numStr, "%d", &p1)
		}

		if i < len(parts2) {
			numStr := extractNumericPrefix(parts2[i])
			fmt.Sscanf(numStr, "%d", &p2)
		}

		if p1 < p2 {
			return -1, nil
		}
		if p1 > p2 {
			return 1, nil
		}
	}

	return 0, nil
}

// extractNumericPrefix extracts the numeric prefix from a version part
// e.g., "3beta" -> "3", "12-rc1" -> "12"
func extractNumericPrefix(s string) string {
	result := ""
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result += string(c)
		} else {
			break
		}
	}
	if result == "" {
		return "0"
	}
	return result
}

// GetVersionInfo returns complete version information for an agent
func (a *AgentDefinition) GetVersionInfo(installedVersion string) (*VersionInfo, error) {
	latest, err := a.GetLatestVersion()
	if err != nil {
		return nil, err
	}

	info := &VersionInfo{
		Installed: installedVersion,
		Latest:    latest,
		HasUpdate: false,
	}

	// Only compare if we have an installed version
	if installedVersion != "" {
		cmp, err := CompareVersions(installedVersion, latest)
		if err == nil && cmp < 0 {
			info.HasUpdate = true
		}
	}

	return info, nil
}
