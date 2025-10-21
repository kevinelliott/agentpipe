package bridge

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// SystemInfo contains system information collected for streaming events
type SystemInfo struct {
	AgentPipeVersion string `json:"agentpipe_version"`
	OS               string `json:"os"`
	OSVersion        string `json:"os_version"`
	GoVersion        string `json:"go_version"`
	Architecture     string `json:"architecture"`
}

// CollectSystemInfo collects system information for the current environment
func CollectSystemInfo(version string) SystemInfo {
	osVersion := getOSVersion()

	return SystemInfo{
		AgentPipeVersion: version,
		OS:               runtime.GOOS,
		OSVersion:        osVersion,
		GoVersion:        runtime.Version(),
		Architecture:     runtime.GOARCH,
	}
}

// getOSVersion returns the OS version string for the current platform
func getOSVersion() string {
	switch runtime.GOOS {
	case "darwin":
		return getMacOSVersion()
	case "linux":
		return getLinuxVersion()
	case "windows":
		return getWindowsVersion()
	default:
		return "unknown"
	}
}

// getMacOSVersion returns the macOS version string
func getMacOSVersion() string {
	cmd := exec.Command("sw_vers", "-productVersion")
	output, err := cmd.Output()
	if err != nil {
		return "macOS (version unknown)"
	}

	version := strings.TrimSpace(string(output))
	return fmt.Sprintf("macOS %s", version)
}

// getLinuxVersion returns the Linux distribution version string
func getLinuxVersion() string {
	// Try to read /etc/os-release (standard location on most modern distributions)
	cmd := exec.Command("sh", "-c", "cat /etc/os-release 2>/dev/null | grep -E '^(PRETTY_NAME|NAME|VERSION_ID)=' | head -1")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		// Parse the output - typically "PRETTY_NAME="Ubuntu 22.04.3 LTS""
		line := strings.TrimSpace(string(output))
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			// Remove quotes
			value := strings.Trim(parts[1], "\"'")
			return value
		}
	}

	// Fallback: try lsb_release
	cmd = exec.Command("lsb_release", "-ds")
	output, err = cmd.Output()
	if err == nil && len(output) > 0 {
		return strings.Trim(strings.TrimSpace(string(output)), "\"'")
	}

	// Fallback: try uname
	cmd = exec.Command("uname", "-sr")
	output, err = cmd.Output()
	if err == nil && len(output) > 0 {
		return strings.TrimSpace(string(output))
	}

	return "Linux (version unknown)"
}

// getWindowsVersion returns the Windows version string
func getWindowsVersion() string {
	// Try using wmic first (more detailed information)
	cmd := exec.Command("wmic", "os", "get", "Caption", "/value")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		// Parse output like "Caption=Microsoft Windows 11 Pro"
		lines := bytes.Split(output, []byte("\n"))
		for _, line := range lines {
			lineStr := strings.TrimSpace(string(line))
			if strings.HasPrefix(lineStr, "Caption=") {
				return strings.TrimPrefix(lineStr, "Caption=")
			}
		}
	}

	// Fallback: try ver command
	cmd = exec.Command("cmd", "/c", "ver")
	output, err = cmd.Output()
	if err == nil && len(output) > 0 {
		version := strings.TrimSpace(string(output))
		// ver output is typically like "Microsoft Windows [Version 10.0.22621.1]"
		// Extract just the relevant part
		if strings.Contains(version, "[") && strings.Contains(version, "]") {
			start := strings.Index(version, "[")
			end := strings.Index(version, "]")
			if start < end {
				return strings.TrimSpace(version[start+1 : end])
			}
		}
		return version
	}

	return "Windows (version unknown)"
}
