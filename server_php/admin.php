<?php
/**
 * 喵喵云结算 - 授权管理后台 (RSA+AES加密版)
 * 完整功能：签发授权、列表管理、搜索筛选、分页、统计面板、批量操作、到期提醒
 */

// ==================== 数据库配置 ====================
$DB_HOST = '127.0.0.1';
$DB_PORT = '3306';
$DB_USER = 'root';
$DB_PASS = 'cxdxfx12';
$DB_NAME = 'dasheng';

// ==================== AES-256 密钥 (与客户端 crypto.go 完全一致) ====================
define('AES_KEY_HEX', '7a3b951c4e62af8d2f51dc731a886b0e3947b5e2149d26f85c03ea67901fd4ab');

// ==================== RSA 私钥 (仅服务器持有，用于签名) ====================
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

// ==================== 管理员密码 ====================
$ADMIN_PASS = 'yunfei@admin2024';

// ==================== 会话 ====================
session_start();

// ==================== 数据库连接 ====================
$pdo = new PDO("mysql:host=$DB_HOST;port=$DB_PORT;dbname=$DB_NAME;charset=utf8mb4", $DB_USER, $DB_PASS);

/**
 * 确保表结构包含新字段 (自动迁移)
 */
function ensureSchema($pdo) {
    $cols = $pdo->query("SHOW COLUMNS FROM yunfei_licenses LIKE 'license_data'")->fetchAll();
    if (empty($cols)) {
        $pdo->exec("ALTER TABLE yunfei_licenses ADD COLUMN license_data TEXT COMMENT '加密授权数据(RSA+AES-256-GCM)' AFTER expires_at");
    }
    $cols = $pdo->query("SHOW COLUMNS FROM yunfei_licenses LIKE 'issued_at'")->fetchAll();
    if (empty($cols)) {
        $pdo->exec("ALTER TABLE yunfei_licenses ADD COLUMN issued_at DATETIME DEFAULT NULL COMMENT '签发时间' AFTER license_data");
    }
    $cols = $pdo->query("SHOW COLUMNS FROM yunfei_licenses LIKE 'duration_days'")->fetchAll();
    if (empty($cols)) {
        $pdo->exec("ALTER TABLE yunfei_licenses ADD COLUMN duration_days INT DEFAULT 0 COMMENT '授权天数' AFTER issued_at");
    }
    $cols = $pdo->query("SHOW COLUMNS FROM yunfei_licenses LIKE 'contact_info'")->fetchAll();
    if (empty($cols)) {
        $pdo->exec("ALTER TABLE yunfei_licenses ADD COLUMN contact_info VARCHAR(200) DEFAULT '' COMMENT '联系方式' AFTER customer_name");
    }
    $cols = $pdo->query("SHOW COLUMNS FROM yunfei_licenses LIKE 'remark'")->fetchAll();
    if (empty($cols)) {
        $pdo->exec("ALTER TABLE yunfei_licenses ADD COLUMN remark VARCHAR(500) DEFAULT '' COMMENT '备注' AFTER duration_days");
    }
}
ensureSchema($pdo);

// ==================== RSA+AES-256-GCM 加密 ====================
function aesKey(): string {
    return hex2bin(AES_KEY_HEX);
}

/**
 * 加密授权数据 — 与客户端 crypto.go EncryptLicense() 完全一致
 */
function encryptLicense(array $payload): string {
    $data = json_encode($payload, JSON_UNESCAPED_UNICODE);

    $privKey = openssl_pkey_get_private(RSA_PRIVATE_KEY);
    openssl_sign($data, $signature, $privKey, OPENSSL_ALGO_SHA256);

    $combined = $data . $signature;

    $nonce = openssl_random_pseudo_bytes(12);
    $tag = '';
    $ciphertext = openssl_encrypt($combined, 'aes-256-gcm', aesKey(), OPENSSL_RAW_DATA, $nonce, $tag);
    $encrypted = $nonce . $ciphertext . $tag;
    return base64_encode($encrypted);
}

// ==================== 操作处理 ====================
$action = $_POST['action'] ?? ($_GET['action'] ?? '');

// 登录/登出
if ($action === 'login') {
    if ($_POST['pass'] === $ADMIN_PASS) {
        $_SESSION['yunfei_admin'] = true;
    }
}
$loggedIn = ($_SESSION['yunfei_admin'] ?? false) === true;
if ($action === 'logout') { session_destroy(); $loggedIn = false; }

$msg = '';
$msgType = 'success';
// 接收重定向传来的消息（防 F5 重复 POST）
if (isset($_GET['msg'])) {
    $msg = $_GET['msg'];
    $msgType = $_GET['type'] ?? 'success';
}

// 签发授权
if ($loggedIn && $action === 'create') {
    $machineCode = trim($_POST['machine_code'] ?? '');
    $customer = trim($_POST['customer_name'] ?? '');
    $days = intval($_POST['duration_days'] ?? 365);
    $contact = trim($_POST['contact_info'] ?? '');
    $remark = trim($_POST['remark'] ?? '');

    if (!$machineCode) {
        $msg = '机器码不能为空'; $msgType = 'error';
    } elseif (!$customer) {
        $msg = '客户名称不能为空'; $msgType = 'error';
    } elseif ($days < 1 || $days > 3650) {
        $msg = '授权天数需在1-3650之间'; $msgType = 'error';
    } else {
        $stmt = $pdo->prepare("SELECT id FROM yunfei_licenses WHERE machine_code=? AND status=2 AND expires_at >= CURDATE()");
        $stmt->execute([$machineCode]);
        if ($stmt->fetch()) {
            $msg = '该机器码已有有效授权，请先停用旧授权'; $msgType = 'error';
        } else {
            $now = new DateTime();
            $issuedAt = $now->format('Y-m-d\TH:i:sP');
            $expiresAt = clone $now;
            $expiresAt->modify("+{$days} days");
            $expiresAtStr = $expiresAt->format('Y-m-d\TH:i:sP');
            $expiresDateStr = $expiresAt->format('Y-m-d');

            $payload = [
                'version' => 1,
                'machine_code' => $machineCode,
                'customer_name' => $customer,
                'issued_at' => $issuedAt,
                'expires_at' => $expiresAtStr,
                'duration_days' => $days,
                'features' => ['freight_calc', 'report_export'],
                'nonce' => bin2hex(random_bytes(8)),
            ];

            $licenseData = encryptLicense($payload);

            $stmt = $pdo->prepare(
                "INSERT INTO yunfei_licenses (license_key, machine_code, customer_name, contact_info, expires_at, license_data, issued_at, duration_days, remark, status, activated_at)
                 VALUES ('', ?, ?, ?, ?, ?, NOW(), ?, ?, 2, NOW())"
            );
            $stmt->execute([$machineCode, $customer, $contact, $expiresDateStr, $licenseData, $days, $remark]);

            $msg = "授权签发成功！客户: {$customer}，到期: {$expiresDateStr} (共{$days}天)";
        }
    }
}

