# 云费 · 快递运费结算软件 — 设计方案

> 版本: 1.0 Draft  
> 目录: `E:\sjfx\yunfei`  
> 定位: 本地独立桌面软件 + 服务器授权验证

---

## 一、计费规则详解

### 1.1 百克续（per 100g continuation）

```
首重费 = 首重单价（1kg内）
续重单元 = ceil[(实际重量 - 1.0) × 10]   # 每100g一单元，不足100g按100g
续重费 = 续重单元 × 续重单价（元/100g）
总运费 = 首重费 + 续重费

示例：
  1.35kg 发某省，首重3元/kg，续重0.3元/100g
  → 首重 3元 + ceil(0.35×10)=4个×0.3 = 3 + 1.2 = 4.20元

  0.45kg 发同省，首重3元/kg，续重0.3元/100g
  → 首重 3元（不足1kg也按首重） = 3.00元
```

**适用场景**: 客户议价能力强，要求精确到100g续重（避免按整kg多收费）

---

### 1.2 全续（full-kg continuation）

```
首重费 = 首重单价（1kg内）
续重单元 = ceil(实际重量 - 1.0)          # 整kg，不足1kg按1kg
续重费 = 续重单元 × 续重单价（元/kg）
总运费 = 首重费 + 续重费

示例：
  1.35kg 发某省，首重3元/kg，续重1.5元/kg
  → 首重 3元 + ceil(0.35)=1×1.5 = 3 + 1.5 = 4.50元

  对比百克续: 4.20元 vs 4.50元 → 百克续便宜0.30元
```

**本质区别**: 同样重量，百克续比全续约便宜 **10%~30%**

---

### 1.3 规则优先级（4级覆盖）

```
┌──────────────────────────┐
│  活动规则     (最高优先级) │  例: 双11期间蜜丝婷发全国一律 2.5元/件
├──────────────────────────┤
│  客户规则                │  例: 蜜丝婷 省内2.8+0.8, 偏远6.0+3.5
├──────────────────────────┤
│  全局规则                │  例: 默认全省份统一 3.5+1.2 (全续)
├──────────────────────────┤
│  默认规则     (兜底)      │  例: 找不到任何规则时 5.0+2.0 (全续)
└──────────────────────────┘
```

每条客户规则额外标记:
- `mode`: `"hundred_gram"` (百克续) | `"full_kg"` (全续)
- 同一客户可按省份混用模式

---

## 二、架构方案对比

共 4 套方案，从 5 个维度横向对比:

| 维度 | 方案A: Electron | 方案B: Wails+Go | 方案C: Tauri | 方案D: Python+Qt |
|---|---|---|---|---|
| **打包体积** | ~150MB | ~25MB | ~8MB 🥇 | ~120MB |
| **内存占用(空闲)** | ~120MB | ~50MB | ~30MB 🥇 | ~90MB |
| **Excel 96万行读写** | 2~5分钟 | 30秒~2分钟 🥇 | 1~3分钟 | 40秒~2分钟 |
| **开发周期** | 2~3周 🥇 | 3~4周 | 4~6周 | 2~3周 |
| **UI美观度** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |
| **Win7兼容** | ✅ | ✅ (需WebView2) | ✅ (需WebView2) | ✅ 原生 |
| **反破解难度** | ⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ 🥇 | ⭐⭐ |
| **打包为单exe** | NSIS安装包 | ✅ 单exe | ✅ 单exe | ✅ 单exe |
| **现有技术栈匹配** | ⭐⭐⭐ (同Vue3) | ⭐⭐⭐⭐ (同Go外挂) | ⭐⭐ | ⭐ |
| **生态成熟度** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ |

---

### 方案A: Electron + Vue3 + SQLite

```
┌──────────────────────────────────────┐
│  Electron Main Process (Node.js)     │
│  ├─ 机器码获取 (systeminformation)    │
│  ├─ License验证 + AES解密            │
│  ├─ better-sqlite3 本地数据库         │
│  └─ exceljs 流式Excel读写            │
├──────────────────────────────────────┤
│  Electron Renderer (Chromium)        │
│  ├─ Vue3 + TypeScript + Pinia        │
│  ├─ Element Plus + ECharts           │
│  └─ 路由: 首页/计费/规则/授权/设置    │
└──────────────────────────────────────┘
```

| 项目 | 说明 |
|---|---|
| **优点** | 技术栈完全匹配现有项目；UI可以做到和Web版一模一样；npm生态极丰富；electron-builder打包成熟 |
| **缺点** | 体积最大（内嵌Chromium）；内存占用高；Node.js容易被反编译；better-sqlite3需要native rebuild |
| **推荐度** | ⭐⭐⭐⭐ (最快落地) |

