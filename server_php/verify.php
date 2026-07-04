<?php
/**
 * 授权验证接口 (RSA+AES加密版)
 * POST /yunfei_api/verify.php
 * 
 * 支持两种验证方式：
 * 1. 客户端提交 license_data → 服务端解密密验证 (推荐)
 * 2. 客户端提交 machine_code → 服务端查库确认
 */

header('Content-Type: application/json; charset=utf-8');
header('Access-Control-Allow-Origin: *');
header('Access-Control-Allow-Headers: Content-Type');

if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') exit;

// ==================== 数据库配置 ====================
$DB_HOST = '127.0.0.1';
$DB_PORT = '3306';
$DB_USER = 'root';
$DB_PASS = 'cxdxfx12';
$DB_NAME = 'dasheng';

// ==================== AES 密钥 (与客户端一致) ====================
define('AES_KEY_HEX', '7a3b951c4e62af8d2f51dc731a886b0e3947b5e2149d26f85c03ea67901fd4ab');

// ==================== RSA 公钥 (用于验证签名) ====================
define('RSA_PUBLIC_KEY', '-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA0m1ORjfDoVg78T5yOQx4
CHuQYtqLmI+stMM42gxtEzl8xDC9OCBTZjkcOyUekisYF5TEvPXSsPAgvoHC+G4y
nXdCWzTFeSWxknZtzebQahNlbm+TFEkpY6e85qBGsxC7d5tnOieCtCkgz1X6CB7o
Lq3YokUw/xpYxonMJNdhlhp+YauV8X1a29KwV7rLJcrZ/01gL3mIEyOvCXpkPrSM
uh+mUDp63kzL5Hx+Fd4OLdejdDw2IhPZnwI175PYo5nylv/IxAQOZIDbDHsAegYs
0lWKZubI7KC+hk+P1YHjzKO0b8WIgpzEtrfvBgbP9QDUquNQnOPgVHaWTEeWYWy+
ZwIDAQAB
-----END PUBLIC KEY-----');

function aesKey(): string {
    return hex2bin(AES_KEY_HEX);
}

function response($data) { echo json_encode($data, JSON_UNESCAPED_UNICODE); exit; }

/**
 * 解密并验证授权数据
 * 流程与客户端 crypto.go DecryptLicense() 一致：
 * Base64解码 → AES-256-GCM解密 → 分离payload和签名 → RSA公钥验签 → 解析JSON
 */
function decryptAndVerify($b64) {
    $encrypted = base64_decode($b64);
    if ($encrypted === false) return [false, 'Base64解码失败'];

    // AES-256-GCM 解密
    if (strlen($encrypted) < 12 + 16) return [false, '数据太短']; // nonce(12) + tag(16) 至少
    $nonce = substr($encrypted, 0, 12);
    $tag = substr($encrypted, -16);
    $ciphertext = substr($encrypted, 12, -16);

    $key = aesKey();
    $plaintext = openssl_decrypt($ciphertext, 'aes-256-gcm', $key, OPENSSL_RAW_DATA, $nonce, $tag);
    if ($plaintext === false) return [false, 'AES解密失败：授权数据无效'];

    // 分离 payload 和 signature (RSA-2048签名 = 256字节)
    if (strlen($plaintext) < 256) return [false, '数据格式错误'];
    $payloadData = substr($plaintext, 0, -256);
    $signature = substr($plaintext, -256);

    // RSA 公钥验签
    $pubKey = openssl_pkey_get_public(RSA_PUBLIC_KEY);
    $verify = openssl_verify($payloadData, $signature, $pubKey, OPENSSL_ALGO_SHA256);
    if ($verify !== 1) return [false, 'RSA签名验证失败：授权文件可能被篡改'];

    $payload = json_decode($payloadData, true);
    if (!$payload) return [false, 'JSON解析失败'];

    // 检查是否过期
    $expiresAt = $payload['expires_at'] ?? '';
    if ($expiresAt) {
        $expireTime = strtotime($expiresAt);
        if ($expireTime && $expireTime < time()) {
            return [false, '授权已过期', $payload];
        }
    }

    return [true, '验证通过', $payload];
}

try {
    $input = json_decode(file_get_contents('php://input'), true);
    if (!$input) response(['valid' => false, 'error' => '参数错误']);

    $licenseData = trim($input['license_data'] ?? '');
    $machineCode = trim($input['machine_code'] ?? '');

    // ========== 方式1: 通过加密授权数据验证 (推荐) ==========
    if ($licenseData) {
        list($ok, $msg, $payload) = decryptAndVerify($licenseData);

        if (!$ok) {
            response(['valid' => false, 'error' => $msg]);
        }

        // 验证机器码是否匹配
        if ($machineCode && $payload['machine_code'] !== $machineCode) {
            response(['valid' => false, 'error' => '机器码不匹配']);
        }

        // 同时查库确认未被吊销
        $pdo = new PDO("mysql:host=$DB_HOST;port=$DB_PORT;dbname=$DB_NAME;charset=utf8mb4", $DB_USER, $DB_PASS);
        $stmt = $pdo->prepare(
            "SELECT id FROM yunfei_licenses WHERE machine_code=? AND status=2 LIMIT 1"
        );
        $stmt->execute([$payload['machine_code']]);
        if (!$stmt->fetch()) {
            response(['valid' => false, 'error' => '授权已被吊销']);
        }

        $daysLeft = 0;
        if (!empty($payload['expires_at'])) {
            $expire = new DateTime($payload['expires_at']);
            $now = new DateTime();
            if ($expire > $now) {
                $daysLeft = (int)$now->diff($expire)->days;
            }
        }

        response([
            'valid' => true,
            'customer_name' => $payload['customer_name'] ?? '',
            'expires_at' => $payload['expires_at'] ?? '',
            'days_left' => $daysLeft,
            'encrypted' => true,
        ]);
    }

    // ========== 方式2: 仅通过机器码查库 (兼容旧版) ==========
    if ($machineCode) {
        $pdo = new PDO("mysql:host=$DB_HOST;port=$DB_PORT;dbname=$DB_NAME;charset=utf8mb4", $DB_USER, $DB_PASS);
        $stmt = $pdo->prepare(
            "SELECT license_key, customer_name, expires_at, license_data FROM yunfei_licenses
             WHERE machine_code = ? AND status = 2 AND expires_at >= CURDATE()
             LIMIT 1"
        );
        $stmt->execute([$machineCode]);
        $row = $stmt->fetch(PDO::FETCH_ASSOC);

        if ($row) {
            $pdo->prepare("UPDATE yunfei_licenses SET last_check_at = NOW() WHERE license_key = ?")
                ->execute([$row['license_key']]);

            $daysLeft = (new DateTime($row['expires_at']))->diff(new DateTime())->days;
            response([
                'valid' => true,
                'customer_name' => $row['customer_name'],
                'expires_at' => $row['expires_at'],
                'days_left' => $daysLeft,
                'license_data' => $row['license_data'] ?? '',  // 同步最新加密授权数据
            ]);
        } else {
            response(['valid' => false, 'error' => '未找到有效授权']);
        }
    }

    response(['valid' => false, 'error' => '缺少参数：请提供 license_data 或 machine_code']);

} catch (Exception $e) {
    response(['valid' => false, 'error' => $e->getMessage()]);
}