// 续期（带防重复令牌）
if ($loggedIn && $action === 'renew') {
    // 防重复提交：检查 nonce
    $nonce = $_POST['_nonce'] ?? '';
    if (!$nonce || isset($_SESSION['renew_nonce']) && $_SESSION['renew_nonce'] === $nonce) {
        $msg = '请不要重复提交'; $msgType = 'error';
    } else {
        $_SESSION['renew_nonce'] = $nonce;
        $id = intval($_POST['id']);
        $days = intval($_POST['days'] ?? 0);
        if ($days < 1 || $days > 3650) {
            $msg = '续期天数需在1-3650之间'; $msgType = 'error';
        } else {
            $stmt = $pdo->prepare("SELECT * FROM yunfei_licenses WHERE id=?");
            $stmt->execute([$id]);
            $row = $stmt->fetch(PDO::FETCH_ASSOC);
            if ($row) {
                $newExpire = new DateTime($row['expires_at']);
                if ($newExpire < new DateTime()) $newExpire = new DateTime();
                $newExpire->modify("+{$days} days");
                $newExpireStr = $newExpire->format('Y-m-d');

                // 重新生成加密授权数据
                $payload = [
                    'version' => 1,
                    'machine_code' => $row['machine_code'],
                    'customer_name' => $row['customer_name'],
                    'issued_at' => (new DateTime())->format('Y-m-d\TH:i:sP'),
                    'expires_at' => $newExpire->format('Y-m-d\TH:i:sP'),
                    'duration_days' => $days + intval($row['duration_days']),
                    'features' => ['freight_calc', 'report_export'],
                    'nonce' => bin2hex(random_bytes(8)),
                ];
                $licenseData = encryptLicense($payload);

                $pdo->prepare("UPDATE yunfei_licenses SET expires_at=?, license_data=?, issued_at=NOW(), duration_days=duration_days+?, status=2 WHERE id=?")
                    ->execute([$newExpireStr, $licenseData, $days, $id]);
                $msg = "续期成功！新到期: {$newExpireStr} (增加{$days}天)";
            } else {
                $msg = '授权记录不存在'; $msgType = 'error';
            }
        }
        // 防 F5 重复提交：重定向
        if ($msgType !== 'error') {
            header('Location: admin.php?' . http_build_query(['msg' => $msg, 'type' => 'success']));
            exit;
        }
    }
}

// 停用/启用
if ($loggedIn && $action === 'toggle') {
    $id = intval($_POST['id']);
    $status = intval($_POST['status']);
    $pdo->prepare("UPDATE yunfei_licenses SET status = ? WHERE id = ?")->execute([$status, $id]);
}

// 批量停用
if ($loggedIn && $action === 'batch_toggle') {
    $ids = json_decode($_POST['ids'] ?? '[]', true);
    $status = intval($_POST['status']);
    if (!empty($ids)) {
        $placeholders = implode(',', array_fill(0, count($ids), '?'));
        $stmt = $pdo->prepare("UPDATE yunfei_licenses SET status = ? WHERE id IN ($placeholders)");
        $stmt->execute(array_merge([$status], $ids));
        $msg = "已批量" . ($status == 3 ? '停用' : '启用') . " " . count($ids) . " 条记录";
    }
}

// 批量删除
if ($loggedIn && $action === 'batch_delete') {
    $ids = json_decode($_POST['ids'] ?? '[]', true);
    if (!empty($ids)) {
        $placeholders = implode(',', array_fill(0, count($ids), '?'));
        $pdo->prepare("DELETE FROM yunfei_licenses WHERE id IN ($placeholders)")->execute($ids);
        $msg = "已删除 " . count($ids) . " 条记录";
    }
}

// 删除
if ($loggedIn && $action === 'delete') {
    $id = intval($_POST['id']);
    $pdo->prepare("DELETE FROM yunfei_licenses WHERE id = ?")->execute([$id]);
}

// 查看单个授权数据（弹窗用JSON）
if ($loggedIn && $action === 'view') {
    header('Content-Type: application/json; charset=utf-8');
    $id = intval($_POST['id']);
    $stmt = $pdo->prepare("SELECT * FROM yunfei_licenses WHERE id=?");
    $stmt->execute([$id]);
    $row = $stmt->fetch(PDO::FETCH_ASSOC);
    if ($row) {
        echo json_encode([
            'ok' => true,
            'license_data' => $row['license_data'],
            'customer_name' => $row['customer_name'],
            'machine_code' => $row['machine_code'],
            'expires_at' => $row['expires_at'],
            'contact_info' => $row['contact_info'],
            'remark' => $row['remark'],
        ], JSON_UNESCAPED_UNICODE);
    } else {
        echo json_encode(['ok' => false, 'msg' => '未找到']);
    }
    exit;
}

// 下载单个授权为 .dat 文件
if ($loggedIn && $action === 'download') {
    $id = intval($_GET['id']);
    $stmt = $pdo->prepare("SELECT license_data, machine_code, customer_name FROM yunfei_licenses WHERE id=?");
    $stmt->execute([$id]);
    $row = $stmt->fetch(PDO::FETCH_ASSOC);
    if ($row && $row['license_data']) {
        $filename = 'license_' . preg_replace('/[^a-zA-Z0-9\-]/', '_', $row['machine_code']) . '.dat';
        header('Content-Type: application/octet-stream');
        header('Content-Disposition: attachment; filename="' . $filename . '"');
        header('Content-Length: ' . strlen($row['license_data']));
        echo $row['license_data'];
        exit;
    }
    die('授权数据不存在');
}

// ==================== 统计面板数据 ====================
$stats = [
    'total' => 0,
    'active' => 0,
    'inactive' => 0,
    'expiring_soon' => 0, // 30天内到期
    'expired' => 0,
    'today_issued' => 0,
];
if ($loggedIn) {
    $stats['total'] = $pdo->query("SELECT COUNT(*) FROM yunfei_licenses")->fetchColumn();
    $stats['active'] = $pdo->query("SELECT COUNT(*) FROM yunfei_licenses WHERE status=2")->fetchColumn();
    $stats['inactive'] = $pdo->query("SELECT COUNT(*) FROM yunfei_licenses WHERE status=3")->fetchColumn();
    $stats['expiring_soon'] = $pdo->query("SELECT COUNT(*) FROM yunfei_licenses WHERE status=2 AND expires_at BETWEEN CURDATE() AND DATE_ADD(CURDATE(), INTERVAL 30 DAY)")->fetchColumn();
    $stats['expired'] = $pdo->query("SELECT COUNT(*) FROM yunfei_licenses WHERE status=2 AND expires_at < CURDATE()")->fetchColumn();
    $stats['today_issued'] = $pdo->query("SELECT COUNT(*) FROM yunfei_licenses WHERE DATE(created_at) = CURDATE()")->fetchColumn();
}