---

### 方案B: Wails + Go + Vue3 ⭐推荐

```
┌──────────────────────────────────────┐
│  Go Backend (编译为原生)              │
│  ├─ 机器码获取 (gopsutil + WMI)       │
│  ├─ License验证 + AES+RSA签名        │
│  ├─ SQLite (modernc.org/sqlite 纯Go) │
│  ├─ excelize 流式Excel读写            │
│  └─ goroutine 并发计算池              │
├──────────────────────────────────────┤
│  Frontend (系统WebView2)              │
│  ├─ Vue3 + TypeScript + Pinia        │
│  ├─ Element Plus + ECharts           │
│  └─ 路由: 首页/计费/规则/授权/设置    │
└──────────────────────────────────────┘
```

| 项目 | 说明 |
|---|---|
| **优点** | Go编译为原生二进制 → 反编译难度高；excelize处理Excel极快；goroutine天然并发；体积小；和你们「Go微服务外挂」架构一脉相承 |
| **缺点** | 需要Go开发环境；CGO交叉编译Windows需注意；WebView2在Win7需手动安装 |
| **推荐度** | ⭐⭐⭐⭐⭐ (性能和安全的平衡最优) |

**关键指标预估**:

| 场景 | Electron | Wails+Go |
|---|---|---|
| 读96万行xlsx | ~120s | ~30s |
| 计算运费 | ~3s | ~1s |
| 写结果xlsx | ~100s | ~40s |
| **总耗时** | ~4分钟 | **~1分钟** 🚀 |

---

### 方案C: Tauri + Vue3

```
┌──────────────────────────────────────┐
│  Rust Backend                        │
│  ├─ 机器码获取 (sysinfo)              │
│  ├─ License验证 + RSA签名            │
│  ├─ SQLite (rusqlite)                │
│  └─ calamine 读 + xlsxwriter 写      │
├──────────────────────────────────────┤
│  Frontend (系统WebView2)              │
│  └─ 同方案B                          │
└──────────────────────────────────────┘
```

| 项目 | 说明 |
|---|---|
| **优点** | 体积最小~8MB；内存占用最低；Rust二进制最难反编译；性能最优 |
| **缺点** | Rust开发周期长；生态不够成熟；xlsx写入不如excelize方便；出bug调试困难 |
| **推荐度** | ⭐⭐⭐ (极致性能，但成本高) |

---

### 方案D: Python + PySide6 + Nuitka

```
┌──────────────────────────────────────┐
│  Python Backend                      │
│  ├─ 机器码获取 (pywin32/subprocess)   │
│  ├─ License验证 + cryptography       │
│  ├─ SQLite (内置sqlite3)             │
│  ├─ polars + xlsxwriter Excel处理    │
│  └─ PySide6 原生UI                   │
└──────────────────────────────────────┘
```

| 项目 | 说明 |
|---|---|
| **优点** | 开发最快（2周）；polars处理Excel极强；Nuitka能编译为exe |
| **缺点** | UI不如Web技术好看；Nuitka打包慢且容易出问题；Python可反编译；PySide6学习成本 |
| **推荐度** | ⭐⭐ (原型验证用，正式产品不推荐) |

---

### 🏆 最终推荐: 方案B Wails+Go

原因: **性能接近原生(1分钟处理百万行) + 反编译难度适中 + 体积小 + 和你们Go外挂架构一致**

---

## 三、机器码 + 授权验证系统

### 3.1 机器码生成

```
采集源（按顺序，取至少3个成功）:
1. MAC地址       → 取第一块物理网卡
2. CPU序列号     → WMI: Win32_Processor.ProcessorId
3. 硬盘序列号    → WMI: Win32_DiskDrive.SerialNumber (C盘)
4. 主板序列号    → WMI: Win32_BaseBoard.SerialNumber

组合: MAC + CPU + DISK → SHA256 → 取前32位 → 机器码

展示格式: XXXX-XXXX-XXXX-XXXX-XXXX-XXXX-XXXX-XXXX
          (8组，便于用户抄写给客服)
```

### 3.2 授权流程

