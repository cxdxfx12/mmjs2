package license

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"strings"
)

// LicenseInfo 授权信息
type LicenseInfo struct {
	MachineCode string `json:"machine_code"`
	Customer    string `json:"customer_name"`
	ExpiresAt   string `json:"expires_at"`
	IssuedAt    string `json:"issued_at"`
	DaysLeft    int    `json:"days_left"`
	IsValid     bool   `json:"is_valid"`
}

// LicensePayload 加密前的原始数据
type LicensePayload struct {
	Version     int    `json:"version"`
	MachineCode string `json:"machine_code"`
	Customer    string `json:"customer_name"`
	IssuedAt    string `json:"issued_at"`
	ExpiresAt   string `json:"expires_at"`
	DurationDays int   `json:"duration_days"`
	Features    []string `json:"features"`
	Nonce       string  `json:"nonce"`
}

// AES密钥(32字节)和RSA公钥 — 硬编码在二进制中
// ⚠️ 生产环境请更换这些密钥！
var (
	// AES-256 key (编译后难以提取)
	aesKey = []byte{
		0x7a, 0x3b, 0x95, 0x1c, 0x4e, 0x62, 0xaf, 0x8d,
		0x2f, 0x51, 0xdc, 0x73, 0x1a, 0x88, 0x6b, 0x0e,
		0x39, 0x47, 0xb5, 0xe2, 0x14, 0x9d, 0x26, 0xf8,
		0x5c, 0x03, 0xea, 0x67, 0x90, 0x1f, 0xd4, 0xab,
	}

	// RSA公钥PEM (仅验证用，私钥在服务器)
	rsaPublicKeyPEM = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA0m1ORjfDoVg78T5yOQx4
CHuQYtqLmI+stMM42gxtEzl8xDC9OCBTZjkcOyUekisYF5TEvPXSsPAgvoHC+G4y
nXdCWzTFeSWxknZtzebQahNlbm+TFEkpY6e85qBGsxC7d5tnOieCtCkgz1X6CB7o
Lq3YokUw/xpYxonMJNdhlhp+YauV8X1a29KwV7rLJcrZ/01gL3mIEyOvCXpkPrSM
uh+mUDp63kzL5Hx+Fd4OLdejdDw2IhPZnwI175PYo5nylv/IxAQOZIDbDHsAegYs
0lWKZubI7KC+hk+P1YHjzKO0b8WIgpzEtrfvBgbP9QDUquNQnOPgVHaWTEeWYWy+
ZwIDAQAB
-----END PUBLIC KEY-----`

	publicKey *rsa.PublicKey
)

func init() {
	block, _ := pem.Decode([]byte(rsaPublicKeyPEM))
	if block != nil {
		pk, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err == nil {
			publicKey = pk.(*rsa.PublicKey)
		}
	}
}

// EncryptLicense 加密License (服务器端使用)
func EncryptLicense(payload LicensePayload, privateKeyPEM string) (string, error) {
	data, _ := json.Marshal(payload)

	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return "", fmt.Errorf("无效的私钥")
	}
	privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}

	// 使用 crypto.SHA256 与 PHP openssl_sign(OPENSSL_ALGO_SHA256) 对齐
	sig, err := rsa.SignPKCS1v15(rand.Reader, privKey, crypto.SHA256, data)
	if err != nil {
		return "", err
	}

	// 组合: payload + signature
	combined := append(data, sig...)

	// AES-GCM 加密
	blockCipher, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(blockCipher)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	encrypted := gcm.Seal(nonce, nonce, combined, nil)
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// DecryptLicense 解密并验证License (客户端使用)
func DecryptLicense(b64 string) (*LicensePayload, error) {
	encrypted, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("base64解码失败")
	}

	blockCipher, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(blockCipher)
	if err != nil {
		return nil, err
	}
	if len(encrypted) < gcm.NonceSize() {
		return nil, fmt.Errorf("数据太短")
	}
	nonce := encrypted[:gcm.NonceSize()]
	ciphertext := encrypted[gcm.NonceSize():]
	combined, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("解密失败: 授权文件无效")
	}

	// 分离 payload 和 signature
	if len(combined) < 256 {
		return nil, fmt.Errorf("数据格式错误")
	}
	payloadData := combined[:len(combined)-256]
	signature := combined[len(combined)-256:]

	// 验证签名
	if publicKey != nil {
		hash := sha256.Sum256(payloadData)
		err = rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hash[:], signature)
		if err != nil {
			return nil, fmt.Errorf("签名验证失败: 授权文件可能被篡改")
		}
	}

	var payload LicensePayload
	if err := json.Unmarshal(payloadData, &payload); err != nil {
		return nil, fmt.Errorf("JSON解析失败")
	}
	return &payload, nil
}

// SaveLicenseFile 保存授权文件到磁盘
func SaveLicenseFile(dataDir, b64 string) error {
	return os.WriteFile(fmt.Sprintf("%s/license.dat", dataDir), []byte(b64), 0644)
}

// ReadLicenseFile 读取本地授权文件
func ReadLicenseFile(dataDir string) (string, error) {
	data, err := os.ReadFile(dataDir + "/license.dat")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