// ==================== 查询列表（搜索+筛选+分页） ====================
$licenses = [];
$totalPages = 0;
$currentPage = 1;
$searchKeyword = '';
$filterStatus = '';

if ($loggedIn) {
    $searchKeyword = trim($_GET['search'] ?? '');
    $filterStatus = $_GET['status'] ?? '';
    $currentPage = max(1, intval($_GET['page'] ?? 1));
    $perPage = 20;

    $where = [];
    $params = [];

    if ($searchKeyword) {
        $where[] = "(customer_name LIKE ? OR machine_code LIKE ? OR contact_info LIKE ? OR remark LIKE ?)";
        $kw = "%{$searchKeyword}%";
        $params = array_merge($params, [$kw, $kw, $kw, $kw]);
    }

    if ($filterStatus !== '') {
        $where[] = "status = ?";
        $params[] = intval($filterStatus);
    }

    $whereClause = $where ? 'WHERE ' . implode(' AND ', $where) : '';

    $totalCount = $pdo->prepare("SELECT COUNT(*) FROM yunfei_licenses $whereClause");
    $totalCount->execute($params);
    $totalRows = $totalCount->fetchColumn();
    $totalPages = ceil($totalRows / $perPage);

    $offset = ($currentPage - 1) * $perPage;
    $stmt = $pdo->prepare(
        "SELECT id, license_key, machine_code, customer_name, contact_info, expires_at, status, activated_at, created_at, duration_days, remark, last_check_at,
                CASE WHEN license_data IS NOT NULL AND license_data != '' THEN 1 ELSE 0 END AS has_encrypted
         FROM yunfei_licenses $whereClause ORDER BY created_at DESC LIMIT $perPage OFFSET $offset"
    );
    $stmt->execute($params);
    $licenses = $stmt->fetchAll(PDO::FETCH_ASSOC);
}

// ==================== 状态映射 ====================
function statusTag($s) {
    if ($s == 1) return ['cls' => 'tag-blue', 'txt' => '未激活'];
    if ($s == 2) return ['cls' => 'tag-green', 'txt' => '已激活'];
    return ['cls' => 'tag-red', 'txt' => '已停用'];
}

function daysUntil($dateStr) {
    if (!$dateStr) return null;
    $expire = new DateTime($dateStr);
    $now = new DateTime();
    if ($expire < $now) return -$now->diff($expire)->days; // 负数表示已过期
    return $now->diff($expire)->days;
}

function expireClass($daysLeft) {
    if ($daysLeft === null) return '';
    if ($daysLeft < 0) return 'expired';
    if ($daysLeft <= 7) return 'danger';
    if ($daysLeft <= 30) return 'warning';
    return '';
}
?>
<!DOCTYPE html>
<html lang="zh">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>喵喵云结算 - 授权管理</title>
<link rel="icon" href="data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 100 100'><text y='.9em' font-size='90'>🔐</text></svg>">
<style>
/* ========== 基础样式 ========== */
*{margin:0;padding:0;box-sizing:border-box}
:root {
  --primary: #6366f1;
  --primary-hover: #4f46e5;
  --primary-light: #eef2ff;
  --success: #10b981;
  --success-light: #d1fae5;
  --danger: #ef4444;
  --danger-light: #fee2e2;
  --warning: #f59e0b;
  --warning-light: #fef3c7;
  --info: #3b82f6;
  --info-light: #dbeafe;
  --bg: #f1f5f9;
  --card: #ffffff;
  --border: #e2e8f0;
  --text: #1e293b;
  --text-muted: #94a3b8;
  --text-secondary: #64748b;
  --radius: 12px;
  --radius-sm: 8px;
  --shadow: 0 1px 3px rgba(0,0,0,.06),0 1px 2px rgba(0,0,0,.04);
  --shadow-md: 0 4px 6px rgba(0,0,0,.05),0 2px 4px rgba(0,0,0,.04);
  --shadow-lg: 0 10px 25px rgba(0,0,0,.08);
  --transition: all .2s ease;
}
body {
  font: 14px/1.6 -apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,"Helvetica Neue",Arial,"PingFang SC","Microsoft YaHei",sans-serif;
  background: var(--bg); color: var(--text); min-height: 100vh;
}
.container { max-width: 1340px; margin: 0 auto; padding: 20px 24px; }

