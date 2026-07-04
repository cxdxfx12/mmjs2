package license

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"yunfei/internal/db"
)

// 默认服务器地址
const defaultServerURL = "http://www.hbdxm.com/yunfei_api"

// 默认 API 密钥（与服务器 yunfei_settings 表 api_secret 一致，可运行时覆盖）
const defaultApiSecret = "yunfei_server_2024_!@#"

// getApiSecret 从本地 app_settings 读取，若无则用默认值
func getApiSecret() string {
	var v string
	err := db.DB.QueryRow("SELECT value FROM app_settings WHERE key='api_secret'").Scan(&v)
	if err != nil || v == "" {
		return defaultApiSecret
	}
	return v
}

// SetApiSecret 设置 API 密钥（持久化到本地数据库）
func SetApiSecret(secret string) {
	db.DB.Exec(`INSERT OR REPLACE INTO app_settings (key, value) VALUES ('api_secret', ?)`, secret)
}

// GetApiSecret 获取当前 API 密钥
func GetApiSecret() string {
	return getApiSecret()
}

// 当前服务器地址（支持运行时可配）
var serverURL = ""
var backupServerURL = ""

// initServerURLs 从数据库读取主/备网址，若无则用默认值
func initServerURLs() {
	serverURL = loadURLSetting("license_server_url", defaultServerURL)
	backupServerURL = loadURLSetting("license_backup_url", "")
}

func loadURLSetting(key, fallback string) string {
	var v string
	err := db.DB.QueryRow("SELECT value FROM app_settings WHERE key=?", key).Scan(&v)
	if err != nil || v == "" {
		return fallback
	}
	return strings.TrimRight(v, "/")
}

func saveURLSetting(key, url string) {
	url = strings.TrimRight(url, "/")
	db.DB.Exec(`INSERT OR REPLACE INTO app_settings (key, value) VALUES (?, ?)`, key, url)
}

type OnlineLicenseInfo struct {
	Valid        bool   `json:"valid"`
	CustomerName string `json:"customer_name"`
	ExpiresAt    string `json:"expires_at"`
	DaysLeft     int    `json:"days_left"`
	LicenseData  string `json:"license_data"` // 服务器返回的最新加密授权数据（续期后同步用）
	Error        string `json:"error"`
}

type ActivateResult struct {
	OK           bool   `json:"ok"`
	Msg          string `json:"msg"`
	CustomerName string `json:"customer_name"`
	ExpiresAt    string `json:"expires_at"`
	DaysLeft     int    `json:"days_left"`
}

// makeSign 生成请求签名
func makeSign(data string) (string, int64) {
	ts := time.Now().Unix()
	raw := fmt.Sprintf("%s|%d|%s", data, ts, getApiSecret())
	sign := fmt.Sprintf("%x", md5.Sum([]byte(raw)))
	return sign, ts
}

// SetServerURL 设置主授权服务器地址
func SetServerURL(url string) {
	if url != "" {
		serverURL = strings.TrimRight(url, "/")
		saveURLSetting("license_server_url", serverURL)
	}
}

// SetBackupServerURL 设置备用授权服务器地址
func SetBackupServerURL(url string) {
	backupServerURL = strings.TrimRight(url, "/")
	saveURLSetting("license_backup_url", backupServerURL)
}

// GetServerURLs 返回主备地址
func GetServerURLs() (primary, backup string) {
	// 确保已初始化
	if serverURL == "" {
		initServerURLs()
	}
	return serverURL, backupServerURL
}

// tryPost 依次尝试主、备地址发起 POST，成功返回响应体
func tryPost(endpoint, body string) ([]byte, error) {
	if serverURL == "" {
		initServerURLs()
	}
	urls := []string{}
	if serverURL != "" {
		urls = append(urls, serverURL+endpoint)
	}
	if backupServerURL != "" && backupServerURL != serverURL {
		urls = append(urls, backupServerURL+endpoint)
	}
	if len(urls) == 0 {
		return nil, fmt.Errorf("未配置授权服务器地址")
	}

	var lastErr error
	for _, u := range urls {
		resp, err := http.Post(u, "application/json", strings.NewReader(body))
		if err != nil {
			lastErr = err
			continue
		}
		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return data, nil
	}
	return nil, lastErr
}

// CheckOnlineLicense 在线验证授权（主失效自动切备）
func CheckOnlineLicense(machineCode string) *OnlineLicenseInfo {
	result := &OnlineLicenseInfo{Valid: false}

	sign, ts := makeSign(machineCode)
	body := fmt.Sprintf(`{"machine_code":"%s","sign":"%s","ts":%d}`, machineCode, sign, ts)

	data, err := tryPost("/verify.php", body)
	if err != nil {
		result.Error = "无法连接授权服务器: " + err.Error()
		return result
	}

	json.Unmarshal(data, result)

	// 缓存验证结果到本地（7天有效）
	if result.Valid {
		cacheOnlineResult(machineCode, result)
		// 同步服务器最新到期时间和加密授权数据到本地 license_info
		syncLicenseInfo(result)
	}

	return result
}

// ActivateOnline 在线激活授权码（旧版YF激活码，保留兼容）
func ActivateOnline(licenseKey, machineCode string) *ActivateResult {
	result := &ActivateResult{OK: false}

	// 服务器 strtoupper 处理，客户端也需统一大写
	licenseKey = strings.ToUpper(strings.TrimSpace(licenseKey))

	signData := licenseKey + "|" + machineCode
	sign, ts := makeSign(signData)
	body := fmt.Sprintf(`{"license_key":"%s","machine_code":"%s","sign":"%s","ts":%d}`,
		licenseKey, machineCode, sign, ts)

	data, err := tryPost("/activate.php", body)
	if err != nil {
		result.Msg = "无法连接授权服务器: " + err.Error()
		return result
	}

	json.Unmarshal(data, result)

	// 激活成功后缓存
	if result.OK {
		info := &OnlineLicenseInfo{
			Valid:        true,
			CustomerName: result.CustomerName,
			ExpiresAt:    result.ExpiresAt,
			DaysLeft:     result.DaysLeft,
		}
		cacheOnlineResult(machineCode, info)
	}

	return result
}

