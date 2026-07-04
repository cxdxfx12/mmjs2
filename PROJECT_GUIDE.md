# 喵喵云结算 - 项目开发指南

> 本文档沉淀了项目开发过程中的所有关键知识、工程约定、踩坑记录，供后期开发维护参考。
> 配套文档：[DESIGN.md](file:///Users/cxd/mmjs-master/mmjs-master/DESIGN.md)（早期设计方案）、[CODE_WIKI.md](file:///Users/cxd/mmjs-master/mmjs-master/CODE_WIKI.md)（代码结构详解）

---

## 目录

1. [项目现状与架构](#1-项目现状与架构)
2. [核心功能模块](#2-核心功能模块)
3. [运费计算规则体系](#3-运费计算规则体系)
4. [工程约定与最佳实践](#4-工程约定与最佳实践)
5. [数据库设计](#5-数据库设计)
6. [前端开发指南](#6-前端开发指南)
7. [后端开发指南](#7-后端开发指南)
8. [构建与部署](#8-构建与部署)
9. [踩坑记录与解决方案](#9-踩坑记录与解决方案)
10. [常见问题速查](#10-常见问题速查)

---

## 1. 项目现状与架构

### 1.1 技术栈

| 层级 | 技术 | 说明 |
|---|---|---|
| 前端 | Vue 3 + TypeScript + Vite + Element Plus + Pinia | 单页应用，embed 进 Go 二进制 |
| 后端 | Go 1.22+ | 纯 Go 实现，无 CGO 依赖 |
| 数据库 | SQLite (modernc.org/sqlite) | 纯 Go SQLite 驱动，本地文件存储 |
| Excel | excelize/v2 | 流式读写，支持大文件 |
| 桌面形态 | 单 exe + 系统默认浏览器 | 非 Wails，直接 HTTP 服务 + 打开浏览器 |
| 授权 | RSA + AES + 机器码 | 离线授权 + 在线验证双模式 |

### 1.2 为什么不用 Wails

项目最初设计用 Wails（见 DESIGN.md），实际采用了更轻量的方案：

- Go 启动 HTTP 服务（默认端口 58080）
- 前端资源通过 `//go:embed frontend/dist` 嵌入二进制
- 启动时自动调用系统默认浏览器打开页面
- **优点**：无 WebView2 依赖、兼容所有 Windows 版本、开发调试简单、体积更小
- **缺点**：不是原生窗口形态（在浏览器中打开）

### 1.3 目录结构（最新）

```
mmjs-master/
├── main.go                    # 入口 + HTTP API 路由 + 前端 embed
├── browser_windows.go         # Windows 打开浏览器（ShellExecuteW，无黑框）
├── browser_unix.go            # Unix 打开浏览器
├── ico_to_syso.py             # ICO 转 Windows 资源文件工具
├── rsrc_windows_amd64.syso    # Windows exe 图标资源（编译时自动链接）
├── monkey.ico                 # 猴子图标源文件
├── build.bat / build.ps1      # Windows 构建脚本
├── wails.json                 # （保留，未实际使用 Wails）
│
├── internal/
│   ├── app/app.go             # 门面层，统一对外接口
│   ├── db/sqlite.go           # 数据库层（连接池、迁移、写锁）
│   ├── excel/reader.go        # Excel 读写
│   ├── freight/
│   │   ├── engine.go          # 运费计算引擎
│   │   └── avgweight.go       # 拉均重/偏差加价计算
│   ├── rules/
│   │   ├── models.go          # 数据模型
│   │   ├── repository.go      # 规则数据访问
│   │   ├── global.go          # 全局规则
│   │   ├── province.go        # 省份加价
│   │   ├── zones.go           # 区域管理（六区体系）
│   │   ├── brackets.go        # 重量区间（阶梯定价）
│   │   └── avgweight.go       # 拉均重规则
│   └── license/               # 授权系统
│       ├── crypto.go          # RSA+AES 加密
│       ├── machine.go         # 机器码生成
│       ├── machine_windows.go # Windows 机器码（WMIC，HideWindow）
│       ├── machine_unix.go    # Unix 机器码
│       ├── online.go          # 在线授权
│       └── validator.go       # 授权验证
│
├── frontend/
│   ├── src/
│   │   ├── views/             # 页面
│   │   │   ├── Layout.vue     # 主布局（含左侧导航）
│   │   │   ├── Home.vue       # 首页仪表盘
│   │   │   ├── Calc.vue       # 计费结算
│   │   │   ├── Rules.vue      # 规则管理（最复杂页面）
│   │   │   ├── History.vue    # 历史记录
│   │   │   ├── License.vue    # 授权管理
│   │   │   ├── Settings.vue   # 系统设置
│   │   │   ├── Login.vue      # 登录
│   │   │   └── TestRule.vue   # 规则快速测试
│   │   ├── components/
│   │   │   └── KnowledgeAI.vue # 知识库AI助手（猴子图标）
│   │   ├── stores/app.ts      # Pinia 状态管理
│   │   ├── api/index.ts       # API 封装
│   │   └── router/index.ts    # 路由配置
│   └── public/
│       ├── monkey-icon.png    # 猴子图标（知识库AI用）
│       └── favicon.png        # 网站图标
│
└── server_php/                # PHP 授权服务端
```

---

## 2. 核心功能模块

### 2.1 运费计算

相关文件：
- [freight/engine.go](file:///Users/cxd/mmjs-master/mmjs-master/internal/freight/engine.go) - 基础运费计算
- [freight/avgweight.go](file:///Users/cxd/mmjs-master/mmjs-master/internal/freight/avgweight.go) - 拉均重计算

计算流程（从高到低）：
1. 基础运费（标准模式/阶梯模式）
2. 偏远附加费（surcharge，规则级）
3. 省份加价（全局省份加价）
4. 保底价/最高价（规则自身的 min_fee/max_fee）
5. 全局加价（固定 + 百分比）
6. 保留 2 位小数

### 2.2 规则管理

相关文件：
- [rules/models.go](file:///Users/cxd/mmjs-master/mmjs-master/internal/rules/models.go)
- [rules/repository.go](file:///Users/cxd/mmjs-master/mmjs-master/internal/rules/repository.go)
- [Rules.vue](file:///Users/cxd/mmjs-master/mmjs-master/frontend/src/views/Rules.vue)

**规则类型**：default（保底）、global（全局）、customer（客户）、campaign（活动）

**优先级**：活动 > 客户 > 全局 > 默认

**匹配逻辑**：精确省份优先于全国通配（空省份）

### 2.3 区域体系（六区）

相关文件：[rules/zones.go](file:///Users/cxd/mmjs-master/mmjs-master/internal/rules/zones.go)

- 按地理位置把省份分为 6 个区域
- **港澳台地区作为六区纳入**
- 支持按区域批量生成规则（区域模板生成）
- 阶段报价表区域排序支持中文数字解析（一到十）

### 2.4 阶梯定价（重量区间）

相关文件：[rules/brackets.go](file:///Users/cxd/mmjs-master/mmjs-master/internal/rules/brackets.go)

6 个标准区间：
1. 0 - 0.5kg：一口价（fixed）
2. 0.5 - 1kg：一口价（fixed）
3. 1 - 2kg：一口价（fixed）
4. 2 - 3kg：一口价（fixed）
5. 3 - 30kg：首续重（first_cont）
6. 30kg 以上：首续重（first_cont，weight_to=0 表示无上限）

两种区间类型：
- **一口价（fixed）**：落在区间内固定价格
- **首续重（first_cont）**：以区间起始重量为首重，超出部分按续重算

### 2.5 拉均重/偏差加价

相关文件：
- [rules/avgweight.go](file:///Users/cxd/mmjs-master/mmjs-master/internal/rules/avgweight.go) - 规则管理
- [freight/avgweight.go](file:///Users/cxd/mmjs-master/mmjs-master/internal/freight/avgweight.go) - 计算逻辑

**触发条件**：批次平均重量 > 基准重量

**配置项**：
- 作用范围：全局 / 客户专属（客户优先级更高）
- 基准重量：平均超过此值才加价
- **重量上限**：超过此重量的大包裹不参与计算（默认 3kg，0=不限制）
- 每公斤加价：每超 1kg 每件加多少钱
- 单件最高加价：每件最多加多少（0=不限制）

**计算流程**：按客户分组 → 排除超上限包裹 → 算平均重量 → 判断加价 → 计算金额 → 分摊到每件

### 2.6 省份加价（全局）

相关文件：[rules/province.go](file:///Users/cxd/mmjs-master/mmjs-master/internal/rules/province.go)

- 在系统设置 → 全局规则中配置
- 按目的省份每票加收固定金额
- 所有规则命中后都加（全局级）

### 2.7 知识库 AI 助手

相关文件：[KnowledgeAI.vue](file:///Users/cxd/mmjs-master/mmjs-master/frontend/src/components/KnowledgeAI.vue)

- 左侧浮动猴子图标，可拖动
- 内置 20+ 条知识库条目（运费计算、规则配置、操作指南等）
- 关键词匹配算法：标题匹配 +20 分，关键词匹配按长度加分，>=2 分返回结果
- 区分点击和拖动（movedDuringDrag 标志）
- 支持鼠标和触摸拖动

---

## 3. 运费计算规则体系

### 3.1 完整计算优先级

```
1. 客户专属规则（精确省份）
2. 客户专属规则（全国通配，province 为空）
3. 活动规则（精确省份）
4. 活动规则（全国通配）
5. 全局规则（精确省份）
6. 全局规则（全国通配）
7. 保底规则（default 类型）
8. 兜底 5 元
```

> **注意**：同级别内，精确省份规则优先于全国通配（空省份）规则。

### 3.2 计费模式

| 模式 | 标识 | 说明 |
|---|---|---|
| 标准模式 | simple | 首重 + 续重，传统计费方式 |
| 阶梯模式 | bracket | 按重量区间定价，每区间可独立设置 |

### 3.3 续重模式

| 模式 | 标识 | 说明 |
|---|---|---|
| 整 kg 续重 | full_kg | 向上取整到整 kg |
| 实际重量 | actual_weight | 按实际重量精确计算 |
| 百克续重 | hundred_gram | 每 100g 为单位向上取整 |

### 3.4 加价叠加顺序

```
基础运费
  + 偏远附加费（surcharge，规则级）
  + 省份加价（全局省份加价）
  → 应用保底价/最高价（规则自身的）
  + 全局加价（固定金额 + 百分比）
  = 最终运费
```

---

## 4. 工程约定与最佳实践

### 4.1 数据库操作约定

> **重要**：以下约定是踩过坑后总结的，必须严格遵守！

1. **所有数据库写操作必须通过 `db.WriteExec()` 执行**
   - 使用 `sync.Mutex` 确保串行写入
   - 防止 `SQLITE_BUSY (database is locked)` 错误

2. **数据库连接池 `MaxOpenConns` 不能设为 1**
   - 需要允许并发查询
   - 写操作通过互斥锁串行化，读操作可以并发

3. **`busy_timeout` 需设置为 30 秒**
   - 给并发查询足够的等待时间

4. **查询需先读所有行并关闭 rows，再加载关联数据**
   - 避免连接死锁
   - 正确模式：rows.Next() 收集所有 ID → rows.Close() → 批量查询关联数据

5. **API 返回空列表时必须序列化为 `[]` 而非 `null`**
   - 避免前端白屏
   - 返回前检查：如果切片为 nil，返回空切片 `[]T{}`

### 4.2 性能优化约定

1. **核心计算逻辑中频繁访问的规则数据需预加载到内存 map**
   - 省份加价、拉均重规则等
   - 避免循环内重复数据库查询
   - 用 RuleIndex 实现 O(1) 规则查找

2. **Excel 大文件预览用 ZIP 直读优化**
   - 从 sheet XML 头部 `<dimension>` 获取总行数（O(1)）
   - 只加载需要的共享字符串（SST）条目

### 4.3 前端开发约定

1. **所有 API 调用需增加 `Array.isArray()` 检查**
   - 数据加载和计算属性需增加空值保护
   - 防止后端返回 null 时前端白屏

2. **Pinia store 中定义的函数必须在 return 语句中显式暴露**
   - 否则组件无法访问

3. **Vue 模板中禁止直接使用全局对象**
   - 如 `Math`、`window` 等
   - 需通过组件实例属性/计算属性访问

4. **类型安全**
   - 涉及数值比较时用 `Number()` 做类型转换
   - 特别是 `is_enabled` 等从后端来的字段，类型可能不统一

5. **状态操作后需刷新列表**
   - 规则启用/禁用、删除后立即刷新
   - 确保数据同步

### 4.4 批量导入约定

1. **自动检测新旧模板格式**
   - 第 3 列是 simple/bracket 则为新格式
   - 兼容旧格式只有基础字段的情况

2. **模板需包含 14 列**
   - 新增：计费模式、区域名称、规则类型、启用状态、备注

3. **需读取并处理"区域名称"列**
   - 通过名称自动关联 zone_id

### 4.5 系统设置约定

1. **保存采用合并模式**
   - 仅更新传入字段
   - 空密码不覆盖旧密码

2. **管理员密码修改后需立即刷新 `authSecret` 使其生效**

---

## 5. 数据库设计

### 5.1 核心表清单

| 表名 | 说明 | 关键字段 |
|---|---|---|
| freight_rules | 运费规则 | rule_type, customer_name, province, cont_mode, first_price, cont_price, min_fee, max_fee, surcharge, is_enabled, zone_id, calc_mode |
| weight_brackets | 重量区间 | rule_id, weight_from, weight_to, price_type (fixed/first_cont), first_price, cont_price |
| zones | 区域 | name, zone_order, provinces (JSON) |
| avg_weight_rules | 拉均重规则 | scope (global/customer), customer_name, base_weight, weight_cap, markup_per_kg, max_markup_per_piece, is_enabled |
| province_surcharges | 省份加价 | province, amount |
| global_rules | 全局规则（单行） | default_first_*, no_weight_price, markup_fixed, markup_percent |
| calc_history | 计算历史 | input_file, total_count, total_fee, rule_summary, calc_duration |
| license_info | 授权信息（单行） | machine_code, customer, expires_at, license_raw |
| app_settings | 设置（key-value） | key, value |

### 5.2 规则字段说明

`freight_rules` 表的关键字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| rule_type | TEXT | default / global / customer / campaign |
| calc_mode | TEXT | simple（标准）/ bracket（阶梯） |
| cont_mode | TEXT | full_kg / actual_weight / hundred_gram |
| province | TEXT | 省份名，空字符串=全国通配 |
| zone_id | INTEGER | 所属区域 ID（0=无区域归属） |
| is_enabled | INTEGER | 1=启用，0=禁用 |
| min_fee / max_fee | REAL | 保底价 / 最高价（0=不限制） |
| surcharge | REAL | 偏远附加费 |

---

## 6. 前端开发指南

### 6.1 规则管理页面（Rules.vue）

这是最复杂的页面，包含两个视图：

**阶段报价视图（bracket 模式）**：
- 按区域分组展示阶梯定价
- 支持六个标准重量区间的配置
- 区域排序：一区到六区（中文数字解析）
- 同区域内：阶梯模式在前，标准模式在后
- 最后按省份名称排序

**列表视图**：
- 表格形式展示所有规则
- 支持搜索、筛选、批量操作
- 启用/禁用开关（el-switch）

**重要**：
- toggleRule 使用 switch 的 val 参数，不要用 `row.is_enabled === 1` 自行判断（类型可能不统一）
- 保存失败时需要回滚开关状态
- 编辑规则时传递完整数据，避免零值覆盖

### 6.2 API 调用模式

```typescript
// 统一使用 apiGet / apiPost
import { apiGet, apiPost } from '@/api'

// 获取数据
const data = await apiGet('/api/rules')
if (Array.isArray(data)) {  // 空值保护
  // 处理
}

// 提交数据
const result = await apiPost('/api/rules/save', rule)
if (result && result.ok !== false) {
  // 成功
}
```

### 6.3 Pinia Store 使用

```typescript
// store 中定义的函数必须在 return 中暴露
export const useAppStore = defineStore('app', () => {
  const rules = ref([])

  async function fetchRules() {
    // ...
  }

  return {
    rules,
    fetchRules,  // 必须显式暴露
  }
})
```

---

## 7. 后端开发指南

### 7.1 新增 API 接口

在 [main.go](file:///Users/cxd/mmjs-master/mmjs-master/main.go) 中注册路由：

```go
mux.HandleFunc("/api/your-endpoint", func(w http.ResponseWriter, r *http.Request) {
    // 认证检查（如需）
    if !checkAuth(r) {
        writeJSON(w, map[string]bool{"ok": false})
        return
    }

    if r.Method == "GET" {
        // 处理 GET
    } else if r.Method == "POST" {
        // 处理 POST
    }
})
```

### 7.2 数据库操作

```go
// 查询：直接用 db.Query / db.QueryRow
rows, err := db.Query("SELECT * FROM freight_rules WHERE is_enabled = 1")
// 注意：先读所有行并关闭，再做其他操作

// 写入：必须用 db.WriteExec
result := db.WriteExec(func(tx *sql.Tx) error {
    _, err := tx.Exec("INSERT INTO ...", ...)
    return err
})
```

### 7.3 新增规则类型

1. 在 [rules/models.go](file:///Users/cxd/mmjs-master/mmjs-master/internal/rules/models.go) 添加模型
2. 在 [rules/repository.go](file:///Users/cxd/mmjs-master/mmjs-master/internal/rules/repository.go) 添加 CRUD 方法
3. 在 [freight/engine.go](file:///Users/cxd/mmjs-master/mmjs-master/internal/freight/engine.go) 集成计算逻辑
4. 在 [app/app.go](file:///Users/cxd/mmjs-master/mmjs-master/internal/app/app.go) 添加门面方法
5. 在 [main.go](file:///Users/cxd/mmjs-master/mmjs-master/main.go) 注册 API
6. 在前端 [stores/app.ts](file:///Users/cxd/mmjs-master/mmjs-master/frontend/src/stores/app.ts) 添加状态和方法
7. 在 [api/index.ts](file:///Users/cxd/mmjs-master/mmjs-master/frontend/src/api/index.ts) 添加 API 封装

---

## 8. 构建与部署

### 8.1 前端构建

```bash
cd frontend
npm install
npm run build
```

产物在 `frontend/dist/`，会被 Go embed 进二进制。

### 8.2 Go 交叉编译（Windows amd64）

```bash
# macOS 上交叉编译 Windows 64 位
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
  go build -ldflags="-s -w -H=windowsgui" -o yunfei.exe .
```

参数说明：
- `CGO_ENABLED=0`：纯 Go 编译，无需 C 工具链
- `-s -w`：去掉符号表和调试信息，减小体积
- `-H=windowsgui`：Windows GUI 程序，不显示控制台黑框

### 8.3 图标资源

工具：[ico_to_syso.py](file:///Users/cxd/mmjs-master/mmjs-master/ico_to_syso.py)

```bash
# 生成 amd64 平台的图标资源
python3 ico_to_syso.py monkey.ico rsrc_windows_amd64.syso amd64
```

生成的 `rsrc_windows_amd64.syso` 放在项目根目录，Go 编译时会自动链接。

> **重要**：文件名必须是 `rsrc_windows_amd64.syso`（或 `rsrc_${GOOS}_${GOARCH}.syso`），Go 才会自动识别。

### 8.4 Windows 上构建

使用 [build.bat](file:///Users/cxd/mmjs-master/mmjs-master/build.bat) 或 [build.ps1](file:///Users/cxd/mmjs-master/mmjs-master/build.ps1)。

### 8.5 数据存储位置

- **Windows**：`%APPDATA%\yunfei\`（即 `C:\Users\<用户名>\AppData\Roaming\yunfei\`）
- **macOS/Linux**：`$HOME/yunfei/`
- 可通过环境变量 `YUNFEI_DATA_DIR` 自定义

包含文件：
- `yunfei.db` - SQLite 数据库（WAL 模式下还有 -wal 和 -shm 文件）
- `settings.json` - 设置文件
- `license.dat` - 授权文件

### 8.6 多实例运行

- 启动时检测端口，已有实例在运行则直接打开浏览器并退出
- 新 exe 会因端口占用直接退出，需先结束旧进程再启动新版本

---

## 9. 踩坑记录与解决方案

### 9.1 SQLite 并发写入锁死

**现象**：多个 goroutine 同时写入时出现 `SQLITE_BUSY (database is locked)`

**原因**：SQLite 不支持并发写，多个 goroutine 同时写会冲突

**解决方案**：
- 所有写操作通过 `db.WriteExec()` 执行
- 内部使用 `sync.Mutex` 确保串行写入
- `MaxOpenConns` 保持大于 1（允许并发读）
- 设置 `busy_timeout = 30s`

### 9.2 前端白屏（空列表 null）

**现象**：页面加载后一直转圈或空白

**原因**：后端返回空列表时序列化为 `null`，前端 `Array.isArray()` 检查失败或 v-for 报错

**解决方案**：
- 后端：返回空列表时确保返回 `[]` 而非 `nil`
- 前端：所有 API 调用加 `Array.isArray()` 检查，计算属性加空值保护

### 9.3 规则启用/禁用后价格字段清零

**现象**：切换启用状态后，规则的价格字段变成 0

**原因**：前端 toggleRule 只传 8 个字段，后端 UPDATE 语句覆盖所有 17 个字段，未传的被置 0

**解决方案**：
- 前端：传递完整规则数据（所有字段）
- 后端：SaveRule 先读取现有记录，零值字段保留现有值

### 9.4 列表视图启用按钮无法重新启用

**现象**：禁用规则后，再次点击启用按钮没反应

**原因**：toggleRule 用 `row.is_enabled === 1` 自行判断新状态，`is_enabled` 类型可能不是 number（字符串或其他），严格相等判断失败

**解决方案**：
- 使用 el-switch 的 `val` 参数（`@change="(val) => toggleRule(row, val)"`）
- 加 `Number()` 类型转换确保安全
- 保存失败时回滚开关状态

### 9.5 exe 图标不显示

**现象**：编译后的 exe 在 Windows 上没有猴子图标

**根因排查过程**：
1. 第一次：资源目录 ID 类型条目错误设置了 `0x80000000` 高位（该位只用于字符串名称）
2. 第二次（本次）：group icon entry 用了 16 字节格式（`<BBBBHHHIH`），Windows 标准要求 14 字节（`<BBBBHHIH`）

**最终修复**：
- GRPICONDIRENTRY 必须是 14 字节：bWidth(1) + bHeight(1) + bColorCount(1) + bReserved(1) + wPlanes(2) + wBitCount(2) + dwBytesInRes(4) + nID(2)
- group icon data 总大小 = 6 + n * 14 字节（header + n 个 entry）
- ICO 文件的 entry 是 16 字节（多了 dwImageOffset），不要搞混

**验证方法**：用 Python 解析 PE 文件的资源段，检查 RT_GROUP_ICON 的数据大小和 entry 格式。

### 9.6 程序启动黑框

**现象**：启动 exe 时弹出黑色控制台窗口

**原因**：编译时缺少 `-H=windowsgui` 链接标志

**解决方案**：
- 编译参数加 `-ldflags="-H=windowsgui"`
- 打开浏览器用 `ShellExecuteW`（Windows），不用 exec.Command
- wmic 命令设 `HideWindow: true`

### 9.7 规则模板生成后重复

**现象**：多次生成区域模板后，规则重复

**解决方案**：生成前自动删除该客户已有的区域型规则（非区域型的全国通配/自定义省份规则不删）

### 9.8 客户名称不一致导致规则不匹配

**现象**：配置了规则但计算时不生效

**原因**：客户名称在规则配置和 Excel 账单中不完全一致（如"珀莱雅" vs "铂莱雅"）

**解决方案**：
- 增加客户名称标准化逻辑（去除空格、统一大小写）
- 规则匹配时做模糊匹配（如包含关系）

### 9.9 阶段报价表省份消失

**现象**：禁用规则后，阶段报价视图里省份消失了

**原因**：前端 toggleRule 没传 zone_id，后端更新时 zone_id 被设为 0，导致 zone_name 查询为空被过滤

**解决方案**：
- 前端 toggleRule 传递完整规则数据（含 zone_id）
- 后端 SaveRule 中如果传入 zone_id 为 0，使用数据库原有值

### 9.10 数据库体积增长

**说明**：
- 规则数据量极小，主要增长来自计算历史记录
- 正常使用下总大小通常 1-5MB
- WAL 模式会产生 -wal 和 -shm 额外文件，程序正常退出时 WAL 合并回主文件
- 删除数据后文件大小不会自动缩小（SQLite 特性）
- 系统已有"清空历史记录"功能，但无自动 VACUUM

---

## 10. 常见问题速查

### 开发相关

| 问题 | 答案 |
|---|---|
| 前端怎么启动？ | `cd frontend && npm run dev`（端口 5173） |
| 后端怎么启动？ | `go run main.go`（端口 58080） |
| 默认账号密码？ | admin / admin123 |
| 数据库在哪？ | Windows: `%APPDATA%\yunfei\yunfei.db` |
| 怎么加新规则字段？ | 改 models.go → repository.go → 前端 store 和视图 |

### 业务相关

| 问题 | 答案 |
|---|---|
| 规则优先级？ | 活动 > 客户 > 全局 > 默认，精确省 > 全国通配 |
| 六区是哪六区？ | 一区到六区，港澳台在六区 |
| 重量上限什么意思？ | 拉均重功能中，超过此重量的包裹不参与平均计算 |
| 保底规则能删吗？ | 不能，是系统兜底 |
| 阶梯定价有几个区间？ | 6 个标准区间，可自定义价格 |

### 运维相关

| 问题 | 答案 |
|---|---|
| exe 图标不显示？ | 确认 rsrc_windows_amd64.syso 存在且格式正确（见 9.5） |
| 启动有黑框？ | 编译加 `-H=windowsgui`（见 9.6） |
| 端口被占用？ | 运行 killport.bat，或等几秒自动打开已有实例 |
| 数据库很大？ | 清空历史记录，或手动 VACUUM |
| 授权失效？ | 检查机器码是否变化，联系客服重新授权 |

---

*文档版本: v1.0*
*最后更新: 2026-07-04*
