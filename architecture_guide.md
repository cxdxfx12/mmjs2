---
name: 架构开发指南
description: 喵喵云结算项目完整架构开发指南，包含技术栈、模块设计、数据模型、API接口和开发规范
type: project
---

# 喵喵云结算 (mmjs2-main) 架构开发指南

> 版本: v1.3.0+bugfix+performance
> 更新日期: 2026-07-05

---

## 目录

1. [项目概述](#1-项目概述)
2. [技术栈](#2-技术栈)
3. [目录结构](#3-目录结构)
4. [后端架构](#4-后端架构)
5. [前端架构](#5-前端架构)
6. [数据模型](#6-数据模型)
7. [API接口设计](#7-api接口设计)
8. [核心流程](#8-核心流程)
9. [性能优化](#9-性能优化)
10. [开发规范](#10-开发规范)

---

## 1. 项目概述

### 1.1 项目简介

**喵喵云结算** 是一个快递运费结算桌面应用软件，支持批量处理 Excel 运单数据，按照配置的规则自动计算运费。

### 1.2 核心功能

| 功能 | 描述 |
|------|------|
| 批量计算 | 支持百万行级 Excel 并行计算 |
| 规则管理 | 客户规则、活动规则、全局规则、区域规则 |
| 双计费模式 | 传统首重续重 + 重量区间计费 |
| 拉均重加价 | 按客户平均重量偏差自动加价 |
| 授权管理 | RSA+AES 加密授权，支持离线验证 |
| 数据导出 | 计算结果和规则导出为 Excel |

---

## 2. 技术栈

### 2.1 技术选型

| 层级 | 技术 | 说明 |
|------|------|------|
| **桌面框架** | Wails v2 | Go + WebView2 跨平台桌面应用 |
| **后端** | Go 1.22 | 高性能计算引擎 |
| **数据库** | SQLite | modernc.org/sqlite 纯 Go 实现 |
| **Excel** | excelize/v2 | Go 高性能 Excel 读写 |
| **前端** | Vue 3 + TypeScript | 响应式 UI |
| **UI组件** | Element Plus 2.9 | 企业级组件库 |
| **状态管理** | Pinia 2.3 | Vue3 官方状态管理 |
| **路由** | Vue Router 4.5 | Hash 模式路由 |
| **构建** | Vite 6.0 | 前端构建工具 |

### 2.2 部署模式

```
┌─────────────────────────────────────────────┐
│           Windows 桌面应用 (.exe)            │
│  ┌─────────────────────────────────────┐    │
│  │           WebView2 嵌入式浏览器        │    │
│  │  ┌─────────────────────────────────┐│    │
│  │  │      Vue 3 前端 (嵌入式)         ││    │
│  │  └─────────────────────────────────┘│    │
│  └─────────────────────────────────────┘    │
│  ┌─────────────────────────────────────┐    │
│  │         Go 后端 (同一进程)            │    │
│  │  ┌──────────┐  ┌──────────┐        │    │
│  │  │ HTTP API │  │ 计算引擎  │        │    │
│  │  └──────────┘  └──────────┘        │    │
│  │  ┌──────────┐  ┌──────────┐        │    │
│  │  │ SQLite   │  │ Excel IO │        │    │
│  │  └──────────┘  └──────────┘        │    │
│  └─────────────────────────────────────┘    │
└─────────────────────────────────────────────┘
```

---

## 3. 目录结构

```
mmjs2-main/
│
├── main.go                     # 主入口 (1051行)
│                               # - HTTP 服务器启动
│                               # - 所有 API 路由注册
│                               # - Token 认证
│                               # - 前端静态资源嵌入
│                               # - 单实例检测
│
├── wails.json                  # Wails 配置
├── go.mod / go.sum             # Go 依赖
├── browser_*.go               # 跨平台打开浏览器
│
├── internal/                    # ========== Go 内部包 ==========
│   │
│   ├── app/
│   │   └── app.go             # 应用核心 (1034行)
│   │                           # 业务逻辑封装层
│   │                           # 隔离 HTTP 处理和核心计算
│   │
│   ├── db/
│   │   └── sqlite.go          # 数据库层 (263行)
│   │                           # - SQLite 初始化
│   │                           # - 数据迁移
│   │                           # - WAL 模式配置
│   │
│   ├── rules/                  # ========== 规则管理模块 ==========
│   │   ├── models.go          # 数据模型定义 (94行)
│   │   ├── repository.go      # 规则 CRUD + RuleIndex (661行)
│   │   ├── global.go          # 全局规则 + 省份加价 (104行)
│   │   ├── zones.go           # 区域管理 (484行)
│   │   ├── brackets.go        # 重量区间 (121行)
│   │   ├── province.go         # 省份名称归一化 (27行)
│   │   └── avgweight.go       # 拉均重规则 (124行)
│   │
│   ├── freight/                # ========== 运费计算引擎 ==========
│   │   ├── engine.go          # 核心计算逻辑 (216行)
│   │   └── avgweight.go       # 拉均重偏差加价 (190行)
│   │
│   ├── excel/
│   │   └── reader.go          # Excel 处理 (874行)
│   │                           # - ZIP 直读快速预览
│   │                           # - 流式读取
│   │                           # - 结果写入
│   │
│   └── license/                # ========== 授权系统 ==========
│       ├── crypto.go           # RSA+AES 加密
│       ├── validator.go        # 本地授权验证
│       ├── online.go           # 在线授权
│       ├── machine.go         # 机器码生成
│       ├── machine_windows.go # Windows 磁盘序列号
│       └── machine_unix.go    # Unix 磁盘序列号
│
├── frontend/                    # ========== Vue 3 前端 ==========
│   ├── src/
│   │   ├── main.ts           # 应用入口
│   │   ├── App.vue           # 根组件
│   │   ├── style.css         # 全局样式
│   │   │
│   │   ├── api/
│   │   │   └── index.ts      # API 请求封装
│   │   │
│   │   ├── router/
│   │   │   └── index.ts      # 路由配置 + 导航守卫
│   │   │
│   │   ├── stores/
│   │   │   └── app.ts        # Pinia 状态管理 (446行)
│   │   │
│   │   ├── views/            # 页面视图
│   │   │   ├── Home.vue      # 首页
│   │   │   ├── Calc.vue      # 计费结算
│   │   │   ├── Rules.vue     # 规则管理 (1747行)
│   │   │   ├── TestRule.vue  # 规则测试
│   │   │   ├── History.vue   # 历史记录
│   │   │   ├── License.vue   # 授权管理
│   │   │   ├── Settings.vue  # 系统设置
│   │   │   └── Layout.vue    # 主布局
│   │   │
│   │   └── components/
│   │       └── KnowledgeAI.vue # AI 知识助手
│   │
│   ├── package.json
│   ├── vite.config.ts
│   └── index.html
│
├── tools/
│   └── genkeys.go             # 密钥生成工具
│
└── server/                    # PHP 授权服务器 (参考)
```

---

## 4. 后端架构

### 4.1 模块分层

```
┌─────────────────────────────────────────────────────┐
│                    main.go                           │
│              HTTP Server + Router                    │
│         (认证、路由分发、静态资源服务)                 │
└─────────────────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────┐
│                   app/app.go                        │
│               业务逻辑封装层                         │
│    (规则CRUD、计算调度、历史管理、授权)               │
└─────────────────────────────────────────────────────┘
         │            │            │            │
         ▼            ▼            ▼            ▼
┌────────────┐ ┌────────────┐ ┌────────────┐ ┌────────────┐
│  rules/    │ │  freight/  │ │   excel/   │ │  license/  │
│  规则管理   │ │  计算引擎   │ │  Excel处理  │ │  授权系统   │
└────────────┘ └────────────┘ └────────────┘ └────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────┐
│                    db/sqlite.go                     │
│              SQLite 数据库访问层                      │
│         (连接管理、事务、写锁、WAL模式)              │
└─────────────────────────────────────────────────────┘
```

### 4.2 main.go — 主入口

**核心职责:**
1. HTTP 服务器启动 (默认端口 58080)
2. 60+ 个 API 路由注册
3. 简单 Token 认证 (小时级有效)
4. 前端静态资源嵌入 (`//go:embed`)
5. 单实例检测 (防止重复启动)
6. 批量计算任务管理

**路由分组:**

| 分组 | 路径前缀 | 功能 |
|------|----------|------|
| 认证 | `/api/auth/` | login, verify |
| 授权 | `/api/license/` | 导入、验证、激活、重置 |
| 机器码 | `/api/machine-code` | 获取本机唯一码 |
| 规则 | `/api/rules/` | CRUD、批量删除、详情、测试 |
| 客户 | `/api/customers/` | 列表、删除、复制、导入导出 |
| 区域 | `/api/zones/` | 列表、模板、批量生成 |
| 拉均重 | `/api/avg-weight/` | 规则 CRUD |
| 全局 | `/api/global-rules` | 全局保底+加价 |
| 省份加价 | `/api/province-surcharges/` | CRUD |
| Excel | `/api/excel/` | 上传、预览 |
| 计算 | `/api/calculate/` | 单文件/批量计算 |
| 导出 | `/api/export` | 下载结算结果 |
| 历史 | `/api/history/` | 列表、详情、清除 |
| 设置 | `/api/settings` | 读写配置 |

### 4.3 app/app.go — 应用核心

**业务逻辑封装层**，隔离 HTTP 处理和核心计算。

| 分组 | 方法 | 功能 |
|------|------|------|
| 生命周期 | `New()`, `Startup()`, `Shutdown()` | 初始化 |
| 授权 | `GetMachineCode()`, `ImportLicense()`, `CheckOnlineLicense()`... | 授权管理 |
| 规则 | `GetRules()`, `SaveRule()`, `DeleteRule()`... | 规则 CRUD |
| 客户 | `GetCustomers()`, `CopyCustomerRules()`, `ImportCustomerRules()` | 客户管理 |
| Excel | `ReadExcelPreview()` | 快速预览 |
| **计算** | `CalculateFreightWithProgress()`, `doCalc()` | **核心计算** |
| 拉均重 | `CalcAvgWeightMarkupFast()` | 偏差加价 |
| 导出 | `ExportResult()`, `ExportCustomerRules()` | 结果导出 |

### 4.4 db/sqlite.go — 数据库层

**关键设计:**

```go
// WAL 模式提升并发读性能
DB, _ = sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=30000")

// 全局写锁避免 SQLITE_BUSY
var writeMu sync.Mutex
func WriteExec(query string, args ...interface{}) (sql.Result, error) {
    writeMu.Lock()
    defer writeMu.Unlock()
    return DB.Exec(query, args...)
}
```

**数据目录:**
- Windows: `%APPDATA%/yunfei`
- Unix: `~/.yunfei`

### 4.5 rules/ — 规则管理模块

| 文件 | 行数 | 职责 |
|------|------|------|
| `models.go` | 94 | 数据模型定义 |
| `repository.go` | 661 | 规则 CRUD + **RuleIndex O(1) 查找** |
| `global.go` | 104 | 全局规则 + 省份加价 |
| `zones.go` | 484 | 六区模板 + 批量生成规则 |
| `brackets.go` | 121 | 重量区间 CRUD |
| `province.go` | 27 | 省份名称归一化 |
| `avgweight.go` | 124 | 拉均重规则 CRUD |

### 4.6 freight/ — 计算引擎

| 文件 | 行数 | 职责 |
|------|------|------|
| `engine.go` | 216 | 核心计算逻辑 |
| `avgweight.go` | 190 | 拉均重偏差加价 |

### 4.7 license/ — 授权系统

```
授权流程:
┌──────────┐     ┌──────────┐     ┌──────────┐
│ 客户端    │────▶│ 服务器    │────▶│ 加密授权  │
│ 生成机器码 │     │ 生成授权  │     │ 文件      │
└──────────┘     └──────────┘     └──────────┘
                                            │
                                            ▼
┌──────────┐     ┌──────────┐     ┌──────────┐
│ 本地验证  │◀────│ 解密验证  │◀────│ 导入授权  │
│ 离线可用  │     │ RSA+AES  │     │          │
└──────────┘     └──────────┘     └──────────┘
```

---

## 5. 前端架构

### 5.1 页面视图

| 页面 | 文件 | 功能 |
|------|------|------|
| **Login** | Login.vue | 登录页 |
| **Layout** | Layout.vue | 主布局 (深色侧边栏) |
| **Home** | Home.vue | 首页 (统计、快捷操作) |
| **Calc** | Calc.vue | 4步计费流程 |
| **Rules** | Rules.vue | 规则管理 (最复杂页面) |
| **TestRule** | TestRule.vue | 规则测试 |
| **History** | History.vue | 历史记录 |
| **License** | License.vue | 授权管理 |
| **Settings** | Settings.vue | 系统设置 |

### 5.2 路由配置

```typescript
// Hash 模式路由 (#/路径)
const routes = [
  { path: '/login', meta: { noAuth: true } },           // 无需认证
  { path: '/', redirect: '/home' },                     // 重定向
  { path: '/calc', meta: { requireLicense: true } },   // 需要授权
  { path: '/rules' },                                  // 普通页面
  // ...
]
```

### 5.3 状态管理 (Pinia)

```typescript
// stores/app.ts - 全局状态
interface AppState {
  license: LicenseInfo      // 授权信息
  rules: FreightRule[]     // 所有规则
  machineCode: string       // 本机码
  calculating: boolean      // 计算中标记
}

// 计算属性
isLicensed: boolean        // 是否已授权
daysLeft: number          // 剩余天数
licenseStatus: string     // active/expiring/expired
```

### 5.4 API 封装

```typescript
// api/index.ts
apiGet(path)           // GET 请求
apiPost(path, body)   // POST JSON
apiUpload(file)        // 上传 (带进度)
apiExport()           // 导出 Excel Blob
```

---

## 6. 数据模型

### 6.1 数据库表结构

```sql
-- ===== 1. 运费规则主表 =====
CREATE TABLE freight_rules (
    id              INTEGER PRIMARY KEY,
    rule_type       TEXT    DEFAULT 'customer',  -- default/customer/campaign/global
    customer_name   TEXT    DEFAULT '',
    province        TEXT    DEFAULT '',           -- 空=全国
    cont_mode       TEXT    DEFAULT 'full_kg',  -- full_kg/hundred_gram/actual_weight
    first_weight    REAL    DEFAULT 1.0,
    first_price     REAL    DEFAULT 5.0,
    cont_price      REAL    DEFAULT 2.0,
    min_fee         REAL    DEFAULT 0,           -- 保底价
    max_fee         REAL    DEFAULT 0,           -- 最高价
    surcharge       REAL    DEFAULT 0,           -- 偏远附加费
    campaign_name   TEXT    DEFAULT '',
    campaign_start  TEXT    DEFAULT '',
    campaign_end    TEXT    DEFAULT '',
    is_enabled      INTEGER DEFAULT 1,
    remark          TEXT    DEFAULT '',
    calc_mode       TEXT    DEFAULT 'simple',     -- simple/bracket
    zone_id         INTEGER DEFAULT 0,
    created_at      TEXT,
    updated_at      TEXT
);

-- ===== 2. 重量区间表 (区间计费模式) =====
CREATE TABLE freight_weight_brackets (
    id            INTEGER PRIMARY KEY,
    rule_id       INTEGER,
    weight_from   REAL    DEFAULT 0,  -- 起始重量(kg)
    weight_to     REAL    DEFAULT 0,  -- 结束重量(kg)，0=无上限
    calc_type     TEXT    DEFAULT 'fixed',  -- fixed/first_cont
    fixed_price   REAL    DEFAULT 0,
    first_weight  REAL    DEFAULT 0,
    first_price   REAL    DEFAULT 0,
    cont_price    REAL    DEFAULT 0,
    cont_mode     TEXT    DEFAULT 'full_kg',
    sort_order    INTEGER DEFAULT 0
);

-- ===== 3. 计算历史 =====
CREATE TABLE calc_history (
    id              INTEGER PRIMARY KEY,
    input_file      TEXT,
    output_file     TEXT,
    total_count     INTEGER,
    total_fee       REAL,
    avg_fee         REAL,
    max_fee         REAL,
    min_fee         REAL,
    rule_summary    TEXT,
    calc_duration   REAL,
    created_at      TEXT
);

-- ===== 4. 全局规则 =====
CREATE TABLE global_rules (
    id                    INTEGER PRIMARY KEY CHECK (id = 1),
    default_first_weight REAL    DEFAULT 1.0,
    default_first_price  REAL    DEFAULT 5.0,
    default_cont_price   REAL    DEFAULT 2.0,
    default_min_fee      REAL    DEFAULT 0,
    no_weight_price      REAL    DEFAULT 5.0,
    markup_fixed         REAL    DEFAULT 0,     -- 固定加价
    markup_percent       REAL    DEFAULT 0,     -- 百分比加价
    updated_at           TEXT
);

-- ===== 5. 区域表 =====
CREATE TABLE freight_zones (
    id          INTEGER PRIMARY KEY,
    zone_name   TEXT,
    zone_order  INTEGER,
    remark      TEXT,
    created_at  TEXT,
    updated_at  TEXT
);

-- ===== 6. 拉均重规则 =====
CREATE TABLE avg_weight_rules (
    id            INTEGER PRIMARY KEY,
    scope_type    TEXT    DEFAULT 'global',  -- global/customer
    customer_name TEXT    DEFAULT '',
    base_weight   REAL    DEFAULT 0.3,      -- 基准重量
    weight_limit  REAL    DEFAULT 0,        -- 重量上限，0=不限制
    step_weight   REAL    DEFAULT 0.1,      -- 偏差步长
    step_price    REAL    DEFAULT 0.1,      -- 每步加价
    max_markup    REAL    DEFAULT 0,        -- 单件最高加价
    round_mode    TEXT    DEFAULT 'ceil',
    is_enabled    INTEGER DEFAULT 1,
    remark        TEXT,
    created_at    TEXT,
    updated_at    TEXT
);

-- ===== 7. 全局省份加价 =====
CREATE TABLE global_province_surcharges (
    id            INTEGER PRIMARY KEY,
    province_name TEXT,
    surcharge     REAL,
    remark        TEXT,
    created_at    TEXT
);
```

### 6.2 核心数据结构

```go
// FreightRule 运费规则
type FreightRule struct {
    ID, ZoneID           int64
    RuleType            string   // default/customer/campaign/global
    CustomerName         string
    Province             string   // 空=全国
    ContMode             string   // full_kg/hundred_gram/actual_weight
    FirstWeight          float64
    FirstPrice           float64
    ContPrice            float64
    MinFee, MaxFee       float64
    Surcharge            float64
    CampaignName         string
    CampaignStart, End   string   // YYYY-MM-DD
    IsEnabled            int
    CalcMode             string   // simple/bracket
    Brackets             []WeightBracket
}

// WeightBracket 重量区间
type WeightBracket struct {
    WeightFrom, WeightTo float64  // 0表示无上限
    CalcType             string   // fixed/first_cont
    FixedPrice           float64
    FirstWeight          float64
    FirstPrice           float64
    ContPrice            float64
    ContMode             string
}

// AvgWeightRule 拉均重规则
type AvgWeightRule struct {
    ScopeType    string   // global/customer
    BaseWeight   float64  // 基准重量，低于此值加价
    WeightLimit  float64  // 重量上限，0=不限制
    StepWeight   float64  // 偏差步长
    StepPrice    float64  // 每步加价
    MaxMarkup    float64  // 单件最高加价
    RoundMode    string   // ceil/floor/round
}

// RuleIndex O(1)规则查找索引
type RuleIndex struct {
    flatRules     map[string]RuleResult  // "客户|省份" -> 规则
    globalRules   map[string]RuleResult  // 省份 -> 全局规则
    defaultResult RuleResult              // 兜底规则
}
```

---

## 7. API接口设计

### 7.1 认证

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/auth/login` | 登录，获取 Token |
| GET | `/api/auth/verify` | 验证 Token 有效性 |

### 7.2 授权

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/license/import` | 导入授权文件 |
| GET | `/api/license/info` | 获取授权信息 |
| POST | `/api/license/check-online` | 在线验证 |
| POST | `/api/license/activate-online` | 在线激活 |
| POST | `/api/license/reset` | 重置授权 |
| GET | `/api/machine-code` | 获取机器码 |

### 7.3 规则

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/rules` | 获取规则列表 |
| POST | `/api/rules/save` | 保存规则 |
| POST | `/api/rules/delete` | 删除规则 |
| POST | `/api/rules/delete-batch` | 批量删除 |
| GET | `/api/rules/detail` | 获取规则详情 |

### 7.4 计算

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/calculate/single` | 单文件计算 |
| POST | `/api/calculate/batch` | 批量计算 |
| GET | `/api/calculate/progress` | 查询进度 |
| GET | `/api/export` | 导出结果 |

---

## 8. 核心流程

### 8.1 运费计算流程

```
┌──────────────────────────────────────────────────────────────────┐
│                      运费计算完整流程                              │
└──────────────────────────────────────────────────────────────────┘

1. Excel 上传
   │
   ▼
2. ReadExcelPreview (快速预览)
   ├─ ZIP 直读 XML (避免全量解析)
   ├─ 采样前 1000 行
   └─ 检测列映射
   │
   ▼
3. CalculateFreightWithProgress
   │
   ▼
4. 数据预加载 (一次性，避免每行查库)
   ├─ LoadRuleBrackets      → bracketMap
   ├─ GetAllProvinceSurcharges → provSurchargeMap
   └─ LoadAllAvgWeightRules  → avgCustomerRules
   │
   ▼
5. BuildRuleIndex → RuleIndex (O(1) 查找)
   │
   ▼
6. 多核并行计算 (Worker Pool)
   │
   ├─ Worker 1: row[0..N/16)      ─┐
   ├─ Worker 2: row[N/16..N/8)    │
   ├─ Worker 3: row[N/8..N/4)     ├─ CalcSingleWithKeys()
   │  ...                          │
   └─ Worker 16: row[15N/16..N)   ─┘
   │
   ▼
7. CalcSingleWithKeys (单笔计算)
   │
   ├─ idx.FindByKeys() → RuleResult (扁平化 map 查找)
   │
   ├─ doCalcSingle() (核心计算)
   │  │
   │  ├─ bracketMap 查找 (预加载)
   │  ├─ provSurchargeMap 查找 (预加载)
   │  │
   │  └─ 计费模式判断
   │     ├─ simple: calcBySimple()
   │     │  └─ 首重价 + ceil(超重/单位) * 续重单价
   │     │
   │     └─ bracket: calcByBracket()
   │        └─ 按重量区间查找后计算
   │
   └─ ApplyGlobalMarkup() → 最终运费
   │
   ▼
8. CalcAvgWeightMarkupFast (拉均重加价)
   │
   ├─ 按客户分组
   ├─ 计算平均重量
   ├─ 偏差步数 = (基准 - 平均) / 步长
   └─ ApplyAvgWeightToRows() → 分摊到每件
   │
   ▼
9. BuildSummary (多维汇总)
   │
   ├─ 按省份汇总
   ├─ 按客户汇总
   ├─ 按区域汇总
   ├─ 按计费模式汇总
   └─ 按规则级别汇总
   │
   ▼
10. WriteResult (Excel 导出)
    │
    ├─ 结算结果 Sheet
    └─ 汇总统计 Sheet
```

### 8.2 规则优先级

```
┌─────────────────────────────────────────────────────┐
│                   规则优先级 (高 → 低)                │
├─────────────────────────────────────────────────────┤
│  1. campaign (活动规则) - 最高优先级                  │
│     ├─ 精确省份匹配                                   │
│     ├─ 通配省份匹配 (空=全国)                         │
│     └─ 需在活动时间内才生效                           │
│                                                      │
│  2. customer (客户规则)                              │
│     ├─ 精确省份匹配                                   │
│     └─ 通配省份匹配 (空=全国)                         │
│                                                      │
│  3. global (全局规则)                                │
│     ├─ 精确省份匹配                                   │
│     └─ 通配省份匹配 (空=全国)                         │
│                                                      │
│  4. default (默认规则) - 最低优先级，兜底             │
└─────────────────────────────────────────────────────┘
```

### 8.3 计费模式

```go
// 1. simple 模式 (传统首重续重)
func calcBySimple(billWeight float64, r FreightRule) float64 {
    if billWeight <= r.FirstWeight {
        return r.FirstPrice  // 首重一口价
    }
    excess := billWeight - r.FirstWeight
    switch r.ContMode {
    case "hundred_gram":
        units := math.Ceil(excess * 10)  // 每100g
    case "actual_weight":
        units := excess                  // 实际重量
    default: // full_kg
        units := math.Ceil(excess)       // 每kg
    }
    return r.FirstPrice + units * r.ContPrice
}

// 2. bracket 模式 (重量区间计费)
func calcByBracket(billWeight float64, brackets []WeightBracket) float64 {
    bracket := FindBracket(billWeight, brackets)  // 线性查找
    if bracket.CalcType == "fixed" {
        return bracket.FixedPrice  // 一口价
    }
    // first_cont: 首重 + 续重
    return bracket.FirstPrice + ceil((billWeight-bracket.FirstWeight)/步长) * bracket.ContPrice
}
```

---

## 9. 性能优化

### 9.1 已实施的优化

| 优化项 | 实现方式 | 效果 |
|--------|---------|------|
| 规则索引 O(1) | `RuleIndex` 预建索引 | O(N) → O(1) |
| 数据预加载 | bracketMap, provSurchargeMap | 消除热点路径数据库查询 |
| 多核并行 | Worker Pool (最多16核) | 线性扩展 |
| Excel 快速预览 | ZIP 直读 XML | 大文件秒开 |
| 预计算归一化键 | RowData.CustKey/ProvKey | 减少字符串处理 |
| Worker 本地累加 | channel 合并结果 | 消除原子竞争 |
| 扁平化 Map | flatRules 组合键 | 减少 map 层级 |
| 原地过滤 | 切片覆盖复用 | 减少 GC |

### 9.2 待考虑的优化

- 定点数运算 (int64 分 替代 float64 元)
- Excel 预分配容量
- 区间二分查找
- 自适应进度回调

---

## 10. 开发规范

### 10.1 Go 代码规范

```go
// 1. 模块组织
// internal/[模块名]/
//   - models.go      // 数据模型
//   - repository.go // 数据访问
//   - service.go    // 业务逻辑
//   - *.go          // 其他

// 2. 错误处理
func SaveRule(r *FreightRule) (int64, error) {
    // 使用 error 而非 bool 返回
    // 调用方负责处理错误
}

// 3. 数据库操作
func WriteExec(query string, args ...interface{}) (sql.Result, error) {
    writeMu.Lock()
    defer writeMu.Unlock()
    return DB.Exec(query, args...)
}

// 4. 并发安全
// - 写操作加锁 (writeMu)
// - 读操作无锁 (SQLite 读兼容)
// - 批量计算使用 channel 合并结果
```

### 10.2 Vue 代码规范

```vue
<!-- 1. 组件结构 -->
<template>
  <div class="page">
    <!-- 布局区域 -->
    <div class="header">...</div>
    <!-- 内容区域 -->
    <div class="content">...</div>
  </div>
</template>

<script setup lang="ts">
// 导入
import { ref, reactive, computed } from 'vue'

// 类型定义
interface Props {
  title: string
}

// Props
const props = defineProps<Props>()

// Emits
const emit = defineEmits<{
  (e: 'save', data: any): void
}>()

// 响应式数据
const loading = ref(false)
const form = reactive({ ... })

// 计算属性
const isValid = computed(() => form.name !== '')

// 方法
async function save() { ... }

// 生命周期
onMounted(() => { ... })
</script>

<style scoped>
/* 组件样式 */
</style>
```

### 10.3 API 设计规范

```go
// HTTP 响应格式
type Response struct {
    Code    int         `json:"code"`    // 0=成功
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

// 路由注册
mux.HandleFunc("/api/rules/save", func(w http.ResponseWriter, r *http.Request) {
    if r.Method != "POST" {
        writeJSON(w, Response{Code: 405, Message: "method not allowed"})
        return
    }
    var req RuleSaveReq
    json.NewDecoder(r.Body).Decode(&req)
    id := a.SaveRule(req)
    writeJSON(w, Response{Code: 0, Message: "ok", Data: map[string]int64{"id": id}})
})
```

### 10.4 数据库规范

```sql
-- 1. 表名使用下划线分隔
freight_rules
calc_history
avg_weight_rules

-- 2. 字段名使用下划线分隔
customer_name
campaign_start
is_enabled

-- 3. 时间字段存储格式
-- 使用 ISO 8601 格式: YYYY-MM-DD HH:MM:SS
-- 或纯日期格式: YYYY-MM-DD

-- 4. 索引
-- 查询字段应建立索引
CREATE INDEX idx_rules_customer ON freight_rules(customer_name);
CREATE INDEX idx_rules_type ON freight_rules(rule_type);
```

---

## 附录

### A. 关键文件速查

| 功能 | 文件路径 | 行号 |
|------|---------|------|
| HTTP 服务器 | main.go | 1-100 |
| API 路由 | main.go | 200-600 |
| 计算引擎 | internal/freight/engine.go | 50-150 |
| 规则索引 | internal/rules/repository.go | 498-653 |
| Excel 读取 | internal/excel/reader.go | 554-625 |
| 规则管理页面 | frontend/src/views/Rules.vue | 1-1747 |
| 状态管理 | frontend/src/stores/app.ts | 1-446 |
| 路由配置 | frontend/src/router/index.ts | 1-100 |

### B. 配置项

```go
// 端口配置
DefaultPort = "58080"

// 数据目录
// Windows: %APPDATA%/yunfei
// Unix: ~/.yunfei

// 数据库文件
yunfei.db

// 日志文件
yunfei.log
```

### C. 相关文档

- [CHANGELOG.md](../CHANGELOG.md) - 更新日志
- [DESIGN.md](../DESIGN.md) - 设计方案
- [PROJECT_GUIDE.md](../PROJECT_GUIDE.md) - 项目指南
- [BUG 修复记录](../.comate/.../bug_fix_2026_07_05.md) - BUG 修复
- [性能优化记录](../.comate/.../performance_optimization_2026_07_05.md) - 性能优化

---

**文档版本**: v1.0.0
**最后更新**: 2026-07-05
**维护者**: 开发团队
