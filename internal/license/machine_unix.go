//go:build !windows
// +build !windows

package license

func getCPUSerial() string {
	return ""
}

func getDiskSerial() string {
	return ""
}