```
┌─────────────┐                    ┌─────────────┐
│  客户端软件  │                    │  授权服务器   │
└──────┬──────┘                    └──────┬──────┘
       │                                  │
       │ 1. 首次启动获取机器码              │
       │    "ABCD-1234-..."               │
       │                                  │
       │ 2. 用户复制机器码发给客服          │
       │  ──────────────────────────────► │
       │                                  │ 3. 客服后台生成License
       │                                  │    - 输入: 机器码+时长+客户
       │                                  │    - 输出: license.dat
       │ 4. 客服发给用户License文件        │
       │ ◄──────────────────────────────  │
       │                                  │
       │ 5. 客户端导入License文件          │
       │    验证签名 → 验证机器码 → 存储   │
       │                                  │
       │ 6. 每次启动:                     │
       │    读取license → 验证签名        │
       │    → 验证机器码 → 检查过期        │
       │    → 离线可用(在有效期内)         │
       │                                  │
       │ 7. 可选: 每次启动联网验证         │
       │    → 服务器可远程吊销授权         │
```

### 3.3 License文件结构

```json
{
  "version": 1,
  "machine_code": "ABCD1234...",
  "customer_name": "XX快递公司",
  "issued_at": "2026-06-22T10:00:00+08:00",
  "expires_at": "2026-12-22T10:00:00+08:00",
  "duration_days": 183,
  "features": ["freight_calc", "report_export"],
  "max_rows_per_file": 5000000,
  "nonce": "random_uuid_v4",
  "signature": "RSA_SIGN_OF_ABOVE_FIELDS"
}
```

**加密方式**: 
1. JSON → msgpack 二进制编码（体积更小）
2. RSA私钥签名 → 附加到末尾
3. AES-256-GCM加密整个二进制块
4. Base64编码 → 写入 `license.dat`

**客户端验证**:
1. 读取 `license.dat` → Base64解码
2. AES解密（密钥编译在Go二进制里，不易提取）
3. msgpack解码
4. RSA公钥验证签名
5. 核对 `machine_code` 一致
6. 检查 `expires_at > now`

### 3.4 防破解措施

| 层级 | 措施 | 说明 |
|---|---|---|
| **编译层** | Go编译为原生二进制 | 难以直接读源码 |
| **构建层** | `garble` 混淆编译 | 符号名/字符串混淆 |
| **密钥层** | AES Key + RSA公钥硬编码 | 分散存储在多个变量中，运行时拼接 |
| **校验层** | 多点校验 | 打开文件/导出结果/设置页面都暗中校验 |
| **时间层** | 本地时间 + NTP双校验 | 防止改系统时间绕过 |
| **数据层** | license.dat绑定机器码 | 换机器授权自动失效 |

---

## 四、数据库设计（本地SQLite）

### 4.1 规则表 `freight_rules`

```sql
CREATE TABLE freight_rules (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    rule_type     TEXT NOT NULL,     -- 'default'|'global'|'customer'|'campaign'
    customer_name TEXT,              -- rule_type='customer'时必填
    campaign_name TEXT,              -- rule_type='campaign'时必填
    campaign_start TEXT,             -- 活动开始日期
    campaign_end   TEXT,             -- 活动结束日期
    province      TEXT,              -- 省份，NULL=所有省份
    
    -- 续重模式
    cont_mode     TEXT NOT NULL DEFAULT 'full_kg',  
    -- 'hundred_gram' (百克续) | 'full_kg' (全续)
    
    first_weight  REAL NOT NULL DEFAULT 1.0,  -- 首重(kg)
    first_price   REAL NOT NULL,              -- 首重单价(元)
    cont_price    REAL NOT NULL,              -- 续重单价(元)
    -- 百克续时 cont_price = 元/100g
    -- 全续时   cont_price = 元/kg
    
    min_fee       REAL DEFAULT 0,             -- 最低收费
    max_fee       REAL,                       -- 最高收费(封顶)
    surcharge     REAL DEFAULT 0,             -- 偏远附加费
    
    remark        TEXT,
    is_enabled    INTEGER DEFAULT 1,
    created_at    TEXT DEFAULT (datetime('now','localtime')),
    updated_at    TEXT DEFAULT (datetime('now','localtime'))
);

CREATE INDEX idx_rules_customer ON freight_rules(rule_type, customer_name);
CREATE INDEX idx_rules_province ON freight_rules(province);
```

### 4.2 历史记录表 `calc_history`

```sql
CREATE TABLE calc_history (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    input_file      TEXT NOT NULL,
    output_file     TEXT,
    total_count     INTEGER,          -- 总件数
    total_fee       REAL,             -- 总运费
    avg_fee         REAL,             -- 均价
    rule_summary    TEXT,             -- 使用的规则摘要 JSON
    calc_duration   REAL,             -- 计算耗时(秒)
    created_at      TEXT DEFAULT (datetime('now','localtime'))
);
```

### 4.3 授权缓存表 `license_info`

