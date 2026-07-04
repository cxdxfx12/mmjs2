//go:build windows
// +build windows

package license

import (
	"os/exec"
	"strings"
	"syscall"
)

func getCPUSerial() string {
	cmd := exec.Command("wmic", "cpu", "get", "ProcessorId")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.EqualFold(line, "ProcessorId") {
			return line
		}
	}
	return ""
}

func getDiskSerial() string {
	cmd := exec.Command("wmic", "diskdrive", "where", `DeviceID='\\.\PHYSICALDRIVE0'`, "get", "SerialNumber")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.EqualFold(line, "SerialNumber") {
			return line
		}
	}
	return ""
}
