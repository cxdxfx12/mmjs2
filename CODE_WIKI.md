# 喵喵云结算 (yunfei) - Code Wiki

> 项目名称：喵喵云结算（云费）- 快递运费结算软件  
> 技术栈：Go + Vue3 + Wails + SQLite + PHP  
> 定位：本地独立桌面软件 + 服务器授权验证

---

## 目录

1. [项目概述](#1-项目概述)
2. [整体架构](#2-整体架构)
3. [后端模块详解 (Go)](#3-后端模块详解-go)
4. [前端模块详解 (Vue3)](#4-前端模块详解-vue3)
5. [PHP 服务端授权系统](#5-php-服务端授权系统)
6. [数据库设计](#6-数据库设计)
7. [核心算法与流程](#7-核心算法与流程)
8. [API 接口清单](#8-api-接口清单)
9. [依赖关系](#9-依赖关系)
10. [项目运行方式](#10-项目运行方式)

---

## 1. 项目概述

### 1.1 项目简介

喵喵云结算是一款面向快递行业的运费结算桌面软件，支持批量导入Excel运单数据，根据多级计费规则自动计算运费，并生成结算结果报表。软件采用本地部署 + 云端授权的模式，确保数据安全和授权可控。

### 1.2 核心功能

- **运费计算引擎**：支持百克续和全续两种计费模式，四级规则优先级（活动 > 客户 > 全局 > 默认）
- **Excel 批量处理**：支持大文件流式读写，多核并行计算，百万行数据分钟级处理
- **规则管理**：可视化的规则增删改查，支持客户规则批量导入、复制
- **授权系统**：机器码绑定 + RSA签名 + AES加密，支持离线授权和在线激活
- **历史记录**：自动保存计算历史，支持结果回溯和重新导出
- **多文件并行**：一次最多处理5个Excel文件，并行计算

### 1.3 技术选型理由

| 维度 | 选择 | 原因 |
|---|---|---|
| 桌面框架 | Wails + Go | 体积小(~25MB)、内存占用低、原生性能、反编译难度适中 |
| 前端框架 | Vue3 + TypeScript + Vite | 开发效率高、生态成熟、Element Plus组件丰富 |
| 数据库 | SQLite (modernc.org/sqlite) | 纯Go实现无CGO、本地文件存储、无需额外服务 |
| Excel处理 | excelize + ZIP直读 | 流式读写性能优异，大文件预览秒开 |
| 授权服务 | PHP + MySQL | 部署简单、与现有服务器架构兼容 |

---

## 2. 整体架构

### 2.1 系统架构图

```
┌─────────────────────────────────────────────────────────────┐
│                        客户端 (Wails)                        │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  前端层 (Vue3 + WebView2)                              │  │
│  │  ┌────────┐  ┌──────┐  ┌──────┐  ┌────────┐         │  │
│  │  │ 首页    │  │ 计费  │  │ 规则  │  │ 授权   │  ...    │  │
│  │  └────────┘  └──────┘  └──────┘  └────────┘         │  │
│  └───────────────────────────────────────────────────────┘  │
│                            │ HTTP (localhost:58080)          │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  后端层 (Go 原生编译)                                   │  │
│  │  ┌─────────┐  ┌──────────┐  ┌─────────┐  ┌──────────┐ │  │
│  │  │ app层   │  │ excel模块 │  │ freight │  │ rules模块 │ │  │
│  │  └─────────┘  └──────────┘  └─────────┘  └──────────┘ │  │
│  │  ┌─────────┐  ┌──────────┐  ┌─────────┐               │  │
│  │  │ license │  │  db模块   │  │  HTTP   │               │  │
│  │  └─────────┘  └──────────┘  └─────────┘               │  │
│  └───────────────────────────────────────────────────────┘  │
│                            │                                │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  数据层 (SQLite 本地文件)                               │  │
│  │  freight_rules / calc_history / license_info / ...     │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                              │ HTTPS
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                   授权服务器 (PHP + MySQL)                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │ activate.php │  │ verify.php   │  │ 管理后台          │  │
│  └──────────────┘  └──────────────┘  └──────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 目录结构

```
mmjs-master/
├── main.go                    # 应用入口 + HTTP API路由
├── wails.json                 # Wails 构建配置
├── go.mod / go.sum           # Go 依赖管理
├── DESIGN.md                  # 设计文档
│
├── internal/                  # Go 内部模块
│   ├── app/                   # 应用核心层（门面模式）
│   │   └── app.go
│   ├── db/                    # 数据库层
│   │   └── sqlite.go
│   ├── excel/                 # Excel 读写模块
│   │   └── reader.go
│   ├── freight/               # 运费计算引擎
│   │   └── engine.go
│   ├── rules/                 # 规则管理
│   │   ├── models.go
│   │   ├── repository.go
│   │   └── global.go
│   └── license/               # 授权系统
│       ├── crypto.go
│       ├── machine.go
│       ├── online.go
│       └── validator.go
│
├── frontend/                  # Vue3 前端
│   ├── src/
│   │   ├── api/               # API 封装
│   │   ├── router/            # 路由配置
│   │   ├── stores/            # Pinia 状态管理
│   │   ├── views/             # 页面视图
│   │   ├── App.vue
│   │   └── main.ts
│   ├── package.json
│   ├── vite.config.ts
│   └── tsconfig.json
│
├── server_php/                # PHP 授权服务端
│   ├── init.php               # 数据库初始化
│   ├── activate.php           # 激活接口
│   ├── verify.php             # 验证接口
│   └── admin.php              # 管理后台
│
└── tools/
    └── genkeys.go             # RSA密钥对生成工具
```

---

## 3. 后端模块详解 (Go)

### 3.1 入口层 - main.go

**文件**: [main.go](file:///Users/cxd/mmjs-master/mmjs-master/main.go)

**职责**:
- 应用启动与初始化
- HTTP 服务路由注册
- 静态文件服务（前端dist嵌入）
- CORS 中间件 + 认证中间件
- 单实例检测（端口占用检测）
- 进度追踪与任务状态管理

**核心结构体**:

```go
type TaskProgress struct {
    TaskID    string  // 任务ID
    Phase     string  // reading / calculating / done / error
    Current   int     // 当前处理数
    Total     int     // 总数
    Pct       int     // 百分比
    Message   string  // 状态消息
    Error     string  // 错误信息
    UpdatedAt int64   // 更新时间戳
}
```

**关键全局变量**:
- `progressStore sync.Map` - 任务进度存储（taskID -> *TaskProgress）
- `taskResults sync.Map` - 任务结果存储（taskID -> *CalcResult）
- `batchProgresses sync.Map` - 批量任务进度
- `previewStore sync.Map` - Excel预览缓存

**认证机制**:
- 简单的基于SHA256的Token认证
- Token有效期1小时（支持边界1小时容错）
- 默认账号: admin / admin123（可配置）
- 白名单接口: `/api/auth/*`, 在线授权相关接口

---

### 3.2 应用核心层 - app/app.go

**文件**: [app.go](file:///Users/cxd/mmjs-master/mmjs-master/internal/app/app.go)

**职责**: 门面模式，封装各子模块，提供统一对外接口

**核心结构体**:

| 结构体 | 说明 |
|---|---|
| `App` | 应用核心，持有 context |
| `CalcRequest` | 计算请求（文件路径） |
| `CalcResult` | 计算结果（数据+汇总+错误） |
| `RuleSaveReq` | 规则保存请求参数 |
| `CalcHistory` | 历史记录条目 |

**主要方法分类**:

#### 授权相关
- `GetMachineCode() string` - 获取机器码
- `ImportLicense(b64 string)` - 导入离线授权
- `GetLicenseInfo() *LicenseInfo` - 获取授权信息
- `CheckOnlineLicense()` - 在线检查授权
- `ActivateOnline(licenseKey string)` - 旧版激活码激活
- `ActivateWithLicenseData(licenseData string)` - 新版加密数据激活
- `SetServerURL(url string)` / `SetBackupServerURL(url string)` - 配置授权服务器
- `ResetLicense()` - 重置授权

#### 规则管理
- `GetRules() []FreightRule` - 获取所有规则
- `GetRulesByCustomer(name string)` - 按客户获取规则
- `SaveRule(r RuleSaveReq) int64` - 保存规则
- `DeleteRule(id int64) bool` - 删除规则
- `DeleteRulesBatch(ids []int64) bool` - 批量删除

#### 客户管理
- `GetCustomers() []CustomerInfo` - 获取客户列表
- `DeleteCustomer(name string) bool` - 删除客户
- `CopyCustomerRules(from, to string) int` - 复制客户规则
- `ImportCustomerRules(records [][]string) (int, string)` - 批量导入客户规则

#### 运费计算
- `CalculateFreight(req CalcRequest) *CalcResult` - 同步计算
- `CalculateFreightWithProgress(req, progressFn) *CalcResult` - 带进度计算
- `doCalc(rowData, progress, inputFile) *CalcResult` - 核心计算逻辑（多核并行）
- `ExportResult(data, outputPath, summary) string` - 导出结果

**多核并行计算策略**:
```go
numWorkers := runtime.NumCPU()  // CPU核数，上限16
chunkSize := (total + numWorkers - 1) / numWorkers  // 分块大小
// 使用 sync.WaitGroup + atomic 计数器并发计算
```

---

### 3.3 数据库层 - db/sqlite.go

**文件**: [sqlite.go](file:///Users/cxd/mmjs-master/mmjs-master/internal/db/sqlite.go)

**职责**: SQLite 数据库初始化、迁移、连接管理

**关键配置**:
- 驱动: `modernc.org/sqlite`（纯Go实现，无CGO依赖）
- 日志模式: WAL (Write-Ahead Logging)
- 忙超时: 5000ms
- 最大打开连接: 1（SQLite写并发限制）

**数据目录**:
- Windows: `%APPDATA%/yunfei/yunfei.db`
- macOS/Linux: `$HOME/yunfei/yunfei.db`

**初始化流程**:
1. 打开数据库连接
2. 执行 `migrate()` 创建所有表（IF NOT EXISTS）
3. 初始化全局规则单行数据
4. 植入默认运费规则（如果不存在）

**数据库表**:
- `freight_rules` - 运费规则表
- `calc_history` - 计算历史表
- `license_info` - 授权信息表（单行）
- `app_settings` - 应用设置表（key-value）
- `global_rules` - 全局规则表（单行）

---

### 3.4 Excel 模块 - excel/reader.go

**文件**: [reader.go](file:///Users/cxd/mmjs-master/mmjs-master/internal/excel/reader.go)

**职责**: Excel 文件的读取、预览、写入

**核心数据结构**:

| 结构体 | 说明 |
|---|---|
| `RowData` | 一行运单数据（重量、省份、客户、运费等） |
| `ExcelPreview` | Excel预览信息（行数、列名、客户、省份、样本） |
| `CalcSummary` | 计算汇总（总件数、总运费、按省/客户/规则级别统计） |

**核心函数**:

#### 快速预览 - `ReadPreviewFast()`
**优化策略**: 直接读取ZIP原始XML，不走excelize全量解析
1. 用regex从sheet XML头部提取 `<dimension ref="A1:Z500000"/>` 获取总行数（O(1)）
2. 流式解析前N行（默认1000行）的原始cell数据
3. 收集需要的共享字符串索引（SST）
4. 只加载需要的SST条目（不全量加载数百万字符串）
5. 解析表头，自动识别列映射

**列名自动识别** (`detectColumns`):
支持多种列名变体，按优先级匹配：
- 日期: 业务时间 > 时间/日期/time
- 重量: 结算重量 > 计费重量 > 重量/weight
- 省份: 目的省 > 签收省 > 省份/province
- 客户: 精确"客户" > 含"客户" > customer/client

#### 全量读取 - `ReadAllRowsWithProgress()`
- 使用 excelize 流式行迭代器
- 计费重量取 `max(实际重量, 体积重)`，最小0.01kg
- 每1000行报告一次进度

#### 结果写入 - `WriteResult()`
- 使用 `StreamWriter` 流式写入，内存占用低
- 两个Sheet: "结算结果" + "汇总统计"
- 表头样式（蓝底白字加粗居中）
- 自动列宽

---

### 3.5 运费计算引擎 - freight/engine.go

**文件**: [engine.go](file:///Users/cxd/mmjs-master/mmjs-master/internal/freight/engine.go)

**职责**: 单笔运费计算的核心逻辑

**核心函数**: `doCalcSingle()`

**计费公式**:

```
如果 重量 <= 首重:
    运费 = 首重单价
否则:
    超出重量 = 重量 - 首重
    如果 模式 = hundred_gram (百克续):
        续重单元 = ceil(超出重量 × 10)  // 每100g一单元
    否则 (full_kg 全续):
        续重单元 = ceil(超出重量)       // 每kg一单元
    运费 = 首重单价 + 续重单元 × 续重单价

然后:
    运费 += 偏远附加费
    如果 最低收费 > 0 且 运费 < 最低收费: 运费 = 最低收费
    如果 最高收费 > 0 且 运费 > 最高收费: 运费 = 最高收费
    应用全局加价: 加价 = 固定加价 + 运费 × 百分比加价
    最终运费 = round((运费 + 加价) × 100) / 100
```

**特殊处理**:
- 零重量保护: 重量<=0 且配置了 `no_weight_price` 时，使用无重量价格
- 无匹配规则时: 使用全局保底规则，最后兜底5元
- 重量最小单位: 0.01kg

---

### 3.6 规则模块 - rules/

**文件**:
- [models.go](file:///Users/cxd/mmjs-master/mmjs-master/internal/rules/models.go) - 数据模型
- [repository.go](file:///Users/cxd/mmjs-master/mmjs-master/internal/rules/repository.go) - 数据访问
- [global.go](file:///Users/cxd/mmjs-master/mmjs-master/internal/rules/global.go) - 全局规则

#### 数据模型

**FreightRule 运费规则**:

| 字段 | 类型 | 说明 |
|---|---|---|
| `ID` | int64 | 主键 |
| `RuleType` | string | 规则类型: default/global/customer/campaign |
| `CustomerName` | string | 客户名（customer/campaign类型） |
| `Province` | string | 省份（空=所有省份） |
| `ContMode` | string | 续重模式: hundred_gram / full_kg |
| `FirstWeight` | float64 | 首重(kg) |
| `FirstPrice` | float64 | 首重单价(元) |
| `ContPrice` | float64 | 续重单价(元/100g 或 元/kg) |
| `MinFee` | float64 | 最低收费(保底价) |
| `MaxFee` | float64 | 最高收费(封顶) |
| `Surcharge` | float64 | 偏远附加费 |
| `CampaignName` | string | 活动名称 |
| `CampaignStart/End` | string | 活动起止日期 |
| `IsEnabled` | int | 是否启用 |
| `Remark` | string | 备注 |

**GlobalRule 全局规则**:

| 字段 | 说明 |
|---|---|
| `DefaultFirstWeight` | 默认首重 |
| `DefaultFirstPrice` | 默认首重单价 |
| `DefaultContPrice` | 默认续重单价 |
| `DefaultMinFee` | 默认最低收费 |
| `NoWeightPrice` | 无重量时价格 |
| `MarkupFixed` | 固定加价(元/件) |
| `MarkupPercent` | 百分比加价(%) |

#### 规则匹配优先级（4级覆盖）

```
┌──────────────────────────┐
│  活动规则 (campaign)      │  最高优先级
├──────────────────────────┤
│  客户规则 (customer)      │
├──────────────────────────┤
│  全局规则 (global)        │
├──────────────────────────┤
│  默认规则 (default)       │  兜底
└──────────────────────────┘
```

匹配逻辑 (`FindBestRule`):
1. 先过滤 `IsEnabled=1` 的规则
2. 按优先级从高到低遍历
3. 匹配条件: `RuleType` 匹配 + `CustomerName` 匹配 + `Province` 匹配
4. 省份匹配: 空字符串通配所有省份

#### RuleIndex: O(1) 规则索引

为批量计算预建索引，避免每行数据 O(R) 遍历规则。

**索引结构**:
```go
type RuleIndex struct {
    customerRules map[string]map[string]RuleResult  // customer -> province -> rule
    globalRules   map[string]RuleResult              // province -> rule
    defaultResult RuleResult                         // 兜底规则
}
```

**构建策略** (`BuildRuleIndex`):
1. 按优先级从低到高遍历（后者覆盖前者的方式相反）
2. 先放 global，再放 customer/campaign（后者不覆盖已存在的，保证高优先级胜出）
3. 省级别精确匹配 + 空省名通配

**查找策略** (`Find`):
1. 客户规则精确省匹配
2. 客户规则通配省匹配
3. 全局规则精确省匹配
4. 全局规则通配省匹配
5. 默认规则兜底

---

### 3.7 授权模块 - license/

**文件**:
- [crypto.go](file:///Users/cxd/mmjs-master/mmjs-master/internal/license/crypto.go) - 加密解密
- [machine.go](file:///Users/cxd/mmjs-master/mmjs-master/internal/license/machine.go) - 机器码
- [online.go](file:///Users/cxd/mmjs-master/mmjs-master/internal/license/online.go) - 在线授权
- [validator.go](file:///Users/cxd/mmjs-master/mmjs-master/internal/license/validator.go) - 授权验证

#### 机器码生成 - machine.go

**采集源** (Windows):
1. MAC地址 - 第一块物理网卡
2. CPU序列号 - WMIC Win32_Processor.ProcessorId
3. 硬盘序列号 - WMIC Win32_DiskDrive (PHYSICALDRIVE0)
4. 兜底: 主机名

**采集源** (非Windows):
1. MAC地址 + 主机名

**算法**:
```
combined = join(parts, "|")
hash = SHA256(combined)
machineCode = hex(hash)[:32]
格式化: XXXX-XXXX-XXXX-XXXX-XXXX-XXXX-XXXX-XXXX (8组，每组4字符)
```

#### 加密系统 - crypto.go

**加密算法栈**:
```
原始 JSON Payload
    │
    ▼
RSA-SHA256 签名 (私钥签名，公钥验证)
    │
    ▼
Payload + 签名 拼接
    │
    ▼
AES-256-GCM 加密 (随机Nonce)
    │
    ▼
Base64 编码
    │
    ▼
license.dat / license_data 字符串
```

**LicensePayload 结构**:
```go
type LicensePayload struct {
    Version      int      // 版本号
    MachineCode  string   // 绑定的机器码
    Customer     string   // 客户名称
    IssuedAt     string   // 签发时间
    ExpiresAt    string   // 过期时间
    DurationDays int      // 授权天数
    Features     []string // 功能列表
    Nonce        string   // 随机数
}
```

**密钥硬编码**:
- AES-256 密钥: 32字节硬编码在二进制中
- RSA公钥: PEM格式硬编码，用于验证签名
- RSA私钥: 仅在服务器端，客户端不持有

#### 在线授权 - online.go

**服务器配置**:
- 默认主服务器: `http://www.hbdxm.com/yunfei_api`
- 支持主备双服务器配置（主失效自动切换备）
- API密钥签名: `MD5(data + "|" + ts + "|" + api_secret)`

**核心功能**:
- `CheckOnlineLicense(machineCode)` - 在线验证授权（7天缓存）
- `ActivateOnline(licenseKey, machineCode)` - 旧版YF激活码激活
- `ActivateWithLicenseData(licenseData, machineCode)` - 新版加密数据激活
- `syncLicenseInfo(info)` - 同步服务器最新到期时间到本地

**缓存机制**:
- 在线验证结果缓存7天
- 缓存到 `app_settings` 表的 `online_license_cache` 键
- 缓存包含机器码校验，防止换机复用

#### 验证器 - validator.go

**验证流程** (`VerifyLicense`):
1. 从 `license_info` 表读取加密的授权数据
2. AES解密 + RSA签名验证
3. 校验机器码是否匹配
4. 校验过期时间
5. 返回 LicenseInfo（含剩余天数）

**导入流程** (`ImportLicense`):
1. Base64解码 + AES解密 + RSA验证
2. 机器码匹配校验
3. 过期时间校验
4. 存入 `license_info` 表

---

## 4. 前端模块详解 (Vue3)

### 4.1 技术栈

| 库 | 版本 | 用途 |
|---|---|---|
| Vue | ^3.5.13 | 前端框架 |
| Vue Router | ^4.5.0 | 路由管理 |
| Pinia | ^2.3.0 | 状态管理 |
| Element Plus | ^2.9.1 | UI组件库 |
| ECharts | ^5.5.1 | 图表 |
| vue-echarts | ^7.0.3 | Vue图表封装 |
| dayjs | ^1.11.13 | 日期处理 |
| Vite | ^6.0.5 | 构建工具 |
| TypeScript | ~5.6.3 | 类型系统 |

### 4.2 路由与页面

**文件**: [router/index.ts](file:///Users/cxd/mmjs-master/mmjs-master/frontend/src/router/index.ts)

**路由配置**:

| 路径 | 页面 | 说明 | 权限 |
|---|---|---|---|
| `/login` | Login.vue | 登录页 | 无需认证 |
| `/home` | Home.vue | 首页仪表盘 | 需登录 |
| `/calc` | Calc.vue | 计费结算 | 需登录+授权 |
| `/rules` | Rules.vue | 规则管理 | 需登录 |
| `/history` | History.vue | 历史记录 | 需登录 |
| `/license` | License.vue | 授权管理 | 需登录 |
| `/settings` | Settings.vue | 系统设置 | 需登录 |

**路由守卫逻辑**:
1. 白名单页面（登录页）直接放行
2. 检查本地 token 有效性
3. 首次登录后检查授权，未授权跳转授权页
4. 计费等核心功能页需要有效授权

### 4.3 状态管理 - Pinia

**文件**: [stores/app.ts](file:///Users/cxd/mmjs-master/mmjs-master/frontend/src/stores/app.ts)

**Store**: `useAppStore`

**State**:
- `license: LicenseInfo | null` - 授权信息
- `rules: FreightRule[]` - 规则列表
- `machineCode: string` - 机器码
- `calculating: boolean` - 是否正在计算

**Getters**:
- `isLicensed` - 是否已授权
- `daysLeft` - 剩余天数
- `licenseStatus` - 授权状态 (unknown/expired/expiring/active)

**Actions**（分组）:

| 分组 | 方法 |
|---|---|
| 授权 | fetchLicense, fetchMachineCode, importLicense, checkOnlineLicense, activateOnline |
| 规则 | fetchRules, fetchRulesByCustomer, saveRule, deleteRule, deleteRulesBatch |
| 客户 | fetchCustomers, deleteCustomer, copyCustomerRules, importCustomerRules, downloadTemplate |
| 全局规则 | fetchGlobalRules, saveGlobalRules |

### 4.4 API 封装

**文件**: [api/index.ts](file:///Users/cxd/mmjs-master/mmjs-master/frontend/src/api/index.ts)

**封装的请求方法**:
- `apiGet(path)` - GET 请求（自动带Bearer Token）
- `apiPost(path, body)` - POST 请求（JSON Body）
- `apiUpload(file, onProgress)` - 文件上传（XHR，带进度回调）
- `apiExport(onProgress)` - 导出下载（Blob响应）

### 4.5 核心页面 - 计费结算 Calc.vue

**文件**: [Calc.vue](file:///Users/cxd/mmjs-master/mmjs-master/frontend/src/views/Calc.vue)

**四步流程**:

| 步骤 | 名称 | 功能 |
|---|---|---|
| Step 0 | 上传文件 | 拖拽/点击选择，最多5个文件，上传到服务端临时目录 |
| Step 1 | 预览确认 | 显示每个文件的行数、客户数、省份数、前5行采样数据 |
| Step 2 | 并行计算 | 每个文件独立进度条，实时更新，支持后台运行 |
| Step 3 | 导出结果 | 汇总统计卡片、按省/客户明细、单文件/一键导出 |

**关键特性**:
- 批次状态持久化到 localStorage，刷新页面可恢复
- 计算在后端 goroutine 执行，离开页面不中断
- 进度轮询间隔 400ms
- 批次30分钟过期自动清理

---

## 5. PHP 服务端授权系统

### 5.1 概述

部署在 Web 服务器上的 PHP 脚本，负责授权签发、验证和管理。

### 5.2 文件清单

**文件**:
- [init.php](file:///Users/cxd/mmjs-master/mmjs-master/server_php/init.php) - 数据库初始化
- `activate.php` - 激活接口
- `verify.php` - 验证接口
- `admin.php` - 管理后台

### 5.3 数据库表 (MySQL)

**yunfei_licenses 授权表**:

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | INT | 主键 |
| `license_key` | VARCHAR(32) | 旧版激活码（兼容） |
| `machine_code` | VARCHAR(64) | 绑定的机器码 |
| `customer_name` | VARCHAR(100) | 客户名称 |
| `contact_info` | VARCHAR(200) | 联系方式 |
| `expires_at` | DATE | 到期日期 |
| `license_data` | TEXT | 加密授权数据(RSA+AES) |
| `issued_at` | DATETIME | 签发时间 |
| `duration_days` | INT | 授权天数 |
| `remark` | VARCHAR(500) | 备注 |
| `activated_at` | DATETIME | 激活时间 |
| `last_check_at` | DATETIME | 最后验证时间 |
| `status` | TINYINT | 1=未激活 2=已激活 3=已停用 |

**yunfei_settings 设置表**:
- key-value 结构
- `api_secret` - API签名密钥

### 5.4 接口签名机制

客户端请求签名算法:
```
sign = MD5(data + "|" + timestamp + "|" + api_secret)
```

请求体携带:
```json
{
  "machine_code": "XXX",
  "sign": "md5hash",
  "ts": 1234567890
}
```

### 5.5 部署

1. 将 `server_php/` 下的文件上传到 PHP 服务器
2. 修改 `init.php` 中的数据库配置
3. 访问一次 `init.php` 初始化数据库
4. 客户端配置授权服务器地址指向该目录

---

## 6. 数据库设计

### 6.1 ER 图

```
┌──────────────────┐
│  freight_rules   │  运费规则 (N条)
├──────────────────┤
│ id (PK)          │
│ rule_type        │  default/global/customer/campaign
│ customer_name    │
│ province         │
│ cont_mode        │  hundred_gram/full_kg
│ first_weight     │
│ first_price      │
│ cont_price       │
│ min_fee/max_fee  │
│ surcharge        │
│ ...              │
└──────────────────┘

┌──────────────────┐
│  calc_history    │  计算历史 (N条)
├──────────────────┤
│ id (PK)          │
│ input_file       │
│ output_file      │
│ total_count      │
│ total_fee        │
│ avg_fee/max/min  │
│ rule_summary     │
│ calc_duration    │
│ created_at       │
└──────────────────┘

┌──────────────────┐
│  license_info    │  授权信息 (1条)
├──────────────────┤
│ id=1 (PK)        │
│ machine_code     │
│ customer_name    │
│ expires_at       │
│ license_raw      │
│ last_verify_at   │
└──────────────────┘

┌──────────────────┐
│  app_settings    │  设置 (key-value)
├──────────────────┤
│ key (PK)         │
│ value            │
└──────────────────┘

┌──────────────────┐
│  global_rules    │  全局规则 (1条)
├──────────────────┤
│ id=1 (PK)        │
│ default_first_*  │
│ no_weight_price  │
│ markup_fixed/%   │
└──────────────────┘
```

### 6.2 索引

| 表 | 索引 | 用途 |
|---|---|---|
| freight_rules | (rule_type, customer_name) | 按客户查询规则 |
| freight_rules | (province) | 按省份筛选 |
| yunfei_licenses | idx_license_key | 激活码查询 |
| yunfei_licenses | idx_machine | 机器码查询 |

---

## 7. 核心算法与流程

### 7.1 批量计算完整流程

```
用户上传Excel文件
    │
    ▼
后端保存到临时目录
    │
    ▼
快速预览 (ReadPreviewFast)
  ├─ ZIP直读dimension → 总行数 O(1)
  ├─ 流式读前1000行 → 采样数据
  └─ 检测列名映射 → 自动识别重量/省份/客户
    │
    ▼
用户确认，开始计算
    │
    ▼
1. 全量读取Excel (ReadAllRows)
    ├─ 流式逐行读取
    ├─ 构建RowData数组
    └─ 计费重量 = max(实重, 体积重)
    │
    ▼
2. 构建规则索引 (BuildRuleIndex)
    ├─ 加载所有启用规则
    └─ 构建 customerRules / globalRules 哈希表
    │
    ▼
3. 多核并行计算
    ├─ 按CPU核数分块
    ├─ 每个goroutine处理一块
    └─ 每行调用 CalcSingleWithIndex (O(1)查规则)
    │
    ▼
4. 构建汇总 (BuildSummary)
    ├─ 总件数/总运费/平均/最高/最低
    ├─ 按省份汇总
    ├─ 按客户汇总
    └─ 按规则级别汇总
    │
    ▼
5. 保存历史记录
    │
    ▼
用户导出结果
    │
    ▼
流式写入Excel (WriteResult)
  ├─ 结算结果 Sheet
  └─ 汇总统计 Sheet
```

### 7.2 规则匹配算法

```
输入: customer, province, 规则列表
输出: 最佳匹配的 RuleResult

1. 过滤: 只保留 is_enabled=1 的规则

2. 按优先级从高到低尝试:
   a. 活动规则 (rule_type='campaign')
      - 条件: customer_name == customer AND (province == target OR province == '')
   b. 客户规则 (rule_type='customer')
      - 条件: 同上
   c. 全局规则 (rule_type='global')
      - 条件: (province == target OR province == '')
   d. 默认规则 (rule_type='default')
      - 兜底，必返回

3. 返回第一个匹配的规则
```

**RuleIndex 优化版**（批量计算用，O(1)查找）:
```
构建时:
  按优先级从低到高遍历规则，插入哈希表
  （高优先级后插入，不覆盖已存在的 → 高优先级胜出）

查找时:
  1. customerMap[customer][province] - 客户精确省
  2. customerMap[customer][''] - 客户通配省
  3. globalMap[province] - 全局精确省
  4. globalMap[''] - 全局通配省
  5. defaultResult - 兜底
```

---

## 8. API 接口清单

### 8.1 认证接口

| 方法 | 路径 | 说明 | 认证 |
|---|---|---|---|
| POST | `/api/auth/login` | 登录获取token | 否 |
| GET | `/api/auth/verify` | 验证token有效性 | 否 |

### 8.2 授权接口

| 方法 | 路径 | 说明 | 认证 |
|---|---|---|---|
| GET | `/api/machine-code` | 获取机器码 | 是 |
| POST | `/api/license/import` | 导入离线授权 | 是 |
| GET | `/api/license/info` | 获取授权信息 | 是 |
| GET | `/api/license/check-online` | 在线检查授权 | 否 |
| POST | `/api/license/activate-online` | 激活码激活 | 否 |
| POST | `/api/license/activate-license-data` | 加密数据激活 | 否 |
| GET | `/api/license/server-info` | 获取服务器地址 | 是 |
| POST | `/api/license/set-server` | 设置主服务器 | 是 |
| POST | `/api/license/set-backup-server` | 设置备服务器 | 是 |
| GET/POST | `/api/license/api-secret` | 获取/设置API密钥 | 是 |
| POST | `/api/license/reset` | 重置授权 | 是 |

### 8.3 规则接口

| 方法 | 路径 | 说明 | 认证 |
|---|---|---|---|
| GET | `/api/rules` | 获取所有规则（?customer=过滤） | 是 |
| POST | `/api/rules/save` | 保存规则 | 是 |
| POST | `/api/rules/delete` | 删除规则 | 是 |
| POST | `/api/rules/delete-batch` | 批量删除 | 是 |
| POST | `/api/rules/seed` | 初始化默认规则 | 是 |
| GET/POST | `/api/global-rules` | 获取/保存全局规则 | 是 |

### 8.4 客户接口

| 方法 | 路径 | 说明 | 认证 |
|---|---|---|---|
| GET | `/api/customers` | 获取客户列表 | 是 |
| POST | `/api/customers/delete` | 删除客户 | 是 |
| POST | `/api/customers/copy-rules` | 复制客户规则 | 是 |
| POST | `/api/customers/import` | 批量导入客户规则 | 是 |
| GET | `/api/customers/template` | 下载导入模板 | 是 |

### 8.5 Excel 接口

| 方法 | 路径 | 说明 | 认证 |
|---|---|---|---|
| POST | `/api/excel/upload` | 上传Excel文件（支持多文件） | 是 |
| POST | `/api/excel/preview` | 预览单个文件 | 是 |
| POST | `/api/excel/preview-multi` | 预览多个文件 | 是 |

### 8.6 计算接口

| 方法 | 路径 | 说明 | 认证 |
|---|---|---|---|
| POST | `/api/calculate` | 单文件计算（旧版） | 是 |
| POST | `/api/calculate/batch` | 批量计算（多文件并行） | 是 |
| GET | `/api/calculate/progress` | 单文件进度查询 | 是 |
| GET | `/api/calculate/batch-progress` | 批量进度查询 | 是 |
| GET | `/api/calculate/result` | 单文件结果查询 | 是 |
| GET | `/api/calculate/batch-result` | 批量结果查询 | 是 |

### 8.7 导出与历史

| 方法 | 路径 | 说明 | 认证 |
|---|---|---|---|
| GET | `/api/export` | 导出结果（?task_id=指定任务） | 是 |
| GET | `/api/history` | 历史记录列表 | 是 |
| DELETE | `/api/history` | 删除单条历史 | 是 |
| POST | `/api/history/clear` | 清空历史 | 是 |
| GET | `/api/history/detail` | 历史详情 | 是 |

### 8.8 设置

| 方法 | 路径 | 说明 | 认证 |
|---|---|---|---|
| GET/POST | `/api/settings` | 获取/保存设置 | 是 |

---

## 9. 依赖关系

### 9.1 Go 依赖

**核心依赖**:

| 包 | 版本 | 用途 |
|---|---|---|
| `github.com/xuri/excelize/v2` | v2.8.1 | Excel 文件读写 |
| `modernc.org/sqlite` | v1.28.0 | 纯Go SQLite驱动（无CGO） |
| `github.com/mattn/go-sqlite3` | v1.14.16 | SQLite驱动（备用） |

**间接依赖**:
- `golang.org/x/crypto` - 加密算法
- `golang.org/x/sys` - 系统调用
- `github.com/google/uuid` - UUID生成
- 其他详见 [go.mod](file:///Users/cxd/mmjs-master/mmjs-master/go.mod)

### 9.2 前端依赖

**运行时**:
- vue, vue-router, pinia - Vue生态
- element-plus, @element-plus/icons-vue - UI组件
- echarts, vue-echarts - 图表
- dayjs - 日期处理

**开发时**:
- vite - 构建工具
- typescript, vue-tsc - TypeScript
- @vitejs/plugin-vue - Vue Vite插件

### 9.3 模块依赖图

```
main.go (入口 + HTTP)
    │
    ▼
internal/app/ (门面层)
    │
    ├────► internal/rules/    ─────► internal/db/
    ├────► internal/freight/  ─────► internal/rules/
    ├────► internal/excel/
    └────► internal/license/  ─────► internal/db/
                                    │
                                    ▼
                              外部 PHP 授权服务器
```

**依赖方向**: 上层依赖下层，同层模块间低耦合
- app 层依赖所有子模块
- freight 依赖 rules（调用规则查找）
- rules 依赖 db（数据持久化）
- license 依赖 db（本地缓存）+ 外部HTTP（在线验证）
- excel 无内部依赖（纯数据处理）

---

## 10. 项目运行方式

### 10.1 环境要求

- **Go**: >= 1.22.0
- **Node.js**: >= 16.x（推荐 18+）
- **Wails**: v2.x（桌面打包用，开发模式可选）
- **PHP**: >= 7.4（服务端用）
- **MySQL**: >= 5.7（服务端用）

### 10.2 开发模式运行

#### 方式一：纯Go HTTP + 前端开发服务器（推荐）

```bash
# 1. 启动前端开发服务器 (端口 5173)
cd frontend
npm install
npm run dev

# 2. 新开终端，启动 Go 后端 (端口 58080)
cd ..
go run main.go

# 3. 访问 http://localhost:5173
#    （Vite dev server 自动代理 API 到 58080，需自行配置 vite.config.ts）
```

#### 方式二：Go 嵌入前端（生产模式预览）

```bash
# 1. 构建前端
cd frontend
npm run build

# 2. 启动 Go 后端 (前端 dist 被 embed 到二进制中)
cd ..
go run main.go

# 3. 自动打开浏览器 http://localhost:58080
```

#### 方式三：Wails 桌面模式

```bash
# 安装 Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 开发模式
wails dev

# 构建生产版本
wails build
```

### 10.3 默认账号

- **用户名**: `admin`
- **密码**: `admin123`
- 可在设置页面修改，保存在 `settings.json` 中

### 10.4 服务端部署

```bash
# 1. 上传 server_php/ 到 PHP 服务器目录
# 2. 修改 init.php 中的数据库配置
# 3. 浏览器访问一次: https://your-domain.com/yunfei_api/init.php
# 4. 客户端设置授权服务器地址指向该目录
```

### 10.5 数据存储位置

- **Windows**: `%APPDATA%\yunfei\`
  - `yunfei.db` - SQLite数据库
  - `settings.json` - 设置文件
- **macOS**: `$HOME/yunfei/`
- **Linux**: `$HOME/yunfei/`

### 10.6 构建脚本

- `build.bat` - Windows 批处理构建脚本
- `build.ps1` - PowerShell 构建脚本
- `killport.bat` - 释放端口脚本

### 10.7 端口配置

- 默认端口: `58080`
- 可通过环境变量 `PORT` 修改
- 单实例检测: 启动时检测端口，已有实例则自动打开浏览器并退出

---

## 附录

### 常用操作速查

| 操作 | 位置 |
|---|---|
| 修改默认账号密码 | 设置页面 / settings.json |
| 修改授权服务器地址 | 授权页面 / API设置 |
| 导入离线授权 | 授权页面 → 导入License |
| 在线激活 | 授权页面 → 输入激活码 |
| 导入客户规则 | 规则页面 → 批量导入 |
| 复制客户规则 | 规则页面 → 客户管理 → 复制 |
| 查看计算历史 | 历史记录页面 |
| 重新导出结果 | 历史记录 → 详情 → 导出 |

### 故障排查

| 问题 | 可能原因 | 解决方案 |
|---|---|---|
| 无法启动，端口被占用 | 已有实例在运行 | 等几秒自动打开，或运行 killport.bat |
| Excel 打开失败 | 文件格式不对 / 文件损坏 | 确认是 .xlsx 格式，用 Excel 另存为一次 |
| 运费计算为0 | 没有匹配的规则 | 检查规则配置，确保客户名/省份匹配 |
| 授权失败 | 机器码不匹配 / 已过期 | 联系客服，提供新机器码 |
| 在线验证失败 | 网络不通 / 服务器故障 | 检查网络，或切换备用服务器 |

---

*文档生成时间: 2026-07-02*  
*基于代码版本: mmjs-master*
