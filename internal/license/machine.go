package license

import (
	"crypto/sha256"
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
)

// GetMachineCode 获取唯一的机器码
func GetMachineCode() string {
	var parts []string

	if runtime.GOOS == "windows" {
		if mac := getMACAddress(); mac != "" {
			parts = append(parts, mac)
		}
		if cpu := getCPUSerial(); cpu != "" {
			parts = append(parts, cpu)
		}
		if disk := getDiskSerial(); disk != "" {
			parts = append(parts, disk)
		}
		if len(parts) == 0 {
			host, _ := os.Hostname()
			parts = append(parts, host)
		}
	} else {
		if mac := getMACAddress(); mac != "" {
			parts = append(parts, mac)
		}
		host, _ := os.Hostname()
		parts = append(parts, host)
	}

	combined := strings.Join(parts, "|")
	hash := sha256.Sum256([]byte(combined))
	machineCode := fmt.Sprintf("%x", hash)[:32]

	return formatMachineCode(machineCode)
}

func formatMachineCode(hex string) string {
	var groups []string
	for i := 0; i < len(hex); i += 4 {
		end := i + 4
		if end > len(hex) {
			end = len(hex)
		}
		groups = append(groups, strings.ToUpper(hex[i:end]))
	}
	return strings.Join(groups, "-")
}

func getMACAddress() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp != 0 && len(iface.HardwareAddr) >= 6 {
			return iface.HardwareAddr.String()
		}
	}
	return ""
}
