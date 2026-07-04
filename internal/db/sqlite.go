package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

// writeMu 全局写锁，SQLite并发写入会报SQLITE_BUSY，串行写入避免锁冲突
var writeMu sync.Mutex

// WriteExec 串行执行写操作，避免SQLITE_BUSY
func WriteExec(query string, args ...interface{}) (sql.Result, error) {
	writeMu.Lock()
	defer writeMu.Unlock()
	return DB.Exec(query, args...)
}

func GetDataDir() string {
	if envDir := os.Getenv("YUNFEI_DATA_DIR"); envDir != "" {
		os.MkdirAll(envDir, 0755)
		return envDir
	}
	var base string
	if runtime.GOOS == "windows" {
		base = os.Getenv("APPDATA")
	} else {
		base = os.Getenv("HOME")
	}
	dir := filepath.Join(base, "yunfei")
	os.MkdirAll(dir, 0755)
	return dir
}

func Init() error {
	dbPath := filepath.Join(GetDataDir(), "yunfei.db")
	var err error
	DB, err = sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=30000")
	if err != nil {
		return err
	}
	DB.SetMaxOpenConns(10)
	return migrate()
}

func migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS freight_rules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		rule_type TEXT NOT NULL DEFAULT 'customer',
		customer_name TEXT DEFAULT '',
		province TEXT DEFAULT '',
		cont_mode TEXT NOT NULL DEFAULT 'full_kg',
		first_weight REAL NOT NULL DEFAULT 1.0,
		first_price REAL NOT NULL DEFAULT 5.0,
		cont_price REAL NOT NULL DEFAULT 2.0,
		min_fee REAL DEFAULT 0,
		max_fee REAL DEFAULT 0,
		surcharge REAL DEFAULT 0,
		campaign_name TEXT DEFAULT '',
		campaign_start TEXT DEFAULT '',
		campaign_end TEXT DEFAULT '',
		is_enabled INTEGER DEFAULT 1,
		remark TEXT DEFAULT '',
		created_at TEXT DEFAULT (datetime('now','localtime')),
		updated_at TEXT DEFAULT (datetime('now','localtime'))
	);

	CREATE TABLE IF NOT EXISTS calc_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		input_file TEXT NOT NULL DEFAULT '',
		output_file TEXT DEFAULT '',
		total_count INTEGER DEFAULT 0,
		total_fee REAL DEFAULT 0,
		avg_fee REAL DEFAULT 0,
		max_fee REAL DEFAULT 0,
		min_fee REAL DEFAULT 0,
		rule_summary TEXT DEFAULT '',
		calc_duration REAL DEFAULT 0,
		created_at TEXT DEFAULT (datetime('now','localtime'))
	);

	CREATE TABLE IF NOT EXISTS license_info (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		machine_code TEXT NOT NULL DEFAULT '',
		customer_name TEXT DEFAULT '',
		expires_at TEXT NOT NULL DEFAULT '',
		issued_at TEXT DEFAULT '',
		features TEXT DEFAULT '',
		license_raw TEXT DEFAULT '',
		last_verify_at TEXT DEFAULT '',
		created_at TEXT DEFAULT (datetime('now','localtime'))
	);

	CREATE TABLE IF NOT EXISTS app_settings (
		key TEXT PRIMARY KEY,
		value TEXT DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS global_rules (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		default_first_weight REAL NOT NULL DEFAULT 1.0,
		default_first_price REAL NOT NULL DEFAULT 5.0,
		default_cont_price REAL NOT NULL DEFAULT 2.0,
		default_min_fee REAL DEFAULT 0,
		no_weight_price REAL DEFAULT 5.0,
		markup_fixed REAL DEFAULT 0,
		markup_percent REAL DEFAULT 0,
		updated_at TEXT DEFAULT (datetime('now','localtime'))
	);

	-- ===== 区域表 =====
	CREATE TABLE IF NOT EXISTS freight_zones (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		zone_name TEXT NOT NULL DEFAULT '',
		zone_order INTEGER DEFAULT 0,
		remark TEXT DEFAULT '',
		created_at TEXT DEFAULT (datetime('now','localtime')),
		updated_at TEXT DEFAULT (datetime('now','localtime'))
	);

	-- ===== 区域-省份映射表 =====
	CREATE TABLE IF NOT EXISTS freight_zone_provinces (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		zone_id INTEGER NOT NULL DEFAULT 0,
		province_name TEXT NOT NULL DEFAULT ''
	);

	-- ===== 重量区间表 =====
	CREATE TABLE IF NOT EXISTS freight_weight_brackets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		rule_id INTEGER NOT NULL DEFAULT 0,
		weight_from REAL NOT NULL DEFAULT 0,
		weight_to REAL NOT NULL DEFAULT 0,
		calc_type TEXT NOT NULL DEFAULT 'fixed',
		fixed_price REAL DEFAULT 0,
		first_weight REAL DEFAULT 0,
		first_price REAL DEFAULT 0,
		cont_price REAL DEFAULT 0,
		cont_mode TEXT DEFAULT 'full_kg',
		sort_order INTEGER DEFAULT 0
	);

	-- ===== 全局省份加价表 =====
	CREATE TABLE IF NOT EXISTS global_province_surcharges (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		province_name TEXT NOT NULL DEFAULT '',
		surcharge REAL NOT NULL DEFAULT 0,
		remark TEXT DEFAULT '',
		created_at TEXT DEFAULT (datetime('now','localtime'))
	);

	-- ===== 拉均重规则表 =====
	CREATE TABLE IF NOT EXISTS avg_weight_rules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		scope_type TEXT NOT NULL DEFAULT 'global',
		customer_name TEXT DEFAULT '',
		base_weight REAL NOT NULL DEFAULT 0.3,
		step_weight REAL NOT NULL DEFAULT 0.1,
		step_price REAL NOT NULL DEFAULT 0.1,
		max_markup REAL DEFAULT 0,
		round_mode TEXT DEFAULT 'ceil',
		is_enabled INTEGER DEFAULT 1,
		remark TEXT DEFAULT '',
		created_at TEXT DEFAULT (datetime('now','localtime')),
		updated_at TEXT DEFAULT (datetime('now','localtime'))
	);
	`
	_, err := DB.Exec(schema)
	if err != nil {
		return err
	}

	// ===== 字段迁移：为旧表添加新字段 =====
	migrateAddColumn("freight_rules", "calc_mode", "TEXT NOT NULL DEFAULT 'simple'")
	migrateAddColumn("freight_rules", "zone_id", "INTEGER DEFAULT 0")
	migrateAddColumn("avg_weight_rules", "weight_limit", "REAL DEFAULT 0")

	seedGlobalRules()
	seedDefaults()
	seedZones()
	seedAvgWeightRule()
	return nil
}

// migrateAddColumn 安全地添加列（如果不存在）
func migrateAddColumn(table, column, def string) {
	// 先检查列是否存在
	rows, err := DB.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return
	}
	defer rows.Close()
	exists := false
	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dfltValue interface{}
		var pk int
		rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk)
		if name == column {
			exists = true
			break
		}
	}
	if exists {
		return
	}
	DB.Exec("ALTER TABLE " + table + " ADD COLUMN " + column + " " + def)
}

func seedGlobalRules() {
	var cnt int
	DB.QueryRow("SELECT COUNT(*) FROM global_rules").Scan(&cnt)
	if cnt > 0 {
		return
	}
	DB.Exec(`INSERT INTO global_rules (id) VALUES (1)`)
}

func seedDefaults() error {
	var cnt int
	DB.QueryRow("SELECT COUNT(*) FROM freight_rules WHERE rule_type='default'").Scan(&cnt)
	if cnt > 0 {
		return nil
	}
	// 默认全国规则（全续）
	_, err := DB.Exec(`INSERT INTO freight_rules (rule_type, province, cont_mode, calc_mode, first_weight, first_price, cont_price, remark) 
		VALUES ('default', '', 'full_kg', 'simple', 1.0, 5.0, 2.0, '系统默认规则-全续')`)
	return err
}

func seedZones() {
	// 由 rules 包负责初始化（避免循环引用）
	// 通过调用 rules.InitDefaultZones() 完成
	// 这里不直接调用，在应用启动时由 app 层调用
}

func seedAvgWeightRule() {
	var cnt int
	DB.QueryRow("SELECT COUNT(*) FROM avg_weight_rules").Scan(&cnt)
	if cnt > 0 {
		return
	}
	DB.Exec(`INSERT INTO avg_weight_rules 
		(scope_type, customer_name, base_weight, step_weight, step_price, max_markup, round_mode, is_enabled, remark)
		VALUES ('global', '', 0.3, 0.1, 0.1, 0, 'ceil', 0, '系统默认拉均重规则（默认关闭）')`)
}

func Close() {
	if DB != nil {
		DB.Close()
	}
}
