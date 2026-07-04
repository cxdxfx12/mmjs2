<?php
/**
 * 喵喵云结算 - 授权管理 API
 * 部署到现有 ThinkPHP 服务器即可，也可独立运行
 * 数据库表自动创建（首次访问时）
 */

header('Content-Type: application/json; charset=utf-8');
header('Access-Control-Allow-Origin: *');
header('Access-Control-Allow-Methods: GET,POST,OPTIONS');
header('Access-Control-Allow-Headers: Content-Type,Authorization');

if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') { http_response_code(200); exit; }

// ==================== 配置 ====================
define('DB_HOST', '127.0.0.1');
define('DB_PORT', '3307');
define('DB_USER', 'root');
define('DB_PASS', 'cxdxfx12');
define('DB_NAME', 'yunfei');

// AES-256 密钥 (32字节，与客户端 crypto.go 中完全一致)
define('AES_KEY_HEX', '7a3b951c4e62af8d2f51dc731a886b0e3947b5e2149d26f85c03ea67901fd4ab');

// RSA 私钥 (仅服务器持有，用于签名授权文件)
define('RSA_PRIVATE_KEY', '-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEA0m1ORjfDoVg78T5yOQx4CHuQYtqLmI+stMM42gxtEzl8xDC9
OCBTZjkcOyUekisYF5TEvPXSsPAgvoHC+G4ynXdCWzTFeSWxknZtzebQahNlbm+T
FEkpY6e85qBGsxC7d5tnOieCtCkgz1X6CB7oLq3YokUw/xpYxonMJNdhlhp+YauV
8X1a29KwV7rLJcrZ/01gL3mIEyOvCXpkPrSMuh+mUDp63kzL5Hx+Fd4OLdejdDw2
IhPZnwI175PYo5nylv/IxAQOZIDbDHsAegYs0lWKZubI7KC+hk+P1YHjzKO0b8WI
gpzEtrfvBgbP9QDUquNQnOPgVHaWTEeWYWy+ZwIDAQABAoIBAGVXXBOoJpDNSC0S
iHseHK7lJ2/cVc+XHjN+M6KbymowTPzFhlOCCfhYt9ZqNZNqqrMspCVm9F3ff72Q
C+COXdUFSxFW1GXYd/EUFSTNLQFxLu/lTz29UHAcp/agKPxzKf3b+3Z/8cwnZJRG
EWEY1XQpqIPZ89NgEnIniggSLH7Xii/o+pYgsEPbMSrwQGshAnEcTqMMxAgtRYej
4FJh++inFWDDZEIqpUu0WF9pykfbaDEI8I1ZFUqtiLs1LhTwUNj18m+ku8HVz0jm
vJjeoigdi6nVWXtBdtrzNZMwPbvnPwXXxPm8xIyyz1Pk4zihyi4bw6hLTZVpTFWN
DgV0WPkCgYEA1Mh4aCPezRq6I+LtOBTN/v0eayM3jg/VJTseY65YfU2YAUHAZAmp
gi0xO8nEUkIIii4klnOXI4nnZOhDtWGibdNfWADCU5aMFh7S56OCljcacCxzYKcL
ctNV1mU+Q561UVYUFDREpOEj/APZzm8KL9u89Hg1azui0hpdKz0OOikCgYEA/SpU
r16o/knZlF4Mb18NM9JiSPTKJlK7+hTx9vubgV1YqYPFN1SaqY41BbgPKPJP+OdZ
D8zXUBX7bZhecnF/XZgsnieTLyNyzKLTS68rlsdbRaSRzeuy4A5eVOelU3k6k844
DlgOGAtG9/RZWvlegmMrhTZZ83aCS2eeajJzZg8CgYAaZ+pxWgo7P9bbvzybGhHa
VAUjXJJ3YcWcwjJqQmee3TNA7Kz4fS55Biy287oWTzWKGGHX/e5Crcl2f1BvwPcM
VA/f9vAmuWcXE6ourt70z0/LneiPlQtZq5paaeQJNjfgKSOCwl3GbF4v1zZ/ZM5J
1CYl3IkcjqENG9J2HDSYyQKBgQDHkC/TUe4rDXHrR7vLqwTQPd5mHjifvwYY25vl
Em+BqWCzt4Cl3hZQ5B2d1Xp1z5UE4vFMyC9OHRXmTX7d/ePllohNX2rhdLMQ5qVi
+sGEiL/FBTY+Obb2cb0gdr3XMC/hxWRgwj7R60nVOZOaaAp9A8mRp8d+aIPLBvJU
SlK3NwKBgF3VmgixiL22v8nHVt3gZH4Bt/28p0FFC2pMkIE9RiTRKLbTSs80UTkr
+cJwXVrZLDrwxCzkfzKZ0I5uCLr2JuOsJF4st2bEFIafxWbdp1eGGdWnrVALy8MU
IiuE8/aOT4eGLr+uBrrE8GLJXgP3CJpFQ52VFxSeJ5I5DGGEBJSs
-----END RSA PRIVATE KEY-----');

// 管理密码（SHA256哈希）
define('ADMIN_PASSWORD_HASH', 'a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3'); // 默认: 123

// ==================== 数据库初始化 ====================
function getDB(): PDO {
    static $pdo = null;
    if ($pdo === null) {
        $dsn = 'mysql:host=' . DB_HOST . ';port=' . DB_PORT . ';dbname=' . DB_NAME . ';charset=utf8mb4';
        $pdo = new PDO($dsn, DB_USER, DB_PASS, [
            PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION,
            PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC,
        ]);
        // 自动建表
        $pdo->exec("CREATE TABLE IF NOT EXISTS yunfei_licenses (
            id INT AUTO_INCREMENT PRIMARY KEY,
            machine_code VARCHAR(64) NOT NULL COMMENT '机器码',
            customer_name VARCHAR(100) NOT NULL COMMENT '客户名称',
            contact_info VARCHAR(200) DEFAULT '' COMMENT '联系方式',
            issued_at DATETIME NOT NULL COMMENT '签发时间',
            expires_at DATETIME NOT NULL COMMENT '到期时间',
            duration_days INT NOT NULL COMMENT '授权天数',
            license_data TEXT NOT NULL COMMENT '加密后的授权数据',
            status TINYINT DEFAULT 1 COMMENT '1=有效 0=已吊销',
            revoked_at DATETIME DEFAULT NULL COMMENT '吊销时间',
            remark VARCHAR(500) DEFAULT '' COMMENT '备注',
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            INDEX idx_machine (machine_code),
            INDEX idx_status (status),
            INDEX idx_expires (expires_at)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='喵喵云结算授权记录'");
    }
    return $pdo;
}

// ==================== 加解密 ====================
function aesKey(): string {
    return hex2bin(AES_KEY_HEX);
}

function encryptLicense(array $payload): string {
    $data = json_encode($payload, JSON_UNESCAPED_UNICODE);

    // RSA 签名 (直接签名原始数据，openssl_sign 内部做 SHA256+DigestInfo)
    $privKey = openssl_pkey_get_private(RSA_PRIVATE_KEY);
    openssl_sign($data, $signature, $privKey, OPENSSL_ALGO_SHA256);

    // 组合 payload + signature (RSA-2048 → 256字节签名)
    $combined = $data . $signature;

    // AES-256-GCM 加密
    $iv = openssl_random_pseudo_bytes(12);
    $tag = '';
    $ciphertext = openssl_encrypt($combined, 'aes-256-gcm', aesKey(), OPENSSL_RAW_DATA, $iv, $tag);
    // 格式: iv(12) + ciphertext + tag(16)
    $encrypted = $iv . $ciphertext . $tag;
    return base64_encode($encrypted);
}

// ==================== API 路由 ====================
$action = $_GET['action'] ?? '';

// 简单鉴权（生产环境建议用 JWT 或 session）
function checkAuth(): void {
    $auth = $_SERVER['HTTP_AUTHORIZATION'] ?? $_SERVER['REDIRECT_HTTP_AUTHORIZATION'] ?? '';
    // PHP 内置服务器兼容：从 getallheaders() 获取 Authorization
    if (!$auth && function_exists('getallheaders')) {
        $headers = @getallheaders();
        $auth = $headers['Authorization'] ?? $headers['authorization'] ?? '';
    }
    $inputPwd = '';
    if (preg_match('/Bearer\s+(.+)/', $auth, $m)) {
        $inputPwd = $m[1];
    }
    // 也支持 ?token= 参数
    if (!$inputPwd) $inputPwd = $_GET['token'] ?? '';
    if (hash('sha256', $inputPwd) !== ADMIN_PASSWORD_HASH) {
        http_response_code(401);
        echo json_encode(['error' => '未授权访问', 'code' => 401], JSON_UNESCAPED_UNICODE);
        exit;
    }
}

switch ($action) {
    // ========== 签发授权 ==========
    case 'issue':
        checkAuth();
        $input = json_decode(file_get_contents('php://input'), true);
        $machineCode = trim($input['machine_code'] ?? '');
        $customer = trim($input['customer_name'] ?? '');
        $durationDays = intval($input['duration_days'] ?? 0);
        $contact = trim($input['contact_info'] ?? '');
        $remark = trim($input['remark'] ?? '');

        if (!$machineCode) die(json_encode(['error' => '机器码不能为空']));
        if (!$customer) die(json_encode(['error' => '客户名称不能为空']));
        if ($durationDays < 1 || $durationDays > 3650) die(json_encode(['error' => '授权天数需在1-3650之间']));

        $now = new DateTime();
        $issuedAt = $now->format('Y-m-d\TH:i:sP');
        $expiresAt = (clone $now)->modify("+{$durationDays} days")->format('Y-m-d\TH:i:sP');

        // 检查该机器码是否已有有效授权
        $db = getDB();
        $stmt = $db->prepare("SELECT id FROM yunfei_licenses WHERE machine_code=? AND status=1 AND expires_at > NOW()");
        $stmt->execute([$machineCode]);
        if ($stmt->fetch()) {
            die(json_encode(['error' => '该机器码已有有效授权，请先吊销旧授权']));
        }

        $payload = [
            'version' => 1,
            'machine_code' => $machineCode,
            'customer_name' => $customer,
            'issued_at' => $issuedAt,
            'expires_at' => $expiresAt,
            'duration_days' => $durationDays,
            'features' => ['freight_calc', 'report_export'],
            'nonce' => bin2hex(random_bytes(8)),
        ];

        $licenseData = encryptLicense($payload);

        $stmt = $db->prepare("INSERT INTO yunfei_licenses (machine_code, customer_name, contact_info, issued_at, expires_at, duration_days, license_data, remark) VALUES (?,?,?,?,?,?,?,?)");
        $stmt->execute([$machineCode, $customer, $contact, $issuedAt, $expiresAt, $durationDays, $licenseData, $remark]);
        $id = $db->lastInsertId();

        echo json_encode([
            'ok' => true,
            'id' => (int)$id,
            'machine_code' => $machineCode,
            'customer_name' => $customer,
            'expires_at' => $expiresAt,
            'license_data' => $licenseData,
            'message' => "授权签发成功！到期: {$expiresAt}",
        ], JSON_UNESCAPED_UNICODE);
        break;

    // ========== 查询授权列表 ==========
    case 'list':
        checkAuth();
        $db = getDB();
        $status = $_GET['status'] ?? '';
        $search = $_GET['search'] ?? '';
        $page = max(1, intval($_GET['page'] ?? 1));
        $limit = 20;
        $offset = ($page - 1) * $limit;

        $where = '1=1';
        $params = [];
        if ($status !== '') {
            $where .= ' AND status=?';
            $params[] = intval($status);
        }
        if ($search) {
            $where .= ' AND (machine_code LIKE ? OR customer_name LIKE ?)';
            $params[] = "%{$search}%";
            $params[] = "%{$search}%";
        }

        $stmt = $db->prepare("SELECT COUNT(*) FROM yunfei_licenses WHERE {$where}");
        $stmt->execute($params);
        $total = (int)$stmt->fetchColumn();

        $stmt = $db->prepare("SELECT id, machine_code, customer_name, contact_info, issued_at, expires_at, duration_days, status, revoked_at, remark, created_at FROM yunfei_licenses WHERE {$where} ORDER BY id DESC LIMIT {$limit} OFFSET {$offset}");
        $stmt->execute($params);
        $list = $stmt->fetchAll();

        echo json_encode(['ok' => true, 'total' => $total, 'page' => $page, 'limit' => $limit, 'list' => $list], JSON_UNESCAPED_UNICODE);
        break;

    // ========== 吊销授权 ==========
    case 'revoke':
        checkAuth();
        $input = json_decode(file_get_contents('php://input'), true);
        $id = intval($input['id'] ?? 0);
        if (!$id) die(json_encode(['error' => '缺少授权ID']));

        $db = getDB();
        $stmt = $db->prepare("UPDATE yunfei_licenses SET status=0, revoked_at=NOW() WHERE id=? AND status=1");
        $stmt->execute([$id]);

        if ($stmt->rowCount() > 0) {
            echo json_encode(['ok' => true, 'message' => '已吊销']);
        } else {
            echo json_encode(['error' => '授权不存在或已被吊销']);
        }
        break;

    // ========== 验证授权（客户端调用） ==========
    case 'verify':
        $input = json_decode(file_get_contents('php://input'), true);
        $machineCode = trim($input['machine_code'] ?? '');
        $licenseData = trim($input['license_data'] ?? '');

        if (!$machineCode) die(json_encode(['valid' => false, 'reason' => '缺少机器码']));

        // 从数据库查有效授权
        $db = getDB();
        $stmt = $db->prepare("SELECT license_data FROM yunfei_licenses WHERE machine_code=? AND status=1 AND expires_at > NOW() ORDER BY id DESC LIMIT 1");
        $stmt->execute([$machineCode]);
        $row = $stmt->fetch();

        $valid = $row !== false;
        echo json_encode([
            'valid' => $valid,
            'message' => $valid ? '授权有效' : '未找到有效授权',
        ], JSON_UNESCAPED_UNICODE);
        break;

    // ========== 获取授权数据（用于重新下载） ==========
    case 'get_license':
        checkAuth();
        $id = intval($_GET['id'] ?? 0);
        if (!$id) die(json_encode(['error' => '缺少ID']));

        $db = getDB();
        $stmt = $db->prepare("SELECT license_data, machine_code, customer_name, expires_at FROM yunfei_licenses WHERE id=?");
        $stmt->execute([$id]);
        $row = $stmt->fetch();

        if (!$row) {
            echo json_encode(['error' => '授权不存在']);
        } else {
            echo json_encode([
                'ok' => true,
                'license_data' => $row['license_data'],
                'machine_code' => $row['machine_code'],
                'customer_name' => $row['customer_name'],
                'expires_at' => $row['expires_at'],
                'filename' => "license_{$row['machine_code']}.dat",
            ], JSON_UNESCAPED_UNICODE);
        }
        break;

    default:
        http_response_code(404);
        echo json_encode(['error' => '未知操作', 'available' => ['issue', 'list', 'revoke', 'verify', 'get_license']]);
}
