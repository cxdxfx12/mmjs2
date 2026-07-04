<?php
/**
 * 激活授权接口 (RSA+AES加密版)
 * POST /yunfei_api/activate.php
 * 
 * 新流程：客户端提交管理员签发的加密授权数据，服务端验签后激活
 * Body: { "license_data": "加密的base64字符串", "machine_code": "xxxx-xxxx-...", "sign": "MD5签名", "ts": 时间戳 }
 */

header('Content-Type: application/json; charset=utf-8');
header('Access-Control-Allow-Origin: *');
header('Access-Control-Allow-Headers: Content-Type');

if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') exit;

// ==================== 配置 ====================
$DB_HOST = '127.0.0.1';
$DB_PORT = '3306';
$DB_USER = 'root';
$DB_PASS = 'cxdxfx12';
$DB_NAME = 'dasheng';

define('AES_KEY_HEX', '7a3b951c4e62af8d2f51dc731a886b0e3947b5e2149d26f85c03ea67901fd4ab');
define('RSA_PUBLIC_KEY', '-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA0m1ORjfDoVg78T5yOQx4
CHuQYtqLmI+stMM42gxtEzl8xDC9OCBTZjkcOyUekisYF5TEvPXSsPAgvoHC+G4y
nXdCWzTFeSWxknZtzebQahNlbm+TFEkpY6e85qBGsxC7d5tnOieCtCkgz1X6CB7o
Lq3YokUw/xpYxonMJNdhlhp+YauV8X1a29KwV7rLJcrZ/01gL3mIEyOvCXpkPrSM
uh+mUDp63kzL5Hx+Fd4OLdejdDw2IhPZnwI175PYo5nylv/IxAQOZIDbDHsAegYs
0lWKZubI7KC+hk+P1YHjzKO0b8WIgpzEtrfvBgbP9QDUquNQnOPgVHaWTEeWYWy+
ZwIDAQAB
-----END PUBLIC KEY-----');

function aesKey(): string { return hex2bin(AES_KEY_HEX); }
function response($data) { echo json_encode($data, JSON_UNESCAPED_UNICODE); exit; }

/**
 * 解密并验证授权数据 (与客户端 crypto.go 一致)
 */
function decryptAndVerify($b64) {
    $encrypted = base64_decode($b64);
    if ($encrypted === false) return [false, 'Base64解码失败'];

    if (strlen($encrypted) < 12 + 16) return [false, '数据太短'];
    $nonce = substr($encrypted, 0, 12);
    $tag = substr($encrypted, -16);
    $ciphertext = substr($encrypted, 12, -16);

    $plaintext = openssl_decrypt($ciphertext, 'aes-256-gcm', aesKey(), OPENSSL_RAW_DATA, $nonce, $tag);
    if ($plaintext === false) return [false, 'AES解密失败：授权数据无效'];

    if (strlen($plaintext) < 256) return [false, '数据格式错误'];
    $payloadData = substr($plaintext, 0, -256);
    $signature = substr($plaintext, -256);

    $pubKey = openssl_pkey_get_public(RSA_PUBLIC_KEY);
    $verify = openssl_verify($payloadData, $signature, $pubKey, OPENSSL_ALGO_SHA256);
    if ($verify !== 1) return [false, '签名验证失败：授权文件可能被篡改'];

    $payload = json_decode($payloadData, true);
    if (!$payload) return [false, 'JSON解析失败'];

    // 检查过期
    $expiresAt = $payload['expires_at'] ?? '';
    if ($expiresAt) {
        if (strtotime($expiresAt) < time()) {
            return [false, '授权已过期', $payload];
        }
    }

    return [true, '验证通过', $payload];
}

/**
 * API签名验证 (兼容旧版)
 */
function getSecret($pdo) {
    $stmt = $pdo->query("SELECT `value` FROM yunfei_settings WHERE `key`='api_secret'");
    $row = $stmt->fetch(PDO::FETCH_ASSOC);
    return $row ? $row['value'] : '';
}

function checkSign($data, $sign, $ts, $secret) {
    if (abs(time() - $ts) > 300) return false;
    return hash_equals(md5($data . '|' . $ts . '|' . $secret), $sign);
}

try {
    $input = json_decode(file_get_contents('php://input'), true);
    if (!$input) response(['ok' => false, 'msg' => '参数错误']);

    $licenseData = trim($input['license_data'] ?? '');
    $machineCode = trim($input['machine_code'] ?? '');
    $sign = $input['sign'] ?? '';
    $ts = intval($input['ts'] ?? 0);

    if (!$licenseData) response(['ok' => false, 'msg' => '缺少授权数据(license_data)']);
    if (!$machineCode) response(['ok' => false, 'msg' => '缺少机器码(machine_code)']);

    $pdo = new PDO("mysql:host=$DB_HOST;port=$DB_PORT;dbname=$DB_NAME;charset=utf8mb4", $DB_USER, $DB_PASS);

    // 签名验证 (防止恶意激活)
    $secret = getSecret($pdo);
    $signData = $licenseData . '|' . $machineCode;
    if ($sign && $ts) {
        if (!checkSign($signData, $sign, $ts, $secret)) {
            response(['ok' => false, 'msg' => '签名验证失败']);
        }
    } else {
        // 没有签名时，至少用简化验证
        if (strlen($licenseData) < 100) {
            response(['ok' => false, 'msg' => '授权数据无效(太短)']);
        }
    }

    // 解密并验证授权数据
    list($ok, $msg, $payload) = decryptAndVerify($licenseData);
    if (!$ok) {
        response(['ok' => false, 'msg' => $msg]);
    }

    // 验证机器码
    if ($payload['machine_code'] !== $machineCode) {
        response(['ok' => false, 'msg' => '机器码不匹配，此授权文件不是为本电脑签发的']);
    }

    // 检查是否已有有效授权
    $stmt = $pdo->prepare("SELECT id, status FROM yunfei_licenses WHERE machine_code=? AND status=2 AND expires_at >= CURDATE()");
    $stmt->execute([$machineCode]);
    $existing = $stmt->fetch(PDO::FETCH_ASSOC);
    if ($existing) {
        // 已有有效授权，更新 license_data
        $pdo->prepare("UPDATE yunfei_licenses SET license_data=?, last_check_at=NOW() WHERE id=?")
            ->execute([$licenseData, $existing['id']]);
        $daysLeft = (new DateTime($payload['expires_at']))->diff(new DateTime())->days;
        response([
            'ok' => true,
            'msg' => '授权已更新',
            'customer_name' => $payload['customer_name'],
            'expires_at' => $payload['expires_at'],
            'days_left' => $daysLeft,
        ]);
    }

    // 新激活：查旧版激活码记录或新建
    $stmt = $pdo->prepare("SELECT id FROM yunfei_licenses WHERE machine_code=? AND status=1 LIMIT 1");
    $stmt->execute([$machineCode]);
    $row = $stmt->fetch(PDO::FETCH_ASSOC);

    $expiresDate = date('Y-m-d', strtotime($payload['expires_at']));
    $daysLeft = (new DateTime($payload['expires_at']))->diff(new DateTime())->days;

    if ($row) {
        // 更新旧记录
        $pdo->prepare(
            "UPDATE yunfei_licenses SET license_data=?, customer_name=?, expires_at=?, issued_at=NOW(), duration_days=?, status=2, activated_at=NOW(), last_check_at=NOW() WHERE id=?"
        )->execute([$licenseData, $payload['customer_name'], $expiresDate, $payload['duration_days'] ?? 365, $row['id']]);
    } else {
        // 新建记录
        $pdo->prepare(
            "INSERT INTO yunfei_licenses (machine_code, customer_name, expires_at, license_data, issued_at, duration_days, status, activated_at, last_check_at)
             VALUES (?, ?, ?, ?, NOW(), ?, 2, NOW(), NOW())"
        )->execute([$machineCode, $payload['customer_name'], $expiresDate, $licenseData, $payload['duration_days'] ?? 365]);
    }

    response([
        'ok' => true,
        'msg' => '激活成功',
        'customer_name' => $payload['customer_name'],
        'expires_at' => $payload['expires_at'],
        'days_left' => $daysLeft,
    ]);

} catch (Exception $e) {
    response(['ok' => false, 'msg' => $e->getMessage()]);
}
