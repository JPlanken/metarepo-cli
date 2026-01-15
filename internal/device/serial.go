package device

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// GetSerialNumber returns the hardware serial number of the current device
func GetSerialNumber() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return getMacSerial()
	case "linux":
		return getLinuxSerial()
	case "windows":
		return getWindowsSerial()
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// getMacSerial returns the serial number on macOS
func getMacSerial() (string, error) {
	cmd := exec.Command("ioreg", "-l")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run ioreg: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "IOPlatformSerialNumber") {
			// Extract the serial number from the line
			// Format: "IOPlatformSerialNumber" = "SERIAL"
			parts := strings.Split(line, "=")
			if len(parts) == 2 {
				serial := strings.TrimSpace(parts[1])
				serial = strings.Trim(serial, "\"")
				return serial, nil
			}
		}
	}

	return "", fmt.Errorf("could not find serial number in ioreg output")
}

// getLinuxSerial returns the serial number on Linux
func getLinuxSerial() (string, error) {
	// Try multiple methods to get a unique identifier

	// Method 1: DMI product serial
	cmd := exec.Command("cat", "/sys/class/dmi/id/product_serial")
	if output, err := cmd.Output(); err == nil {
		serial := strings.TrimSpace(string(output))
		if serial != "" && serial != "To Be Filled By O.E.M." {
			return serial, nil
		}
	}

	// Method 2: Machine ID
	cmd = exec.Command("cat", "/etc/machine-id")
	if output, err := cmd.Output(); err == nil {
		machineID := strings.TrimSpace(string(output))
		if machineID != "" {
			return machineID, nil
		}
	}

	// Method 3: Board serial
	cmd = exec.Command("cat", "/sys/class/dmi/id/board_serial")
	if output, err := cmd.Output(); err == nil {
		serial := strings.TrimSpace(string(output))
		if serial != "" && serial != "To Be Filled By O.E.M." {
			return serial, nil
		}
	}

	return "", fmt.Errorf("could not determine serial number on Linux")
}

// getWindowsSerial returns the serial number on Windows
func getWindowsSerial() (string, error) {
	cmd := exec.Command("wmic", "bios", "get", "serialnumber")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run wmic: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip header and empty lines
		if line != "" && line != "SerialNumber" {
			return line, nil
		}
	}

	return "", fmt.Errorf("could not find serial number in wmic output")
}