// ActivateWithLicenseData 使用加密授权数据激活（新版 RSA+AES 方式）
// licenseData: 管理员在后台签发的加密授权数据（Base64字符串）
// machineCode: 当前电脑的机器码
func ActivateWithLicenseData(licenseData, machineCode string) *ActivateResult {
	result := &ActivateResult{OK: false}

	// ⚠️ 先本地验证，防止无效数据发到服务器
	payload, err := DecryptLicense(licenseData)
	if err != nil {
		result.Msg = "授权数据无效: " + err.Error()
		return result
	}
	if payload.MachineCode != machineCode {
		result.Msg = "机器码不匹配，此授权文件不是为本电脑签发的"
		return result
	}

	// 生成本地签名并提交到服务器注册
	signData := licenseData + "|" + machineCode
	sign, ts := makeSign(signData)

	// 对 JSON 中的特殊字符做转义
	escaped := strings.ReplaceAll(licenseData, `"`, `\"`)
	escaped = strings.ReplaceAll(escaped, "\n", "\\n")
	escaped = strings.ReplaceAll(escaped, "\r", "\\r")
	body := fmt.Sprintf(`{"license_data":"%s","machine_code":"%s","sign":"%s","ts":%d}`,
		escaped, machineCode, sign, ts)

	data, err := tryPost("/activate.php", body)
	if err != nil {
		result.Msg = "无法连接授权服务器: " + err.Error()
		return result
	}

	json.Unmarshal(data, result)

	// 激活成功后缓存
	if result.OK {
		// 保存加密授权数据到本地
		saveEncryptedLicense(licenseData)
		info := &OnlineLicenseInfo{
			Valid:        true,
			CustomerName: result.CustomerName,
			ExpiresAt:    result.ExpiresAt,
			DaysLeft:     result.DaysLeft,
		}
		cacheOnlineResult(machineCode, info)
	}

	return result
}

// saveEncryptedLicense 将加密授权数据保存到本地数据库
func saveEncryptedLicense(licenseData string) {
	db.DB.Exec(`INSERT OR REPLACE INTO app_settings (key, value) VALUES ('encrypted_license_data', ?)`, licenseData)
}

// cachedResult 带时间戳的缓存结构
type cachedResult struct {
	OnlineLicenseInfo
	MachineCode string `json:"machine_code"`
	CachedAt    string `json:"cached_at"`
}

// GetCachedOnlineLicense 获取本地缓存的在线验证结果（7天内有效）
func GetCachedOnlineLicense(machineCode string) *OnlineLicenseInfo {
	var raw string
	err := db.DB.QueryRow(
		"SELECT value FROM app_settings WHERE key='online_license_cache'",
	).Scan(&raw)
	if err != nil || raw == "" {
		return nil
	}

	var cached cachedResult
	if json.Unmarshal([]byte(raw), &cached) != nil {
		return nil
	}

	// 检查缓存是否过期（7天）
	if cached.CachedAt != "" {
		t, err := time.Parse("2006-01-02 15:04:05", cached.CachedAt)
		if err == nil && time.Since(t) > 7*24*time.Hour {
			return nil
		}
	}

	// 验证机器码是否匹配
	if cached.MachineCode != machineCode {
		return nil
	}

	return &cached.OnlineLicenseInfo
}

// syncLicenseInfo 将服务器返回的最新到期时间/加密授权数据同步到本地 license_info 表
// 只有当服务器到期时间晚于本地时才更新（防止回滚）
func syncLicenseInfo(info *OnlineLicenseInfo) {
	if info == nil || !info.Valid || info.ExpiresAt == "" {
		return
	}

	// 读取本地当前到期时间
	var localExpire string
	err := db.DB.QueryRow("SELECT expires_at FROM license_info LIMIT 1").Scan(&localExpire)
	if err != nil {
		return // 本地没有授权记录，无需同步
	}

	// 只有服务器到期时间晚于本地时才更新
	if localExpire >= info.ExpiresAt {
		return
	}

	// 更新到期时间
	if info.LicenseData != "" {
		// 同时更新加密授权数据（续期后 license_data 含新的过期时间）
		db.DB.Exec(`UPDATE license_info SET expires_at=?, license_raw=?, updated_at=datetime('now') WHERE id=(SELECT id FROM license_info LIMIT 1)`,
			info.ExpiresAt, info.LicenseData)
	} else {
		db.DB.Exec(`UPDATE license_info SET expires_at=?, updated_at=datetime('now') WHERE id=(SELECT id FROM license_info LIMIT 1)`,
			info.ExpiresAt)
	}
}

// cacheOnlineResult 缓存在线验证结果到本地
func cacheOnlineResult(machineCode string, info *OnlineLicenseInfo) {
	cached := cachedResult{
		OnlineLicenseInfo: *info,
		MachineCode:       machineCode,
		CachedAt:          time.Now().Format("2006-01-02 15:04:05"),
	}
	data, _ := json.Marshal(cached)

	db.DB.Exec(`INSERT OR REPLACE INTO app_settings (key, value) VALUES ('online_license_cache', ?)`, string(data))
}