```sql
CREATE TABLE license_info (
    id              INTEGER PRIMARY KEY CHECK (id = 1),
    machine_code    TEXT NOT NULL,
    customer_name   TEXT,
    expires_at      TEXT NOT NULL,
    issued_at       TEXT,
    features        TEXT,              -- JSON array
    license_raw     TEXT,              -- base64的license文件内容
    last_verify_at  TEXT,
    created_at      TEXT DEFAULT (datetime('now','localtime'))
);
```

---

## 五、服务器端授权管理

### 5.1 接口设计（最简方案：PHP单文件）

```
POST /api/license/issue
  入参: { machine_code, customer_name, duration_days, admin_key }
  出参: { license_b64, expires_at }
  逻辑: 验证admin_key → 生成license → RSA签名 → AES加密 → 返回Base64

GET  /api/license/list?admin_key=xxx
  出参: [{ machine_code, customer_name, expires_at, issued_at, revoked }]

POST /api/license/revoke
  入参: { machine_code, admin_key }
  逻辑: 标记吊销（下次客户端联网验证时生效）

GET  /api/license/verify?machine_code=xxx
  出参: { valid: true/false, expires_at, revoked }
  逻辑: 查询是否有效（客户端启动时可选调用）
```

### 5.2 管理后台（Web版）

```
功能:
├─ 授权管理
│   ├─ 生成新授权
│   ├─ 授权列表（搜索/筛选/导出）
│   ├─ 即将到期提醒（7天内标红）
│   └─ 吊销授权（远程失效）
├─ 客户管理
│   └─ 客户名/联系方式/备注
└─ 统计面板
    ├─ 总授权数 / 活跃数 / 到期数
    └─ 即将到期列表
```

---

## 六、客户端界面规划

```
┌──────────────────────────────────────────┐
│  云费 v1.0           [授权到期: 30天] 🔔  │
├──────────┬───────────────────────────────┤
│          │                               │
│  首页    │  📊 面板                       │
│  计费    │  ├─ 使用中规则: 客户规则(蜜丝婷)│
│  规则    │  ├─ 今日计算: 0件              │
│  历史    │  ├─ 累计计算: 960,150件        │
│  授权    │  └─ 累计运费: ¥3,809,964       │
│  设置    │                               │
│          │  🚀 快速操作                   │
│          │  [选择文件] [开始计算]         │
│          │                               │
└──────────┴───────────────────────────────┘
```

**核心页面**:
1. **计费**: 选择Excel → 预览 → 选择规则 → 计算 → 查看结果 → 导出
2. **规则管理**: 表格增删改查 + 省份选择器 + 百克续/全续切换
3. **授权**: 显示机器码 → 导入license → 显示到期时间
4. **设置**: 输出目录 / 默认规则 / 自动备份

---

## 七、开发计划

| 阶段 | 内容 | 工期 |
|---|---|---|
| **P0: 核心引擎** | Go后端: Excel读写 + 规则引擎 + SQLite | 1周 |
| **P1: 前端框架** | Vue3项目搭建 + 路由 + Element Plus布局 | 3天 |
| **P2: 计费页面** | 文件选择→计算→结果展示→导出 | 3天 |
| **P3: 规则管理** | 4级规则CRUD + 省份选择器 + 百克续/全续 | 3天 |
| **P4: 授权系统** | 机器码 + license验证 + 服务器API + 管理后台 | 3天 |
| **P5: 打包测试** | Wails打包 + Win7/10/11兼容测试 + 反破解加固 | 2天 |
| **合计** | | ~3周 |

---

## 八、费用估算

| 项目 | 方式 | 预估 |
|---|---|---|
| 客户端开发 | 自己开发/我帮你写 | 0元 |
| 服务器授权API | PHP单文件部署在现有服务器 | 0元 |
| SQLite | 本地文件 | 0元 |
| Wails | MIT开源 | 0元 |
| 代码签名证书 | 避免杀毒误报（可选） | ~2000元/年 |
| **总计** | | **约0~2000元** |

---

## 九、选择建议

```
┌─────────────────────────────────────────────────────┐
│                                                     │
│  ★ 如果你追求最快落地: 方案A Electron              │
│    2周出活，技术栈无缝衔接                          │
│                                                     │
│  ★★ 如果你追求最佳平衡: 方案B Wails+Go  ⭐推荐     │
│    3周出活，性能/体积/安全/美观 五项全优            │
│                                                     │
│  ★★★ 如果你追求极致安全和体积: 方案C Tauri        │
│    5周+，但反编译难度最高                          │
│                                                     │
│  不建议: 方案D Python+Qt (UI不好看，打包不稳定)     │
│                                                     │
└─────────────────────────────────────────────────────┘
```

---

下一步: 选定方案后，我可以立即开始搭建项目框架。
