<?php
/**
 * 初始化数据库表 (RSA+AES加密版)
 * 部署后访问一次: https://你的域名/yunfei_api/init.php
 */

header('Content-Type: application/json; charset=utf-8');

// ========== 数据库配置 ==========
$DB_HOST = '127.0.0.1';
$DB_PORT = '3306';
$DB_USER = 'root';
$DB_PASS = 'cxdxfx12';
$DB_NAME = 'dasheng';

// ========== API 密钥 (客户端签名用) ==========
define('API_SECRET', 'yunfei_server_2024_!@#');

$results = [];

try {
    $pdo = new PDO(
        "mysql:host=$DB_HOST;port=$DB_PORT;dbname=$DB_NAME;charset=utf8mb4",
        $DB_USER, $DB_PASS,
        [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION]
    );

    // ========== 建表（包含新字段） ==========
    $pdo->exec("CREATE TABLE IF NOT EXISTS yunfei_licenses (
        id INT AUTO_INCREMENT PRIMARY KEY,
        license_key VARCHAR(32) DEFAULT '' COMMENT '旧版激活码(保留兼容)',
        machine_code VARCHAR(64) DEFAULT '' COMMENT '绑定的机器码',
        customer_name VARCHAR(100) DEFAULT '' COMMENT '客户名称',
        contact_info VARCHAR(200) DEFAULT '' COMMENT '联系方式',
        expires_at DATE NOT NULL COMMENT '到期日期',
        license_data TEXT COMMENT '加密授权数据(RSA+AES-256-GCM)',
        issued_at DATETIME DEFAULT NULL COMMENT '签发时间',
        duration_days INT DEFAULT 0 COMMENT '授权天数',
        remark VARCHAR(500) DEFAULT '' COMMENT '备注',
        activated_at DATETIME DEFAULT NULL COMMENT '激活时间',
        last_check_at DATETIME DEFAULT NULL COMMENT '最后验证时间',
        status TINYINT DEFAULT 1 COMMENT '1=未激活 2=已激活/已签发 3=已停用',
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        INDEX idx_key (license_key),
        INDEX idx_machine (machine_code)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='喵喵云结算授权记录'");
    $results[] = '表 yunfei_licenses 已确认';

    // ========== 兼容迁移：为旧表添加新字段 ==========
    $existingCols = [];
    foreach ($pdo->query("SHOW COLUMNS FROM yunfei_licenses") as $col) {
        $existingCols[$col['Field']] = true;
    }

    $migrations = [
        'license_data'  => "ALTER TABLE yunfei_licenses ADD COLUMN license_data TEXT COMMENT '加密授权数据' AFTER expires_at",
        'issued_at'     => "ALTER TABLE yunfei_licenses ADD COLUMN issued_at DATETIME DEFAULT NULL COMMENT '签发时间' AFTER license_data",
        'duration_days' => "ALTER TABLE yunfei_licenses ADD COLUMN duration_days INT DEFAULT 0 COMMENT '授权天数' AFTER issued_at",
        'contact_info'  => "ALTER TABLE yunfei_licenses ADD COLUMN contact_info VARCHAR(200) DEFAULT '' COMMENT '联系方式' AFTER customer_name",
        'remark'        => "ALTER TABLE yunfei_licenses ADD COLUMN remark VARCHAR(500) DEFAULT '' COMMENT '备注' AFTER duration_days",
    ];

    foreach ($migrations as $col => $sql) {
        if (!isset($existingCols[$col])) {
            $pdo->exec($sql);
            $results[] = "已添加字段: {$col}";
        }
    }

    // ========== 设置表 ==========
    $pdo->exec("CREATE TABLE IF NOT EXISTS yunfei_settings (
        `key` VARCHAR(50) PRIMARY KEY,
        `value` TEXT
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4");
    $results[] = '表 yunfei_settings 已确认';

    // 写入 API 密钥
    $stmt = $pdo->prepare("INSERT INTO yunfei_settings (`key`,`value`) VALUES ('api_secret',?) ON DUPLICATE KEY UPDATE `value`=VALUES(`value`)");
    $stmt->execute([API_SECRET]);
    $results[] = 'API密钥已写入';

    echo json_encode([
        'ok' => true,
        'msg' => '数据库初始化完成',
        'details' => $results,
        'encryption' => 'RSA签名 + AES-256-GCM',
        'api_secret' => API_SECRET,
    ], JSON_UNESCAPED_UNICODE | JSON_PRETTY_PRINT);

} catch (Exception $e) {
    echo json_encode(['ok' => false, 'msg' => $e->getMessage()], JSON_UNESCAPED_UNICODE);
}