/* ========== 头部导航 ========== */
.header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 16px 24px; background: var(--card); border-radius: var(--radius);
  box-shadow: var(--shadow); margin-bottom: 20px;
}
.header-left { display: flex; align-items: center; gap: 12px; }
.header-left .logo {
  width: 40px; height: 40px; background: linear-gradient(135deg, var(--primary), #8b5cf6);
  border-radius: 10px; display: flex; align-items: center; justify-content: center;
  color: #fff; font-size: 20px;
}
.header-left h1 { font-size: 20px; font-weight: 700; color: var(--text); }
.header-left .badge {
  font-size: 11px; background: var(--primary-light); color: var(--primary);
  padding: 2px 10px; border-radius: 20px; font-weight: 600;
}
.header-right { display: flex; align-items: center; gap: 12px; }
.header-right .encryption-info {
  display: flex; align-items: center; gap: 6px; font-size: 12px;
  color: var(--text-muted); background: var(--bg); padding: 6px 14px; border-radius: 20px;
}
.header-right .encryption-info .dot { width: 8px; height: 8px; background: var(--success); border-radius: 50%; }

/* ========== 统计面板 ========== */
.stats-grid {
  display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px; margin-bottom: 20px;
}
.stat-card {
  background: var(--card); border-radius: var(--radius); padding: 20px;
  box-shadow: var(--shadow); display: flex; align-items: center; gap: 16px;
  transition: var(--transition); cursor: default; border: 1px solid transparent;
}
.stat-card:hover { box-shadow: var(--shadow-md); transform: translateY(-1px); }
.stat-card.clickable { cursor: pointer; }
.stat-card.clickable:hover { border-color: var(--primary); }
.stat-icon {
  width: 48px; height: 48px; border-radius: 12px; display: flex;
  align-items: center; justify-content: center; font-size: 22px; flex-shrink: 0;
}
.stat-icon.total { background: var(--info-light); color: var(--info); }
.stat-icon.active { background: var(--success-light); color: var(--success); }
.stat-icon.inactive { background: var(--danger-light); color: var(--danger); }
.stat-icon.expiring { background: var(--warning-light); color: var(--warning); }
.stat-icon.expired { background: #fee2e2; color: #dc2626; }
.stat-icon.today { background: #ede9fe; color: #7c3aed; }
.stat-info { display: flex; flex-direction: column; }
.stat-value { font-size: 28px; font-weight: 700; line-height: 1; }
.stat-label { font-size: 13px; color: var(--text-muted); margin-top: 4px; }

/* ========== 卡片 ========== */
.card {
  background: var(--card); border-radius: var(--radius); padding: 24px;
  margin-bottom: 20px; box-shadow: var(--shadow);
}
.card-header {
  display: flex; align-items: center; justify-content: space-between;
  margin-bottom: 20px; flex-wrap: wrap; gap: 12px;
}
.card-header h3 { font-size: 17px; font-weight: 600; display: flex; align-items: center; gap: 8px; }
.card-header h3 .icon { font-size: 20px; }

/* ========== 表单 ========== */
input, select, textarea, button {
  font-family: inherit; font-size: 14px; transition: var(--transition);
}
input, select, textarea {
  padding: 10px 14px; border: 1.5px solid var(--border);
  border-radius: var(--radius-sm); background: var(--card); color: var(--text);
  outline: none;
}
input:focus, select:focus, textarea:focus {
  border-color: var(--primary); box-shadow: 0 0 0 3px rgba(99,102,241,.1);
}
textarea { resize: vertical; min-height: 60px; }

/* ========== 按钮 ========== */
.btn {
  display: inline-flex; align-items: center; justify-content: center; gap: 6px;
  padding: 10px 20px; border: none; border-radius: var(--radius-sm);
  cursor: pointer; font-weight: 500; font-size: 14px; text-decoration: none;
  transition: var(--transition); white-space: nowrap;
}
.btn-primary { background: var(--primary); color: #fff; }
.btn-primary:hover { background: var(--primary-hover); box-shadow: 0 2px 8px rgba(99,102,241,.3); }
.btn-success { background: var(--success); color: #fff; }
.btn-success:hover { background: #059669; box-shadow: 0 2px 8px rgba(16,185,129,.3); }
.btn-danger { background: var(--danger); color: #fff; }
.btn-danger:hover { background: #dc2626; box-shadow: 0 2px 8px rgba(239,68,68,.3); }
.btn-warning { background: var(--warning); color: #fff; }
.btn-warning:hover { background: #d97706; }
.btn-outline {
  background: transparent; color: var(--primary); border: 1.5px solid var(--primary);
}
.btn-outline:hover { background: var(--primary-light); }
.btn-outline-danger {
  background: transparent; color: var(--danger); border: 1.5px solid var(--danger);
}
.btn-outline-danger:hover { background: var(--danger-light); }
.btn-sm { padding: 6px 14px; font-size: 13px; border-radius: 6px; }
.btn-xs { padding: 4px 10px; font-size: 12px; border-radius: 5px; }

/* ========== 表格 ========== */
.table-wrap { overflow-x: auto; border-radius: var(--radius-sm); border: 1px solid var(--border); }
table { width: 100%; border-collapse: collapse; font-size: 13px; }
th, td { padding: 12px 14px; text-align: left; }
th {
  background: #f8fafc; font-weight: 600; color: var(--text-secondary);
  font-size: 12px; text-transform: uppercase; letter-spacing: .05em;
  border-bottom: 2px solid var(--border); white-space: nowrap; user-select: none;
}
td { border-bottom: 1px solid #f1f5f9; }
tr:last-child td { border-bottom: none; }
tr:hover td { background: #f8fafc; }
tr.selected td { background: var(--primary-light); }

/* 状态标签 */
.tag {
  display: inline-flex; align-items: center; gap: 4px;
  padding: 3px 12px; border-radius: 20px; font-size: 12px; font-weight: 500; white-space: nowrap;
}
.tag-green { background: var(--success-light); color: #059669; }
.tag-blue { background: var(--info-light); color: #2563eb; }
.tag-red { background: var(--danger-light); color: #dc2626; }
.tag-gray { background: #f1f5f9; color: #64748b; }
.tag-warning { background: var(--warning-light); color: #d97706; }
.tag-purple { background: #ede9fe; color: #7c3aed; }

/* 到期天数 */
.expire-days { font-weight: 600; }
.expire-days.expired { color: var(--danger); }
.expire-days.danger { color: var(--danger); }
.expire-days.warning { color: var(--warning); }

/* ========== 消息提示 ========== */
.msg {
  padding: 12px 18px; border-radius: var(--radius-sm); margin-bottom: 16px;
  font-weight: 500; display: flex; align-items: center; gap: 8px;
  animation: slideDown .3s ease;
}
@keyframes slideDown { from { opacity: 0; transform: translateY(-8px); } to { opacity: 1; transform: translateY(0); } }
.msg-success { background: var(--success-light); color: #059669; border: 1px solid #a7f3d0; }
.msg-error { background: var(--danger-light); color: #dc2626; border: 1px solid #fecaca; }
.msg-warning { background: var(--warning-light); color: #d97706; border: 1px solid #fde68a; }

/* ========== 工具条 ========== */
.toolbar {
  display: flex; align-items: center; gap: 10px; flex-wrap: wrap; margin-bottom: 16px;
}
.toolbar .search-box { position: relative; flex: 1; min-width: 200px; max-width: 360px; }
.toolbar .search-box input { width: 100%; padding-left: 38px; }
.toolbar .search-box .search-icon {
  position: absolute; left: 12px; top: 50%; transform: translateY(-50%);
  color: var(--text-muted); font-size: 16px; pointer-events: none;
}
.toolbar select { padding: 10px 14px; }

/* ========== 分页 ========== */
.pagination {
  display: flex; align-items: center; justify-content: space-between;
  margin-top: 16px; flex-wrap: wrap; gap: 12px;
}
.pagination-info { font-size: 13px; color: var(--text-muted); }
.pagination-btns { display: flex; gap: 4px; }
.pagination-btns a, .pagination-btns span {
  display: inline-flex; align-items: center; justify-content: center;
  min-width: 36px; height: 36px; padding: 0 8px;
  border-radius: var(--radius-sm); font-size: 13px; font-weight: 500;
  text-decoration: none; transition: var(--transition);
  border: 1px solid var(--border); color: var(--text-secondary); background: var(--card);
}
.pagination-btns a:hover { background: var(--primary-light); color: var(--primary); border-color: var(--primary); }
.pagination-btns .active { background: var(--primary); color: #fff; border-color: var(--primary); }
.pagination-btns .disabled { color: #cbd5e1; pointer-events: none; }

/* ========== 模态框 ========== */
.modal-mask {
  display: none; position: fixed; inset: 0; background: rgba(15,23,42,.5);
  backdrop-filter: blur(4px); z-index: 1000; justify-content: center; align-items: center;
  animation: fadeIn .2s ease;
}
.modal-mask.active { display: flex; }
@keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }
.modal-box {
  background: var(--card); border-radius: 16px; padding: 28px;
  width: 90%; max-width: 680px; max-height: 85vh; overflow-y: auto;
  box-shadow: var(--shadow-lg); animation: scaleIn .25s ease;
}
@keyframes scaleIn { from { opacity: 0; transform: scale(.95); } to { opacity: 1; transform: scale(1); } }
.modal-box h3 { font-size: 18px; margin-bottom: 8px; display: flex; align-items: center; gap: 8px; }
.modal-box .modal-subtitle { color: var(--text-muted); font-size: 13px; margin-bottom: 20px; }
.modal-box .info-grid {
  display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 12px; margin-bottom: 16px;
}
.modal-box .info-item { background: #f8fafc; border-radius: var(--radius-sm); padding: 12px; }
.modal-box .info-item .label { font-size: 11px; color: var(--text-muted); text-transform: uppercase; letter-spacing: .05em; margin-bottom: 4px; }
.modal-box .info-item .value { font-size: 14px; font-weight: 500; word-break: break-all; }
.license-data-preview {
  background: #1e293b; color: #e2e8f0; font-family: 'SF Mono','Fira Code','Consolas',monospace;
  font-size: 12px; padding: 16px; border-radius: var(--radius-sm);
  max-height: 200px; overflow-y: auto; word-break: break-all; line-height: 1.7;
  margin-bottom: 16px; position: relative;
}
.license-data-preview .copy-hint {
  position: absolute; top: 8px; right: 8px;
  font-size: 11px; color: #64748b; font-family: inherit; opacity: 0;
  transition: opacity .2s;
}
.license-data-preview:hover .copy-hint { opacity: 1; }
.modal-actions { display: flex; gap: 8px; justify-content: flex-end; margin-top: 20px; }

/* ========== 签发表单 ========== */
.issue-form { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
.issue-form .full-width { grid-column: 1 / -1; }
.form-group { display: flex; flex-direction: column; gap: 6px; }
.form-group label { font-size: 13px; font-weight: 600; color: var(--text-secondary); }
.form-group .hint { font-size: 11px; color: var(--text-muted); }

/* ========== 响应式 ========== */
@media (max-width: 768px) {
  .container { padding: 12px; }
  .card { padding: 16px; }
  .header { flex-direction: column; gap: 12px; align-items: flex-start; }
  .stats-grid { grid-template-columns: repeat(2, 1fr); }
  .issue-form { grid-template-columns: 1fr; }
  th, td { padding: 8px 10px; font-size: 12px; }
}

/* ========== 复选框 ========== */
.checkbox-wrap { display: flex; align-items: center; gap: 8px; }
.checkbox-wrap input[type=checkbox] { width: 18px; height: 18px; accent-color: var(--primary); cursor: pointer; }

/* ========== 空状态 ========== */
.empty-state { text-align: center; padding: 60px 20px; color: var(--text-muted); }
.empty-state .empty-icon { font-size: 48px; margin-bottom: 12px; }
.empty-state p { font-size: 15px; }

/* ========== Toast ========== */
.toast {
  position: fixed; top: 20px; right: 20px; z-index: 9999;
  background: var(--card); padding: 14px 20px; border-radius: var(--radius-sm);
  box-shadow: var(--shadow-lg); display: flex; align-items: center; gap: 10px;
  font-weight: 500; font-size: 14px; animation: slideIn .3s ease;
  border-left: 4px solid var(--primary);
}
@keyframes slideIn { from { opacity: 0; transform: translateX(100px); } to { opacity: 1; transform: translateX(0); } }
.toast.success { border-left-color: var(--success); }
.toast.error { border-left-color: var(--danger); }

/* ========== 空状态插图 ========== */
.tip-box {
  background: linear-gradient(135deg, #f0f9ff, #eef2ff);
  border: 1px dashed #c7d2fe; border-radius: var(--radius-sm);
  padding: 14px 18px; font-size: 13px; color: #4338ca;
  display: flex; align-items: center; gap: 8px; margin-bottom: 16px;
}

/* ========== 续期弹窗 ========== */
.renew-form { display: flex; gap: 12px; align-items: flex-end; margin-top: 12px; }
.renew-form .form-group { flex: 1; }

/* ========== 页面加载 ========== */
.loading { opacity: .6; pointer-events: none; }

/* ========== 链接按钮 ========== */
a.btn { text-decoration: none; display: inline-flex; }
</style>
</head>
<body>
<div class="container">

<?php if (!$loggedIn): ?>
<!-- ==================== 登录页面 ==================== -->
<div style="min-height: 100vh; display: flex; align-items: center; justify-content: center;">
  <div style="background: var(--card); border-radius: 16px; padding: 40px; width: 100%; max-width: 400px; box-shadow: var(--shadow-lg); text-align: center;">
    <div style="width: 64px; height: 64px; background: linear-gradient(135deg, var(--primary), #8b5cf6); border-radius: 16px; display: flex; align-items: center; justify-content: center; color: #fff; font-size: 32px; margin: 0 auto 20px;">🔐</div>
    <h2 style="font-size: 24px; margin-bottom: 6px; color: var(--text);">授权管理</h2>
    <p style="color: var(--text-muted); font-size: 14px; margin-bottom: 28px;">喵喵云结算系统 · RSA+AES加密</p>
    <form method="POST">
      <input type="hidden" name="action" value="login">
      <div style="text-align: left; margin-bottom: 20px;">
        <label style="font-size: 13px; font-weight: 600; color: var(--text-secondary); display: block; margin-bottom: 6px;">管理密码</label>
        <input type="password" name="pass" placeholder="请输入管理密码" required style="width: 100%; font-size: 15px; padding: 12px 16px;">
      </div>
      <button class="btn btn-primary" style="width: 100%; padding: 12px; font-size: 16px;">🔓 登录管理后台</button>
    </form>
    <p style="margin-top: 20px; font-size: 12px; color: var(--text-muted);">
      <span style="display: inline-block; width: 8px; height: 8px; background: var(--success); border-radius: 50%; margin-right: 6px;"></span>
      RSA-2048 签名 + AES-256-GCM 加密
    </p>
  </div>
</div>

<?php else: ?>
<!-- ==================== 已登录 ==================== -->

<!-- 头部导航 -->
<div class="header">
  <div class="header-left">
    <div class="logo">🔐</div>
    <h1>授权管理</h1>
    <span class="badge">RSA+AES</span>
  </div>
  <div class="header-right">
    <div class="encryption-info">
      <span class="dot"></span>
      RSA-2048 + AES-256-GCM 加密运行中
    </div>
    <span style="font-size:13px; color: var(--text-muted);"><?= date('Y-m-d H:i') ?></span>
    <form method="POST" style="display:inline">
      <input type="hidden" name="action" value="logout">
      <button class="btn btn-outline-danger btn-sm">退出登录</button>
    </form>
  </div>
</div>

<!-- 消息提示 -->
<?php if($msg): ?>
<div class="msg msg-<?= $msgType ?>">
  <?= $msgType === 'error' ? '⚠️' : '✅' ?> <?= htmlspecialchars($msg) ?>
</div>
<?php endif ?>

<!-- 统计面板 -->
<div class="stats-grid">
  <div class="stat-card clickable" onclick="filterStatus('')">
    <div class="stat-icon total">📊</div>
    <div class="stat-info">
      <div class="stat-value"><?= $stats['total'] ?></div>
      <div class="stat-label">授权总数</div>
    </div>
  </div>
  <div class="stat-card clickable" onclick="filterStatus('2')">
    <div class="stat-icon active">✅</div>
    <div class="stat-info">
      <div class="stat-value"><?= $stats['active'] ?></div>
      <div class="stat-label">已激活</div>
    </div>
  </div>
  <div class="stat-card clickable" onclick="filterStatus('3')">
    <div class="stat-icon inactive">🚫</div>
    <div class="stat-info">
      <div class="stat-value"><?= $stats['inactive'] ?></div>
      <div class="stat-label">已停用</div>
    </div>
  </div>
  <div class="stat-card clickable" onclick="location.href='?status=2&search=expiring'" style="cursor:pointer;">
    <div class="stat-icon expiring">⏰</div>
    <div class="stat-info">
      <div class="stat-value" style="color: <?= $stats['expiring_soon'] > 0 ? 'var(--warning)' : 'inherit' ?>"><?= $stats['expiring_soon'] ?></div>
      <div class="stat-label">30天内到期</div>
    </div>
  </div>
  <div class="stat-card" style="<?= $stats['expired'] > 0 ? 'border-color: var(--danger);' : '' ?>">
    <div class="stat-icon expired">⚠️</div>
    <div class="stat-info">
      <div class="stat-value" style="color: <?= $stats['expired'] > 0 ? 'var(--danger)' : 'inherit' ?>"><?= $stats['expired'] ?></div>
      <div class="stat-label">已过期(仍激活)</div>
    </div>
  </div>
  <div class="stat-card">
    <div class="stat-icon today">📅</div>
    <div class="stat-info">
      <div class="stat-value"><?= $stats['today_issued'] ?></div>
      <div class="stat-label">今日签发</div>
    </div>
  </div>
</div>

<!-- 签发授权 -->
<div class="card" id="issueSection">
  <div class="card-header">
    <h3><span class="icon">📝</span>签发新授权</h3>
    <button class="btn btn-sm btn-outline" onclick="toggleIssueSection()" id="toggleIssueBtn">收起 ▲</button>
  </div>
  <div id="issueBody">
    <div class="tip-box">
      💡 输入客户电脑的<b>机器码</b>（客户端「获取机器码」获得），设置有效期，系统自动生成 RSA+AES 加密授权数据。
    </div>
    <form method="POST" class="issue-form">
      <input type="hidden" name="action" value="create">
      <div class="form-group">
        <label>机器码 *</label>
        <input name="machine_code" required placeholder="xxxx-xxxx-xxxx-xxxx-xxxx-xxxx-xxxx-xxxx">
        <span class="hint">客户端 → 设置 → 获取机器码</span>
      </div>
      <div class="form-group">
        <label>客户名称 *</label>
        <input name="customer_name" required placeholder="张三 / XX公司">
      </div>
      <div class="form-group">
        <label>有效期(天)</label>
        <input name="duration_days" type="number" value="365" min="1" max="3650">
      </div>
      <div class="form-group">
        <label>联系方式</label>
        <input name="contact_info" placeholder="手机/微信(选填)">
      </div>
      <div class="form-group">
        <label>备注</label>
        <input name="remark" placeholder="内部备注(选填)">
      </div>
      <div class="form-group" style="justify-content: flex-end;">
        <label>&nbsp;</label>
        <button class="btn btn-success" style="padding: 10px 28px;">🔒 生成加密授权</button>
      </div>
    </form>
  </div>
</div>

<!-- 授权列表 -->
<div class="card">
  <div class="card-header">
    <h3><span class="icon">📋</span>授权列表</h3>
    <div style="display: flex; gap: 8px;">
      <button class="btn btn-sm btn-outline" onclick="batchToggle(2)" id="batchEnableBtn" style="display:none;">✅ 批量启用</button>
      <button class="btn btn-sm btn-outline-danger" onclick="batchToggle(3)" id="batchDisableBtn" style="display:none;">🚫 批量停用</button>
      <button class="btn btn-sm btn-danger" onclick="batchDelete()" id="batchDeleteBtn" style="display:none;">🗑 批量删除</button>
    </div>
  </div>

  <!-- 搜索与筛选 -->
  <div class="toolbar">
    <div class="search-box">
      <span class="search-icon">🔍</span>
      <input type="text" id="searchInput" placeholder="搜索客户名 / 机器码 / 联系方式..." value="<?= htmlspecialchars($searchKeyword) ?>" onkeydown="if(event.key==='Enter')doSearch()">
    </div>
    <select id="filterSelect" onchange="doSearch()">
      <option value="">全部状态</option>
      <option value="2" <?= $filterStatus === '2' ? 'selected' : '' ?>>✅ 已激活</option>
      <option value="3" <?= $filterStatus === '3' ? 'selected' : '' ?>>🚫 已停用</option>
      <option value="1" <?= $filterStatus === '1' ? 'selected' : '' ?>>📋 未激活</option>
    </select>
    <button class="btn btn-sm btn-primary" onclick="doSearch()">搜索</button>
    <?php if ($searchKeyword || $filterStatus !== ''): ?>
    <a href="?" class="btn btn-sm btn-outline">清除筛选</a>
    <?php endif ?>
  </div>

  <div class="table-wrap">
    <table>
      <thead>
      <tr>
        <th style="width:40px">
          <input type="checkbox" id="selectAll" onchange="toggleSelectAll()" title="全选">
        </th>
        <th>ID</th>
        <th>客户</th>
        <th>机器码</th>
        <th>到期日</th>
        <th>剩余</th>
        <th>天数</th>
        <th>加密</th>
        <th>状态</th>
        <th>最后验证</th>
        <th>操作</th>
      </tr>
      </thead>
      <tbody>
      <?php if (empty($licenses)): ?>
      <tr>
        <td colspan="11">
          <div class="empty-state">
            <div class="empty-icon">📭</div>
            <p><?= $searchKeyword || $filterStatus ? '没有找到匹配的授权记录' : '暂无授权记录，请先签发新授权' ?></p>
          </div>
        </td>
      </tr>
      <?php else: ?>
      <?php foreach($licenses as $l):
        $st = statusTag($l['status']);
        $dLeft = ($l['status'] == 2) ? daysUntil($l['expires_at']) : null;
        $dClass = expireClass($dLeft);
        $dText = ($dLeft === null) ? '-' : (($dLeft < 0) ? '已过期' . abs($dLeft) . '天' : $dLeft . '天');
        $lastCheck = $l['last_check_at'] ? date('m-d H:i', strtotime($l['last_check_at'])) : '-';
      ?>
      <tr id="row-<?= $l['id'] ?>">
        <td><input type="checkbox" class="row-checkbox" value="<?= $l['id'] ?>" onchange="updateBatchBtns()"></td>
        <td style="color:var(--text-muted);font-size:12px;">#<?= $l['id'] ?></td>
        <td>
          <b><?= htmlspecialchars($l['customer_name'] ?: '-') ?></b>
          <?php if ($l['contact_info']): ?>
            <br><span style="font-size:11px;color:var(--text-muted)"><?= htmlspecialchars($l['contact_info']) ?></span>
          <?php endif ?>
        </td>
        <td><code style="font-size:11px;color:var(--text-muted);background:#f1f5f9;padding:2px 6px;border-radius:4px;"><?= htmlspecialchars($l['machine_code'] ?: '-') ?></code></td>
        <td style="white-space:nowrap;"><?= $l['expires_at'] ?></td>
        <td><span class="expire-days <?= $dClass ?>"><?= $dText ?></span></td>
        <td><?= $l['duration_days'] ?: '-' ?>天</td>
        <td>
          <?php if($l['has_encrypted']): ?>
            <span class="tag tag-purple">🔐 加密</span>
          <?php else: ?>
            <span class="tag tag-gray">旧版</span>
          <?php endif ?>
        </td>
        <td><span class="tag <?= $st['cls'] ?>"><?= $st['txt'] ?></span></td>
        <td style="font-size:12px;color:var(--text-muted)"><?= $lastCheck ?></td>
        <td style="white-space:nowrap;">
          <div style="display:flex;gap:4px;align-items:center;">
          <?php if($l['has_encrypted']): ?>
            <button class="btn btn-xs btn-outline" onclick="viewLicense(<?= $l['id'] ?>)" title="查看授权数据">👁</button>
            <a href="?action=download&id=<?= $l['id'] ?>" class="btn btn-xs btn-outline" title="下载 .dat 文件" style="text-decoration:none;">⬇</a>
            <button class="btn btn-xs btn-outline" onclick="renewLicense(<?= $l['id'] ?>,'<?= htmlspecialchars($l['customer_name']) ?>','<?= $l['expires_at'] ?>')" title="续期">⏳</button>
          <?php endif ?>
          <?php if($l['status']==3): ?>
          <form method="POST" style="display:inline;"><input type="hidden" name="action" value="toggle"><input type="hidden" name="id" value="<?=$l['id']?>"><input type="hidden" name="status" value="2"><button class="btn btn-xs btn-success" title="启用">启用</button></form>
          <?php else: ?>
          <form method="POST" style="display:inline;"><input type="hidden" name="action" value="toggle"><input type="hidden" name="id" value="<?=$l['id']?>"><input type="hidden" name="status" value="3"><button class="btn btn-xs btn-outline-danger" title="停用">停用</button></form>
          <?php endif ?>
          <form method="POST" style="display:inline;" onsubmit="return confirm('确定删除此授权？此操作不可恢复')"><input type="hidden" name="action" value="delete"><input type="hidden" name="id" value="<?=$l['id']?>"><button class="btn btn-xs btn-danger" title="删除">🗑</button></form>
          </div>
        </td>
      </tr>
      <?php endforeach ?>
      <?php endif ?>
      </tbody>
    </table>
  </div>

  <!-- 分页 -->
  <?php if ($totalPages > 1): ?>
  <div class="pagination">
    <div class="pagination-info">共 <b><?= $totalRows ?? 0 ?></b> 条记录，第 <b><?= $currentPage ?>/<?= $totalPages ?></b> 页</div>
    <div class="pagination-btns">
      <?php
        $baseParams = [];
        if ($searchKeyword) $baseParams['search'] = $searchKeyword;
        if ($filterStatus !== '') $baseParams['status'] = $filterStatus;

        function pageUrl($page, $baseParams) {
          $p = array_merge($baseParams, ['page' => $page]);
          return '?' . http_build_query($p);
        }

        $prev = max(1, $currentPage - 1);
        $next = min($totalPages, $currentPage + 1);
      ?>
      <a href="<?= pageUrl(1, $baseParams) ?>" class="<?= $currentPage == 1 ? 'disabled' : '' ?>">«</a>
      <a href="<?= pageUrl($prev, $baseParams) ?>" class="<?= $currentPage == 1 ? 'disabled' : '' ?>">‹</a>
      <?php
        $start = max(1, $currentPage - 2);
        $end = min($totalPages, $currentPage + 2);
        for ($i = $start; $i <= $end; $i++):
      ?>
        <a href="<?= pageUrl($i, $baseParams) ?>" class="<?= $i == $currentPage ? 'active' : '' ?>"><?= $i ?></a>
      <?php endfor ?>
      <a href="<?= pageUrl($next, $baseParams) ?>" class="<?= $currentPage == $totalPages ? 'disabled' : '' ?>">›</a>
      <a href="<?= pageUrl($totalPages, $baseParams) ?>" class="<?= $currentPage == $totalPages ? 'disabled' : '' ?>">»</a>
    </div>
  </div>
  <?php endif ?>
</div>

<!-- ==================== 弹窗：查看授权数据 ==================== -->
<div class="modal-mask" id="licenseModal">
  <div class="modal-box">
    <h3>📦 授权数据详情</h3>
    <p class="modal-subtitle" id="modalSubtitle"></p>

    <div class="info-grid">
      <div class="info-item">
        <div class="label">客户名称</div>
        <div class="value" id="modalCustomer">-</div>
      </div>
      <div class="info-item">
        <div class="label">到期日期</div>
        <div class="value" id="modalExpires">-</div>
      </div>
      <div class="info-item">
        <div class="label">机器码</div>
        <div class="value" style="font-size:12px;" id="modalMachine">-</div>
      </div>
      <div class="info-item">
        <div class="label">联系方式</div>
        <div class="value" id="modalContact">-</div>
      </div>
    </div>

    <p style="font-size:13px;font-weight:600;margin-bottom:8px;color:var(--text-secondary);">🔐 加密授权数据</p>
    <div class="license-data-preview" id="licenseDataContent">
      <span class="copy-hint">点击下方按钮复制</span>
    </div>

    <div style="display:flex;align-items:center;gap:8px;margin-bottom:8px;">
      <span style="font-size:12px;color:var(--text-muted);">💡 将此数据发给客户，保存为 <code style="background:#f1f5f9;padding:2px 6px;border-radius:4px;">license.dat</code> 后导入客户端即可激活</span>
    </div>

    <div class="modal-actions">
      <a class="btn btn-success btn-sm" id="modalDownload" style="text-decoration:none;">⬇ 下载 .dat 文件</a>
      <button class="btn btn-primary btn-sm" onclick="copyLicenseData()">📋 复制授权数据</button>
      <button class="btn btn-outline btn-sm" onclick="closeModal()">关闭</button>
    </div>
  </div>
</div>

<!-- ==================== 弹窗：续期 ==================== -->
<div class="modal-mask" id="renewModal">
  <div class="modal-box" style="max-width: 480px;">
    <h3>⏳ 续期授权</h3>
    <p class="modal-subtitle" id="renewSubtitle"></p>
    <form method="POST" onsubmit="return submitRenew()">
      <input type="hidden" name="action" value="renew">
      <input type="hidden" name="id" id="renewId">
      <input type="hidden" name="_nonce" id="renewNonce">
      <div class="renew-form">
        <div class="form-group" style="flex:1">
          <label>续期天数</label>
          <input name="days" type="number" id="renewDays" value="" placeholder="如30" min="1" max="3650" required autofocus>
        </div>
        <button class="btn btn-warning" id="renewBtn">确认续期</button>
      </div>
    </form>
    <div class="modal-actions" style="margin-top:16px;">
      <button class="btn btn-outline btn-sm" onclick="document.getElementById('renewModal').classList.remove('active')">取消</button>
    </div>
  </div>
</div>

<!-- ==================== JavaScript ==================== -->
<script>
// ========== 签发区域折叠 ==========
function toggleIssueSection() {
  const body = document.getElementById('issueBody');
  const btn = document.getElementById('toggleIssueBtn');
  if (body.style.display === 'none') {
    body.style.display = '';
    btn.textContent = '收起 ▲';
  } else {
    body.style.display = 'none';
    btn.textContent = '展开 ▼';
  }
}

// ========== 搜索与筛选 ==========
function doSearch() {
  const search = document.getElementById('searchInput').value.trim();
  const status = document.getElementById('filterSelect').value;
  let url = '?';
  const params = [];
  if (search) params.push('search=' + encodeURIComponent(search));
  if (status !== '') params.push('status=' + status);
  url += params.join('&');
  window.location.href = url || '?';
}

function filterStatus(s) {
  document.getElementById('filterSelect').value = s;
  doSearch();
}

// ========== 查看授权数据弹窗 ==========
let currentLicenseData = '';
let currentLicenseId = '';

function viewLicense(id) {
  fetch('', {
    method: 'POST',
    headers: {'Content-Type': 'application/x-www-form-urlencoded'},
    body: 'action=view&id=' + id
  })
  .then(r => r.json())
  .then(data => {
    if (data.ok) {
      currentLicenseData = data.license_data;
      currentLicenseId = id;
      document.getElementById('modalSubtitle').textContent = 'ID: #' + id;
      document.getElementById('modalCustomer').textContent = data.customer_name || '-';
      document.getElementById('modalExpires').textContent = data.expires_at || '-';
      document.getElementById('modalMachine').textContent = data.machine_code || '-';
      document.getElementById('modalContact').textContent = data.contact_info || '-';
      document.getElementById('licenseDataContent').textContent = data.license_data;
      document.getElementById('modalDownload').href = '?action=download&id=' + id;
      document.getElementById('licenseModal').classList.add('active');
    } else {
      showToast(data.msg || '获取失败', 'error');
    }
  });
}

function closeModal() {
  document.getElementById('licenseModal').classList.remove('active');
}

function copyLicenseData() {
  if (!currentLicenseData) return;
  if (navigator.clipboard && window.isSecureContext) {
    navigator.clipboard.writeText(currentLicenseData).then(() => {
      showToast('已复制到剪贴板！客户可将此内容保存为 license.dat 文件', 'success');
    });
  } else {
    const ta = document.createElement('textarea');
    ta.value = currentLicenseData;
    ta.style.position = 'fixed'; ta.style.opacity = '0';
    document.body.appendChild(ta);
    ta.select();
    document.execCommand('copy');
    document.body.removeChild(ta);
    showToast('已复制到剪贴板！', 'success');
  }
}

// ========== 续期弹窗 ==========
function renewLicense(id, name, expires) {
  document.getElementById('renewId').value = id;
  document.getElementById('renewSubtitle').textContent = '客户: ' + name + ' | 当前到期: ' + expires;
  document.getElementById('renewDays').value = '';
  document.getElementById('renewNonce').value = Date.now() + '_' + Math.random().toString(36).substr(2);
  document.getElementById('renewBtn').disabled = false;
  document.getElementById('renewBtn').textContent = '确认续期';
  document.getElementById('renewModal').classList.add('active');
}
function submitRenew() {
  var btn = document.getElementById('renewBtn');
  var days = document.getElementById('renewDays').value;
  if (!days || parseInt(days) < 1) { return false; }
  btn.disabled = true;
  btn.textContent = '处理中...';
  return true;
}

// ========== Toast 提示 ==========
function showToast(message, type) {
  const existing = document.querySelector('.toast');
  if (existing) existing.remove();

  const toast = document.createElement('div');
  toast.className = 'toast ' + (type || 'success');
  toast.innerHTML = '<span>' + (type === 'error' ? '⚠️' : '✅') + '</span>' + message;
  document.body.appendChild(toast);

  setTimeout(() => {
    toast.style.opacity = '0';
    toast.style.transform = 'translateX(100px)';
    toast.style.transition = 'all .3s ease';
    setTimeout(() => toast.remove(), 300);
  }, 2500);
}

// ========== 模态框遮罩点击关闭 ==========
document.getElementById('licenseModal').addEventListener('click', function(e) {
  if (e.target === this) closeModal();
});
document.getElementById('renewModal').addEventListener('click', function(e) {
  if (e.target === this) document.getElementById('renewModal').classList.remove('active');
});

// ========== 全选/批量操作 ==========
function toggleSelectAll() {
  const master = document.getElementById('selectAll');
  document.querySelectorAll('.row-checkbox').forEach(cb => { cb.checked = master.checked; });
  updateBatchBtns();
}

function updateBatchBtns() {
  const checked = document.querySelectorAll('.row-checkbox:checked').length;
  const show = checked > 0;
  document.getElementById('batchEnableBtn').style.display = show ? '' : 'none';
  document.getElementById('batchDisableBtn').style.display = show ? '' : 'none';
  document.getElementById('batchDeleteBtn').style.display = show ? '' : 'none';
}

function getCheckedIds() {
  return Array.from(document.querySelectorAll('.row-checkbox:checked')).map(cb => cb.value);
}

function batchToggle(status) {
  const ids = getCheckedIds();
  if (!ids.length) return;
  const action = status == 2 ? '启用' : '停用';
  if (!confirm('确定批量' + action + ' ' + ids.length + ' 条授权记录？')) return;

  const form = document.createElement('form');
  form.method = 'POST';
  form.innerHTML = '<input name="action" value="batch_toggle"><input name="ids" value="' + encodeURIComponent(JSON.stringify(ids)) + '"><input name="status" value="' + status + '">';
  document.body.appendChild(form);
  form.submit();
}

function batchDelete() {
  const ids = getCheckedIds();
  if (!ids.length) return;
  if (!confirm('⚠️ 确定永久删除 ' + ids.length + ' 条授权记录？此操作不可恢复！')) return;

  const form = document.createElement('form');
  form.method = 'POST';
  form.innerHTML = '<input name="action" value="batch_delete"><input name="ids" value="' + encodeURIComponent(JSON.stringify(ids)) + '">';
  document.body.appendChild(form);
  form.submit();
}

// 初始化批量按钮状态
document.addEventListener('DOMContentLoaded', function() {
  updateBatchBtns();
});
</script>

<?php endif ?>
</div>
</body>
</html>
