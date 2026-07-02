package license

import (
	"fmt"
	"os"
	"time"
	"yunfei/internal/db"
)

// VerifyLicense 验证授权是否有效
func VerifyLicense(forceNetwork bool) *LicenseInfo {
	machineCode := GetMachineCode()

	info := &LicenseInfo{
		MachineCode: machineCode,
		IsValid:     false,
	}

	var licenseRaw string
	var expiresAt string
	db.DB.QueryRow("SELECT license_raw, expires_at FROM license_info WHERE id=1").Scan(&licenseRaw, &expiresAt)

	if licenseRaw == "" {
		return info
	}

	payload, err := DecryptLicense(licenseRaw)
	if err != nil {
		return info
	}

	if payload.MachineCode != machineCode {
		return info
	}

	expTime, err := time.Parse(time.RFC3339, payload.ExpiresAt)
	if err != nil {
		return info
	}
	now := time.Now()
	daysLeft := int(expTime.Sub(now).Hours() / 24)

	if daysLeft < 0 {
		return info
	}

	info.IsValid = true
	info.Customer = payload.Customer
	info.ExpiresAt = payload.ExpiresAt
	info.IssuedAt = payload.IssuedAt
	info.DaysLeft = daysLeft

	db.DB.Exec("UPDATE license_info SET last_verify_at=datetime('now','localtime') WHERE id=1")
	return info
}

// ImportLicense 导入授权文件
func ImportLicense(b64 string) (bool, string) {
	machineCode := GetMachineCode()

	payload, err := DecryptLicense(b64)
	if err != nil {
		return false, err.Error()
	}

	if payload.MachineCode != machineCode {
		return false, "授权文件的机器码与本机不匹配"
	}

	expTime, err := time.Parse(time.RFC3339, payload.ExpiresAt)
	if err != nil {
		return false, "授权日期格式错误"
	}
	if time.Now().After(expTime) {
		return false, "授权已过期"
	}

	db.DB.Exec(`INSERT OR REPLACE INTO license_info (id, machine_code, customer_name, expires_at, issued_at, features, license_raw, last_verify_at)
		VALUES (1, ?, ?, ?, ?, ?, ?, datetime('now','localtime'))`,
		machineCode, payload.Customer, payload.ExpiresAt, payload.IssuedAt,
		"freight_calc,report_export", b64)

	return true, ""
}

func getDataDir() string {
	if runtimeVar := os.Getenv("APPDATA"); runtimeVar != "" {
		return runtimeVar + "/yunfei"
	}
	return fmt.Sprintf("%s/.yunfei", os.Getenv("HOME"))
}
