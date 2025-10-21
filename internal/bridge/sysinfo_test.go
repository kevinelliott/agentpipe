package bridge

import (
	"runtime"
	"strings"
	"testing"
)

func TestCollectSystemInfo(t *testing.T) {
	version := "0.2.4"
	sysInfo := CollectSystemInfo(version)

	// Test that required fields are populated
	if sysInfo.AgentPipeVersion != version {
		t.Errorf("Expected AgentPipeVersion=%s, got %s", version, sysInfo.AgentPipeVersion)
	}

	if sysInfo.OS != runtime.GOOS {
		t.Errorf("Expected OS=%s, got %s", runtime.GOOS, sysInfo.OS)
	}

	if sysInfo.GoVersion != runtime.Version() {
		t.Errorf("Expected GoVersion=%s, got %s", runtime.Version(), sysInfo.GoVersion)
	}

	if sysInfo.Architecture != runtime.GOARCH {
		t.Errorf("Expected Architecture=%s, got %s", runtime.GOARCH, sysInfo.Architecture)
	}

	// Test that OS version is not empty
	if sysInfo.OSVersion == "" {
		t.Error("Expected OSVersion to be populated, got empty string")
	}

	// Test that OS version doesn't contain "unknown" on current platform
	// (we should be able to detect the version on the test machine)
	if !strings.Contains(sysInfo.OSVersion, "unknown") {
		// This is the expected case - we should be able to detect the OS version
		t.Logf("Successfully detected OS version: %s", sysInfo.OSVersion)
	} else {
		t.Logf("Warning: OS version detection returned 'unknown': %s", sysInfo.OSVersion)
	}
}

func TestGetOSVersion(t *testing.T) {
	osVersion := getOSVersion()

	// Should never return empty string
	if osVersion == "" {
		t.Error("Expected getOSVersion to return non-empty string")
	}

	// Should contain OS-specific information
	switch runtime.GOOS {
	case "darwin":
		// macOS should return "macOS X.Y" or "macOS (version unknown)"
		if !strings.Contains(osVersion, "macOS") {
			t.Errorf("Expected macOS version string, got: %s", osVersion)
		}
	case "linux":
		// Linux should return distribution info or "Linux" keyword
		if !strings.Contains(osVersion, "Linux") && !strings.Contains(osVersion, "Ubuntu") &&
			!strings.Contains(osVersion, "Debian") && !strings.Contains(osVersion, "Red Hat") &&
			!strings.Contains(osVersion, "CentOS") && !strings.Contains(osVersion, "Fedora") {
			t.Logf("Unexpected Linux version format: %s", osVersion)
		}
	case "windows":
		// Windows should return "Windows" keyword
		if !strings.Contains(osVersion, "Windows") && !strings.Contains(osVersion, "Version") {
			t.Errorf("Expected Windows version string, got: %s", osVersion)
		}
	}

	t.Logf("Detected OS version: %s", osVersion)
}

func TestGetMacOSVersion(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping macOS-specific test on non-macOS platform")
	}

	version := getMacOSVersion()

	// Should contain "macOS"
	if !strings.Contains(version, "macOS") {
		t.Errorf("Expected version to contain 'macOS', got: %s", version)
	}

	// Should not be empty
	if version == "" {
		t.Error("Expected non-empty macOS version")
	}

	t.Logf("macOS version: %s", version)
}

func TestGetLinuxVersion(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific test on non-Linux platform")
	}

	version := getLinuxVersion()

	// Should not be empty
	if version == "" {
		t.Error("Expected non-empty Linux version")
	}

	// Should contain some identifying information
	if version == "unknown" {
		t.Error("Expected Linux version detection to succeed on Linux platform")
	}

	t.Logf("Linux version: %s", version)
}

func TestGetWindowsVersion(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test on non-Windows platform")
	}

	version := getWindowsVersion()

	// Should not be empty
	if version == "" {
		t.Error("Expected non-empty Windows version")
	}

	// Should contain "Windows" keyword
	if !strings.Contains(version, "Windows") && !strings.Contains(version, "Version") {
		t.Errorf("Expected version to contain Windows info, got: %s", version)
	}

	t.Logf("Windows version: %s", version)
}

func TestSystemInfoJSONSerialization(t *testing.T) {
	// Test that SystemInfo can be properly serialized to JSON
	// (This is important for event serialization)
	version := "0.2.4"
	sysInfo := CollectSystemInfo(version)

	// The JSON tags should be present and correct
	// We'll verify this in the events_test.go when we marshal full events
	if sysInfo.AgentPipeVersion == "" {
		t.Error("Expected AgentPipeVersion to be set")
	}
}
