package main

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"yunfei/internal/app"
	"yunfei/internal/db"
	"yunfei/internal/excel"
	"yunfei/internal/freight"
	"yunfei/internal/rules"

	"github.com/xuri/excelize/v2"
)

//go:embed frontend/dist
var frontendAssets embed.FS

var (
	a          *app.App
	lastResult *app.CalcResult // 缓存上一次计算结果，供导出用
)

// ========== 进度追踪 ==========
type TaskProgress struct {
	TaskID    string  `json:"task_id"`
	Phase     string  `json:"phase"` // reading / calculating / done / error
	Current   int     `json:"current"`
	Total     int     `json:"total"`
	Pct       int     `json:"pct"`
	Message   string  `json:"message"`
	Error     string  `json:"error,omitempty"`
	UpdatedAt int64   `json:"updated_at"`
}

// BatchTaskInfo 批量任务中单个文件的信息
type BatchTaskInfo struct {
	TaskID   string `json:"task_id"`
	FileName string `json:"file_name"`
	FilePath string `json:"file_path"`
}

// BatchProgress 批量进度汇总
type BatchProgress struct {
	BatchID string          `json:"batch_id"`
	Tasks   []BatchTaskInfo `json:"tasks"`
}

var (
	progressStore   = sync.Map{} // taskID -> *TaskProgress
	taskResults     = sync.Map{} // taskID -> *app.CalcResult
	batchProgresses = sync.Map{} // batchID -> *BatchProgress
	previewStore    = sync.Map{} // path -> *ExcelPreview (cached)
)

// ========== 简单认证 ==========
var authSecret = "yunfei@2024" // 默认密钥
func initAuth() {
	dataDir := db.GetDataDir()
	settingsFile := filepath.Join(dataDir, "settings.json")
	if data, err := os.ReadFile(settingsFile); err == nil {
		var s map[string]interface{}
		if json.Unmarshal(data, &s) == nil {
			if sec, ok := s["auth_secret"].(string); ok && sec != "" {
				authSecret = sec
			}
		}
	}
}
func generateToken(username string) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d", username, authSecret, time.Now().Unix()/3600)))
	return hex.EncodeToString(h[:])[:32]
}
func verifyToken(token string) bool {
	// 简单验证：重新计算当前小时的token
	now := time.Now().Unix() / 3600
	for _, u := range []string{"admin", "user"} {
		h := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d", u, authSecret, now)))
		if hex.EncodeToString(h[:])[:32] == token {
			return true
		}
		// 也检查上一小时（避免边界情况）
		h = sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d", u, authSecret, now-1)))
		if hex.EncodeToString(h[:])[:32] == token {
			return true
		}
	}
	return false
}

func main() {
	a = app.New()
	a.Startup(nil)
	defer a.Shutdown(nil)
	initAuth()

	mux := http.NewServeMux()

	// ========== 认证接口 ==========
	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			writeJSON(w, map[string]string{"error": "method not allowed"})
			return
		}
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		// 从 settings 读取用户名密码，默认 admin/admin123
		dataDir := db.GetDataDir()
		settingsFile := filepath.Join(dataDir, "settings.json")
		adminUser := "admin"
		adminPass := "admin123"
		if data, err := os.ReadFile(settingsFile); err == nil {
			var s map[string]interface{}
			if json.Unmarshal(data, &s) == nil {
				if u, ok := s["admin_user"].(string); ok && u != "" { adminUser = u }
				if p, ok := s["admin_pass"].(string); ok && p != "" { adminPass = p }
			}
		}

		if req.Username == adminUser && req.Password == adminPass {
			token := generateToken(req.Username)
			writeJSON(w, map[string]interface{}{
				"ok": true, "token": token,
				"username": req.Username,
			})
		} else {
			writeJSON(w, map[string]interface{}{"ok": false, "error": "用户名或密码错误"})
		}
	})
	mux.HandleFunc("/api/auth/verify", func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		token = strings.TrimPrefix(token, "Bearer ")
		if token == "" {
			token = r.URL.Query().Get("token")
		}
		writeJSON(w, map[string]bool{"ok": verifyToken(token)})
	})

	// API路由
	mux.HandleFunc("/api/machine-code", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"code": a.GetMachineCode()})
	})
	mux.HandleFunc("/api/license/import", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ License string }
		json.NewDecoder(r.Body).Decode(&req)
		writeJSON(w, a.ImportLicense(req.License))
	})
	mux.HandleFunc("/api/license/info", func(w http.ResponseWriter, r *http.Request) {
		info := a.GetLicenseInfo()
		if info.IsValid {
			writeJSON(w, info)
			return
		}
		// 离线授权无效时，尝试在线检查
		online := a.CheckOnlineLicense()
		if valid, _ := online["valid"].(bool); valid {
			info.IsValid = true
			info.Customer = online["customer_name"].(string)
			info.ExpiresAt = online["expires_at"].(string)
			days, _ := online["days_left"].(float64)
			info.DaysLeft = int(days)
		}
		writeJSON(w, info)
	})
	// 在线授权（不需要登录 token）
	mux.HandleFunc("/api/license/check-online", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, a.CheckOnlineLicense())
	})
	mux.HandleFunc("/api/license/activate-online", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			writeJSON(w, map[string]string{"ok": "false", "msg": "method not allowed"})
			return
		}
		var req struct{ LicenseKey string `json:"license_key"` }
		json.NewDecoder(r.Body).Decode(&req)
		writeJSON(w, a.ActivateOnline(req.LicenseKey))
	})
	// 新版：使用加密授权数据激活（RSA+AES）
	mux.HandleFunc("/api/license/activate-license-data", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			writeJSON(w, map[string]string{"ok": "false", "msg": "method not allowed"})
			return
		}
		var req struct{ LicenseData string `json:"license_data"` }
		json.NewDecoder(r.Body).Decode(&req)
		writeJSON(w, a.ActivateWithLicenseData(req.LicenseData))
	})
	// 授权服务器地址配置（可手动修改主/备地址）
	mux.HandleFunc("/api/license/server-info", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, a.GetServerInfo())
	})
	mux.HandleFunc("/api/license/set-server", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			writeJSON(w, map[string]string{"ok": "false"})
			return
		}
		var req struct{ URL string `json:"url"` }
		json.NewDecoder(r.Body).Decode(&req)
		a.SetServerURL(req.URL)
		writeJSON(w, map[string]bool{"ok": true})
	})
	// 重置授权（清除激活状态）
	mux.HandleFunc("/api/license/reset", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			writeJSON(w, map[string]bool{"ok": false})
			return
		}
		a.ResetLicense()
		writeJSON(w, map[string]bool{"ok": true})
	})
	// API 密钥配置（需与服务器 api_secret 一致）
	mux.HandleFunc("/api/license/api-secret", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			writeJSON(w, map[string]string{"api_secret": a.GetApiSecret()})
			return
		}
		if r.Method == "POST" {
			var req struct{ Secret string `json:"api_secret"` }
			json.NewDecoder(r.Body).Decode(&req)
			if req.Secret != "" {
				a.SetApiSecret(req.Secret)
			}
			writeJSON(w, map[string]bool{"ok": true})
			return
		}
	})
	mux.HandleFunc("/api/license/set-backup-server", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			writeJSON(w, map[string]string{"ok": "false"})
			return
		}
		var req struct{ URL string `json:"url"` }
		json.NewDecoder(r.Body).Decode(&req)
		a.SetBackupServerURL(req.URL)
		writeJSON(w, map[string]bool{"ok": true})
	})
	mux.HandleFunc("/api/rules", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			customer := r.URL.Query().Get("customer")
			if customer != "" {
				writeJSON(w, a.GetRulesByCustomer(customer))
			} else {
				writeJSON(w, a.GetRules())
			}
		} else {
			writeJSON(w, map[string]string{"error": "method not allowed"})
		}
	})
	mux.HandleFunc("/api/rules/save", func(w http.ResponseWriter, r *http.Request) {
		var rule app.RuleSaveReq
		json.NewDecoder(r.Body).Decode(&rule)
		writeJSON(w, map[string]int64{"id": a.SaveRule(rule)})
	})
	mux.HandleFunc("/api/rules/delete", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ ID int64 }
		json.NewDecoder(r.Body).Decode(&req)
		writeJSON(w, map[string]bool{"ok": a.DeleteRule(req.ID)})
	})
	mux.HandleFunc("/api/rules/delete-batch", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ IDs []int64 }
		json.NewDecoder(r.Body).Decode(&req)
		writeJSON(w, map[string]bool{"ok": a.DeleteRulesBatch(req.IDs)})
	})
	// 一键初始化规则（从 Python Zone 体系导入）
	mux.HandleFunc("/api/rules/seed", func(w http.ResponseWriter, r *http.Request) {
		count := seedDefaultRules()
		writeJSON(w, map[string]interface{}{"ok": true, "count": count})
	})
	// 规则详情（含重量区间）
	mux.HandleFunc("/api/rules/detail", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		id, _ := strconv.ParseInt(idStr, 10, 64)
		writeJSON(w, a.GetRuleDetail(id))
	})

	// ========== 区域管理 ==========
	mux.HandleFunc("/api/zones", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			writeJSON(w, a.GetZones())
		} else if r.Method == "POST" {
			var z rules.Zone
			json.NewDecoder(r.Body).Decode(&z)
			id := a.SaveZone(z)
			writeJSON(w, map[string]int64{"id": id})
		}
	})
	mux.HandleFunc("/api/zones/delete", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ ID int64 }
		json.NewDecoder(r.Body).Decode(&req)
		writeJSON(w, map[string]bool{"ok": a.DeleteZone(req.ID)})
	})
	mux.HandleFunc("/api/zones/templates", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, a.GetZoneTemplates())
	})

	// ========== 拉均重规则 ==========
	mux.HandleFunc("/api/avg-weight-rules", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			writeJSON(w, a.GetAvgWeightRules())
		} else if r.Method == "POST" {
			var rl rules.AvgWeightRule
			json.NewDecoder(r.Body).Decode(&rl)
			id := a.SaveAvgWeightRule(rl)
			writeJSON(w, map[string]int64{"id": id})
		}
	})
	mux.HandleFunc("/api/avg-weight-rules/delete", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ ID int64 }
		json.NewDecoder(r.Body).Decode(&req)
		writeJSON(w, map[string]bool{"ok": a.DeleteAvgWeightRule(req.ID)})
	})
	mux.HandleFunc("/api/avg-weight", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			customer := r.URL.Query().Get("customer")
			rule := rules.GetAvgWeightRuleByCustomer(customer)
			if rule == nil {
				writeJSON(w, map[string]string{})
			} else {
				writeJSON(w, rule)
			}
		}
	})
	mux.HandleFunc("/api/avg-weight/toggle", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID        int64 `json:"id"`
			IsEnabled int   `json:"is_enabled"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.ID > 0 {
			rules.ToggleAvgWeight(req.ID, req.IsEnabled)
		}
		writeJSON(w, map[string]bool{"ok": true})
	})

	// ========== 重量区间管理 ==========
	mux.HandleFunc("/api/rules/brackets", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			idStr := r.URL.Query().Get("rule_id")
			var ruleID int64
			fmt.Sscanf(idStr, "%d", &ruleID)
			writeJSON(w, a.GetRuleBrackets(ruleID))
		} else if r.Method == "POST" {
			var req struct {
				RuleID   int64                `json:"rule_id"`
				Brackets []rules.WeightBracket `json:"brackets"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			writeJSON(w, map[string]bool{"ok": a.SaveRuleBrackets(req.RuleID, req.Brackets)})
		}
	})

	// ========== 区域规则模板生成 ==========
	mux.HandleFunc("/api/zones/generate-rules", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			CustomerName string                          `json:"customer_name"`
			ContMode     string                          `json:"cont_mode"`
			CalcMode     string                          `json:"calc_mode"`
			PriceTable   map[string]rules.ZonePriceScheme `json:"price_table"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		result := a.GenerateZoneRulesByTemplate(req.CustomerName, req.ContMode, req.CalcMode, req.PriceTable)
		writeJSON(w, result)
	})
	mux.HandleFunc("/api/zones/sample-price", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, a.GetSamplePriceTable())
	})
	mux.HandleFunc("/api/zones/import-price", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			writeJSON(w, map[string]interface{}{"ok": false, "error": "method not allowed"})
			return
		}
		r.ParseMultipartForm(10 << 20)
		file, _, err := r.FormFile("file")
		if err != nil {
			writeJSON(w, map[string]interface{}{"ok": false, "error": "请选择文件"})
			return
		}
		defer file.Close()
		f, err := excelize.OpenReader(file)
		if err != nil {
			writeJSON(w, map[string]interface{}{"ok": false, "error": "无法解析文件: " + err.Error()})
			return
		}
		defer f.Close()
		rows, err := f.GetRows(f.GetSheetName(0))
		if err != nil {
			writeJSON(w, map[string]interface{}{"ok": false, "error": "无法读取表格"})
			return
		}
		table, errMsg := a.ImportSamplePriceFromExcel(rows)
		if table == nil {
			writeJSON(w, map[string]interface{}{"ok": false, "error": errMsg})
			return
		}
		writeJSON(w, map[string]interface{}{"ok": true, "data": table, "count": len(table)})
	})
	mux.HandleFunc("/api/zones/price-template", func(w http.ResponseWriter, r *http.Request) {
		tf := excelize.NewFile()
		sheet := tf.GetSheetName(0)
		headers := []string{"区域", "0~0.5kg", "0.5~1kg", "1~2kg", "2~3kg", "3~30kg首重", "3~30kg续重", "30kg+首重", "30kg+续重"}
		for i, h := range headers {
			cell, _ := excelize.CoordinatesToCellName(i+1, 1)
			tf.SetCellValue(sheet, cell, h)
		}
		sample := rules.GetSamplePriceTable()
		zoneOrder := []string{"一区", "二区", "三区", "四区", "五区", "六区"}
		for i, zn := range zoneOrder {
			p := sample[zn]
			vals := []interface{}{
				zn, p.Price0_05, p.Price05_1, p.Price1_2, p.Price2_3,
				p.First3_30, p.Cont3_30, p.First30up, p.Cont30up,
			}
			for j, v := range vals {
				cell, _ := excelize.CoordinatesToCellName(j+1, i+2)
				tf.SetCellValue(sheet, cell, v)
			}
		}
		for i := 1; i <= len(headers); i++ {
			col, _ := excelize.ColumnNumberToName(i)
			tf.SetColWidth(sheet, col, col, 14)
		}
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", `attachment; filename="区域参考价模板.xlsx"`)
		tf.Write(w)
	})

	// 全局规则
	mux.HandleFunc("/api/global-rules", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			writeJSON(w, a.GetGlobalRules())
		} else if r.Method == "POST" {
			var gr rules.GlobalRule
			json.NewDecoder(r.Body).Decode(&gr)
			writeJSON(w, map[string]bool{"ok": a.SaveGlobalRules(&gr)})
		}
	})

	// ========== 全局省份加价 ==========
	mux.HandleFunc("/api/province-surcharges", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			writeJSON(w, a.GetProvinceSurcharges())
		} else if r.Method == "POST" {
			var p rules.ProvinceSurcharge
			json.NewDecoder(r.Body).Decode(&p)
			id := a.SaveProvinceSurcharge(p)
			writeJSON(w, map[string]int64{"id": id})
		}
	})
	mux.HandleFunc("/api/province-surcharges/delete", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ ID int64 }
		json.NewDecoder(r.Body).Decode(&req)
		writeJSON(w, map[string]bool{"ok": a.DeleteProvinceSurcharge(req.ID)})
	})

	// ========== 规则快速测试 ==========
	mux.HandleFunc("/api/rules/test", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			writeJSON(w, map[string]string{"error": "method not allowed"})
			return
		}
		var req struct {
			Customer string   `json:"customer"`
			Province string   `json:"province"`
			Weight   float64  `json:"weight"`
			Weights  []float64 `json:"weights"`
			Batch    bool     `json:"batch"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.Batch {
			result := a.TestRuleBatch(req.Customer, req.Province, req.Weights)
			writeJSON(w, result)
		} else {
			result := a.TestRule(req.Customer, req.Province, req.Weight)
			writeJSON(w, result)
		}
	})

	// ========== 导出客户规则 ==========
	mux.HandleFunc("/api/customers/export", func(w http.ResponseWriter, r *http.Request) {
		customerName := r.URL.Query().Get("customer")
		rows, errMsg := a.ExportCustomerRules(customerName)
		if errMsg != "" {
			writeJSON(w, map[string]interface{}{"ok": false, "error": errMsg})
			return
		}
		tf := excelize.NewFile()
		sheet := tf.GetSheetName(0)
		for i, row := range rows {
			for j, v := range row {
				cell, _ := excelize.CoordinatesToCellName(j+1, i+1)
				tf.SetCellValue(sheet, cell, v)
			}
		}
		for i := 0; i < len(rows[0]); i++ {
			col, _ := excelize.ColumnNumberToName(i + 1)
			tf.SetColWidth(sheet, col, col, 18)
		}
		filename := "全部客户规则.xlsx"
		if customerName != "" {
			filename = customerName + "_规则.xlsx"
		}
		escaped := url.QueryEscape(filename)
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, filename, escaped))
		tf.Write(w)
	})

	// 客户管理
	mux.HandleFunc("/api/customers", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, a.GetCustomers())
	})
	mux.HandleFunc("/api/customers/delete", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Name string }
		json.NewDecoder(r.Body).Decode(&req)
		writeJSON(w, map[string]bool{"ok": a.DeleteCustomer(req.Name)})
	})
	mux.HandleFunc("/api/customers/copy-rules", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			From string `json:"from"`
			To   string `json:"to"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.From == "" || req.To == "" {
			writeJSON(w, map[string]interface{}{"ok": false, "error": "客户名不能为空"})
			return
		}
		if req.From == req.To {
			writeJSON(w, map[string]interface{}{"ok": false, "error": "源客户与目标客户相同"})
			return
		}
		count := a.CopyCustomerRules(req.From, req.To)
		writeJSON(w, map[string]interface{}{"ok": count > 0, "count": count})
	})
	mux.HandleFunc("/api/customers/import", func(w http.ResponseWriter, r *http.Request) {
		r.ParseMultipartForm(10 << 20)
		file, _, err := r.FormFile("file")
		if err != nil {
			writeJSON(w, map[string]interface{}{"ok": false, "error": "请选择文件"})
			return
		}
		defer file.Close()
		// 使用 excelize 读取
		f, err := excelize.OpenReader(file)
		if err != nil {
			writeJSON(w, map[string]interface{}{"ok": false, "error": "无法解析文件: " + err.Error()})
			return
		}
		defer f.Close()
		rows, err := f.GetRows(f.GetSheetName(0))
		if err != nil {
			writeJSON(w, map[string]interface{}{"ok": false, "error": "无法读取表格"})
			return
		}
		count, errMsg := a.ImportCustomerRules(rows)
		writeJSON(w, map[string]interface{}{"ok": count > 0, "count": count, "error": errMsg})
	})
	mux.HandleFunc("/api/customers/template", func(w http.ResponseWriter, r *http.Request) {
		tf := excelize.NewFile()
		sheet := tf.GetSheetName(0)
		headers := []string{"客户名称"}
		for i, h := range headers {
			cell, _ := excelize.CoordinatesToCellName(i+1, 1)
			tf.SetCellValue(sheet, cell, h)
		}
		example := []string{"示例客户A"}
		for i, v := range example {
			cell, _ := excelize.CoordinatesToCellName(i+1, 2)
			tf.SetCellValue(sheet, cell, v)
		}
		example2 := []string{"示例客户B"}
		for i, v := range example2 {
			cell, _ := excelize.CoordinatesToCellName(i+1, 3)
			tf.SetCellValue(sheet, cell, v)
		}
		tf.SetColWidth(sheet, "A", "A", 25)
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", `attachment; filename="客户导入模板.xlsx"`)
		tf.Write(w)
	})
	// 文件上传（支持单文件和多文件）
	mux.HandleFunc("/api/excel/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			writeJSON(w, map[string]string{"error": "method not allowed"})
			return
		}
		r.ParseMultipartForm(200 << 20) // 200MB
		// 支持多文件上传
		files := r.MultipartForm.File["files"]
		if len(files) == 0 {
			// 兼容单文件：字段名为 file
			file, header, err := r.FormFile("file")
			if err != nil {
				writeJSON(w, map[string]string{"error": "请选择文件"})
				return
			}
			defer file.Close()
			tmpDir := filepath.Join(os.TempDir(), "yunfei_uploads")
			os.MkdirAll(tmpDir, 0755)
			savePath := filepath.Join(tmpDir, header.Filename)
			dst, _ := os.Create(savePath)
			defer dst.Close()
			io.Copy(dst, file)
			writeJSON(w, map[string]interface{}{"files": []map[string]string{{"path": savePath, "name": header.Filename}}})
			return
		}
		// 多文件模式
		if len(files) > 5 {
			writeJSON(w, map[string]string{"error": "一次最多上传5个文件"})
			return
		}
		var results []map[string]string
		tmpDir := filepath.Join(os.TempDir(), "yunfei_uploads")
		os.MkdirAll(tmpDir, 0755)
		for _, fh := range files {
			src, err := fh.Open()
			if err != nil { continue }
			savePath := filepath.Join(tmpDir, fh.Filename)
			dst, _ := os.Create(savePath)
			io.Copy(dst, src)
			dst.Close()
			src.Close()
			results = append(results, map[string]string{"path": savePath, "name": fh.Filename})
		}
		writeJSON(w, map[string]interface{}{"files": results})
	})
	mux.HandleFunc("/api/excel/preview", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Path string }
		json.NewDecoder(r.Body).Decode(&req)
		// 缓存预览结果
		if cached, ok := previewStore.Load(req.Path); ok {
			writeJSON(w, cached)
			return
		}
		preview := a.ReadExcelPreview(req.Path)
		previewStore.Store(req.Path, preview)
		writeJSON(w, preview)
	})
	// 多文件预览
	mux.HandleFunc("/api/excel/preview-multi", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Files []struct{ Path string; Name string } }
		json.NewDecoder(r.Body).Decode(&req)
		type previewItem struct {
			Name     string  `json:"name"`
			Path     string  `json:"path"`
			TotalRows int    `json:"total_rows"`
			Customers []string `json:"customers"`
			Provinces []string `json:"provinces"`
			Columns  []string `json:"columns"`
			Samples  [][]string `json:"samples"`
			Error    string  `json:"error,omitempty"`
		}
		var results []previewItem
		for _, f := range req.Files {
			var cached previewItem
			if v, ok := previewStore.Load(f.Path); ok {
				p := v.(*excel.ExcelPreview)
				cached = previewItem{Name: f.Name, Path: f.Path, TotalRows: p.TotalRows, Customers: p.Customers, Provinces: p.Provinces, Columns: p.Columns, Samples: p.Samples}
			} else {
				p := a.ReadExcelPreview(f.Path)
				previewStore.Store(f.Path, p)
				cached = previewItem{Name: f.Name, Path: f.Path, TotalRows: p.TotalRows, Customers: p.Customers, Provinces: p.Provinces, Columns: p.Columns, Samples: p.Samples}
				if strings.HasPrefix(p.Columns[0], "ERROR:") {
					cached.Error = p.Columns[0]
				}
			}
			results = append(results, cached)
		}
		writeJSON(w, results)
	})
	mux.HandleFunc("/api/calculate/batch", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Files []struct {
				Path string `json:"path"`
				Name string `json:"name"`
			} `json:"files"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if len(req.Files) == 0 {
			writeJSON(w, map[string]string{"error": "no files"})
			return
		}
		batchID := fmt.Sprintf("batch_%d", time.Now().UnixNano())
		bp := &BatchProgress{BatchID: batchID}
		var tasks []BatchTaskInfo
		for _, f := range req.Files {
			taskID := fmt.Sprintf("calc_%d_%s", time.Now().UnixNano(), sanitizeFilename(f.Name))
			tasks = append(tasks, BatchTaskInfo{TaskID: taskID, FileName: f.Name, FilePath: f.Path})
		}
		bp.Tasks = tasks
		batchProgresses.Store(batchID, bp)

		// 并行启动计算
		allRules, _ := rules.GetAll()
		gr := rules.GetGlobalRules()
		ruleIdx := rules.BuildRuleIndex(allRules, gr)
		// 预加载重量区间数据
		bracketMap, _ := rules.LoadRuleBrackets(allRules)
		// 预加载省份加价数据（避免每行查数据库，大幅提升性能）
		provSurchargeMap := make(map[string]float64)
		if provList, err := rules.GetAllProvinceSurcharges(); err == nil {
			for _, p := range provList {
				provKey := rules.NormalizeProvince(p.ProvinceName)
				provSurchargeMap[provKey] = p.Surcharge
			}
		}
		for _, t := range tasks {
			taskID := t.TaskID
			filePath := t.FilePath
			go func() {
				prog := &TaskProgress{TaskID: taskID, Phase: "reading", Pct: 0, Message: "读取中...", UpdatedAt: time.Now().UnixMilli()}
				progressStore.Store(taskID, prog)

				rowData, _, err := excel.ReadAllRows(filePath, func(cur, total int) {
					prog.Current = cur
					prog.Message = fmt.Sprintf("读取 %d 行", cur)
					prog.UpdatedAt = time.Now().UnixMilli()
				})
				if err != nil {
					prog.Phase = "error"; prog.Error = err.Error(); prog.Message = "读取失败"
					prog.UpdatedAt = time.Now().UnixMilli()
					return
				}

				total := len(rowData)
				prog.Phase = "calculating"
				prog.Total = total
				prog.Message = "计算中..."
				prog.UpdatedAt = time.Now().UnixMilli()

				// 多核并行计算
				numWorkers := runtime.NumCPU()
				if numWorkers < 1 { numWorkers = 1 }
				if numWorkers > 16 { numWorkers = 16 }
				chunkSize := (total + numWorkers - 1) / numWorkers

				var wg sync.WaitGroup
				var processed atomic.Int64
				var markupCents atomic.Int64
				reportEvery := total / 200
				if reportEvery < 100 { reportEvery = 100 }
				if reportEvery > 5000 { reportEvery = 5000 }

				startT := time.Now()
				for w := 0; w < numWorkers; w++ {
					start := w * chunkSize
					end := start + chunkSize
					if end > total { end = total }
					if start >= end { continue }
					wg.Add(1)
					go func(start, end int) {
						defer wg.Done()
						for i := start; i < end; i++ {
							fee, _, markup, _, best := freight.CalcSingleWithIndexFast(
								rowData[i].Weight, rowData[i].Customer, rowData[i].Province, ruleIdx, gr, bracketMap, provSurchargeMap)
							rowData[i].Fee = fee
							markupCents.Add(int64(markup * 100))
							if best != nil {
								rowData[i].RuleLevel = best.RuleLevel
								rowData[i].ContMode = best.Rule.ContMode
								rowData[i].CalcMode = best.Rule.CalcMode
								rowData[i].ZoneName = best.Rule.ZoneName
							}
							cur := processed.Add(1)
							if cur%int64(reportEvery) == 0 {
								pct := int(cur) * 100 / total
								prog.Current = int(cur)
								prog.Pct = pct
								prog.Message = fmt.Sprintf("计算 %d/%d", cur, total)
								prog.UpdatedAt = time.Now().UnixMilli()
							}
						}
					}(start, end)
				}
				wg.Wait()

				// 拉均重偏差加价
				avgResults, totalAvgMarkup := freight.CalcAvgWeightMarkup(rowData)
				if len(avgResults) > 0 && totalAvgMarkup > 0 {
					freight.ApplyAvgWeightToRows(rowData, avgResults)
				}

				duration := math.Round(time.Since(startT).Seconds()*100) / 100
				summary := excel.BuildSummary(rowData, duration)
				tMarkup := float64(markupCents.Load()) / 100.0
				if tMarkup > 0 {
					summary.TotalMarkup = math.Round(tMarkup*100) / 100
				}
				summary.TotalAvgMarkup = math.Round(totalAvgMarkup*100) / 100
				if len(avgResults) > 0 {
					avgInterfaces := make([]interface{}, len(avgResults))
					for i, r := range avgResults {
						avgInterfaces[i] = map[string]interface{}{
							"customer":        r.Customer,
							"avg_weight":      r.AvgWeight,
							"base_weight":     r.BaseWeight,
							"deviation":       r.Deviation,
							"steps":           r.Steps,
							"step_price":      r.StepPrice,
							"per_item_markup": r.PerItemMarkup,
							"item_count":      r.ItemCount,
							"total_markup":    r.TotalMarkup,
						}
					}
					summary.AvgWeightResults = avgInterfaces
				}
				prog.Phase = "done"; prog.Pct = 100; prog.Current = total; prog.Total = total
				prog.Message = "完成"
				prog.UpdatedAt = time.Now().UnixMilli()
				taskResults.Store(taskID, &app.CalcResult{Data: rowData, Summary: summary, FileName: t.FileName})

				// 保存历史记录
				ruleSummary := fmt.Sprintf("%d条规则", len(allRules))
				if _, err := db.WriteExec(`INSERT INTO calc_history (input_file, total_count, total_fee, avg_fee, max_fee, min_fee, rule_summary, calc_duration)
					VALUES (?,?,?,?,?,?,?,?)`,
					t.FileName, summary.TotalCount, summary.TotalFee, summary.AvgFee, summary.MaxFee, summary.MinFee, ruleSummary, duration); err != nil {
					println("[WARN] 批量保存计算历史失败:", err.Error())
				}
			}()
		}

		writeJSON(w, map[string]interface{}{"batch_id": batchID, "tasks": tasks})
	})
	// 单文件计算（兼容旧）
	mux.HandleFunc("/api/calculate", func(w http.ResponseWriter, r *http.Request) {
		var req app.CalcRequest
		json.NewDecoder(r.Body).Decode(&req)
		taskID := fmt.Sprintf("calc_%d", time.Now().UnixNano())

		go func() {
			prog := &TaskProgress{TaskID: taskID, Phase: "reading", Current: 0, Total: 0, Pct: 0, Message: "正在读取Excel...", UpdatedAt: time.Now().UnixMilli()}
			progressStore.Store(taskID, prog)

			result := a.CalculateFreightWithProgress(req, func(phase string, current, total int, msg string) {
				pct := 0
				if total > 0 {
					pct = current * 100 / total
					if pct > 100 { pct = 100 }
				}
				prog.Phase = phase
				prog.Current = current
				prog.Total = total
				prog.Pct = pct
				prog.Message = msg
				prog.UpdatedAt = time.Now().UnixMilli()
			})
			if result.Error != "" {
				prog.Phase = "error"
				prog.Error = result.Error
				prog.Message = "计算失败: " + result.Error
			} else {
				prog.Phase = "done"
				prog.Pct = 100
				prog.Message = "计算完成"
			}
			prog.UpdatedAt = time.Now().UnixMilli()
			taskResults.Store(taskID, result)
			lastResult = result
		}()

		writeJSON(w, map[string]string{"task_id": taskID, "status": "started"})
	})
	// 批量进度轮询
	mux.HandleFunc("/api/calculate/batch-progress", func(w http.ResponseWriter, r *http.Request) {
		batchID := r.URL.Query().Get("batch_id")
		if batchID == "" {
			writeJSON(w, map[string]string{"error": "missing batch_id"})
			return
		}
		bpVal, ok := batchProgresses.Load(batchID)
		if !ok {
			writeJSON(w, map[string]string{"error": "batch not found"})
			return
		}
		bp := bpVal.(*BatchProgress)
		type taskProgResp struct {
			TaskID   string `json:"task_id"`
			FileName string `json:"file_name"`
			Phase    string `json:"phase"`
			Pct      int    `json:"pct"`
			Current  int    `json:"current"`
			Total    int    `json:"total"`
			Message  string `json:"message"`
			Error    string `json:"error,omitempty"`
		}
		var tasks []taskProgResp
		allDone := true
		for _, t := range bp.Tasks {
			tp := taskProgResp{TaskID: t.TaskID, FileName: t.FileName}
			if v, ok := progressStore.Load(t.TaskID); ok {
				p := v.(*TaskProgress)
				tp.Phase = p.Phase; tp.Pct = p.Pct; tp.Current = p.Current; tp.Total = p.Total; tp.Message = p.Message; tp.Error = p.Error
				if p.Phase != "done" && p.Phase != "error" {
					allDone = false
				}
			} else {
				tp.Phase = "waiting"; tp.Message = "等待中"
				allDone = false
			}
			tasks = append(tasks, tp)
		}
		writeJSON(w, map[string]interface{}{"batch_id": batchID, "tasks": tasks, "all_done": allDone})
	})
	// 批量结果获取
	mux.HandleFunc("/api/calculate/batch-result", func(w http.ResponseWriter, r *http.Request) {
		batchID := r.URL.Query().Get("batch_id")
		if batchID == "" {
			writeJSON(w, map[string]string{"error": "missing batch_id"})
			return
		}
		bpVal, ok := batchProgresses.Load(batchID)
		if !ok {
			writeJSON(w, map[string]string{"error": "batch not found"})
			return
		}
		bp := bpVal.(*BatchProgress)
		type resultItem struct {
			TaskID   string             `json:"task_id"`
			FileName string             `json:"file_name"`
			Error    string             `json:"error,omitempty"`
			Summary  *excel.CalcSummary `json:"summary,omitempty"`
		}
		var results []resultItem
		for _, t := range bp.Tasks {
			ri := resultItem{TaskID: t.TaskID, FileName: t.FileName}
			if v, ok := taskResults.Load(t.TaskID); ok {
				r := v.(*app.CalcResult)
				if r.Error != "" {
					ri.Error = r.Error
				} else {
					ri.Summary = r.Summary
				}
			} else {
				ri.Error = "计算未完成"
			}
			results = append(results, ri)
		}
		writeJSON(w, results)
	})
	// 进度轮询（单文件）
	mux.HandleFunc("/api/calculate/progress", func(w http.ResponseWriter, r *http.Request) {
		taskID := r.URL.Query().Get("task_id")
		if taskID == "" {
			writeJSON(w, map[string]string{"error": "missing task_id"})
			return
		}
		val, ok := progressStore.Load(taskID)
		if !ok {
			writeJSON(w, map[string]string{"phase": "unknown", "error": "任务不存在或已过期"})
			return
		}
		writeJSON(w, val)
	})
	// 获取计算结果（单文件）
	mux.HandleFunc("/api/calculate/result", func(w http.ResponseWriter, r *http.Request) {
		taskID := r.URL.Query().Get("task_id")
		if taskID == "" {
			writeJSON(w, map[string]string{"error": "missing task_id"})
			return
		}
		val, ok := taskResults.Load(taskID)
		if !ok {
			writeJSON(w, map[string]string{"error": "任务结果不存在"})
			return
		}
		writeJSON(w, val)
	})
	// 导出 - 支持按 task_id 导出特定任务结果
	mux.HandleFunc("/api/export", func(w http.ResponseWriter, r *http.Request) {
		var result *app.CalcResult
		taskID := r.URL.Query().Get("task_id")
		if taskID != "" {
			if v, ok := taskResults.Load(taskID); ok {
				result = v.(*app.CalcResult)
			}
		}
		if result == nil {
			result = lastResult
		}
		if result == nil || len(result.Data) == 0 {
			writeJSON(w, map[string]string{"error": "没有可导出的计算结果"})
			return
		}
		// 文件名优先用原始文件名
		exportName := "结算结果"
		if result.FileName != "" {
			exportName = strings.TrimSuffix(result.FileName, ".xlsx")
			exportName = strings.TrimSuffix(exportName, ".xls")
			exportName += "_结算结果"
		}
		tmpFile := fmt.Sprintf("%s/yunfei_%s_%s.xlsx", os.TempDir(), exportName, time.Now().Format("150405"))
		err := excel.WriteResult(tmpFile, result.Data, result.Summary)
		if err != nil {
			w.WriteHeader(500)
			writeJSON(w, map[string]string{"error": "导出失败: " + err.Error()})
			return
		}
		if _, err := db.DB.Exec("UPDATE calc_history SET output_file=? WHERE id=(SELECT MAX(id) FROM calc_history)", tmpFile); err != nil {
			println("[WARN] 更新历史输出文件失败:", err.Error())
		}

		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		filename := filepath.Base(tmpFile)
		escaped := url.QueryEscape(filename)
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, filename, escaped))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", getFileSize(tmpFile)))
		http.ServeFile(w, r, tmpFile)
	})
	mux.HandleFunc("/api/history", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			var req struct{ ID int64 }
			json.NewDecoder(r.Body).Decode(&req)
			if req.ID > 0 {
				db.DB.Exec("DELETE FROM calc_history WHERE id=?", req.ID)
			}
			writeJSON(w, map[string]bool{"ok": true})
			return
		}
		writeJSON(w, a.GetHistory())
	})
	mux.HandleFunc("/api/history/clear", func(w http.ResponseWriter, r *http.Request) {
		db.DB.Exec("DELETE FROM calc_history")
		writeJSON(w, map[string]bool{"ok": true})
	})
	// 设置读写（存到本地文件）
	mux.HandleFunc("/api/settings", func(w http.ResponseWriter, r *http.Request) {
		settingsFile := filepath.Join(db.GetDataDir(), "settings.json")
		if r.Method == "GET" {
			data, err := os.ReadFile(settingsFile)
			if err != nil {
				writeJSON(w, map[string]interface{}{})
				return
			}
			var settings map[string]interface{}
			json.Unmarshal(data, &settings)
			writeJSON(w, settings)
			return
		}
		if r.Method == "POST" {
			var newSettings map[string]interface{}
			json.NewDecoder(r.Body).Decode(&newSettings)
			// 读取已有设置，合并而非覆盖（避免密码被空值覆盖）
			existing := make(map[string]interface{})
			if data, err := os.ReadFile(settingsFile); err == nil {
				json.Unmarshal(data, &existing)
			}
			for k, v := range newSettings {
				if k == "admin_pass" {
					if pass, ok := v.(string); ok && pass != "" {
						existing["admin_pass"] = pass // 只在有值时更新密码
					}
					continue
				}
				existing[k] = v
			}
			data, _ := json.MarshalIndent(existing, "", "  ")
			os.WriteFile(settingsFile, data, 0644)
			// 刷新 authSecret（修改密钥后立即生效）
			if sec, ok := existing["auth_secret"].(string); ok && sec != "" {
				authSecret = sec
			}
			writeJSON(w, map[string]bool{"ok": true})
			return
		}
	})
	// 历史记录详情（含结果数据，用于重新导出）
	mux.HandleFunc("/api/history/detail", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		id, _ := strconv.ParseInt(idStr, 10, 64)
		var h app.CalcHistory
		var outFile *string
		err := db.DB.QueryRow(`SELECT id,input_file,output_file,total_count,total_fee,avg_fee,max_fee,min_fee,rule_summary,calc_duration,created_at
			FROM calc_history WHERE id=?`, id).Scan(&h.ID, &h.InputFile, &outFile, &h.TotalCount, &h.TotalFee, &h.AvgFee, &h.MaxFee, &h.MinFee, &h.RuleSummary, &h.Duration, &h.CreatedAt)
		if err != nil {
			writeJSON(w, map[string]string{"error": "未找到记录"})
			return
		}
		if outFile != nil {
			h.OutputFile = *outFile
		}
		writeJSON(w, h)
	})

	// 静态文件（前端编译后嵌入）
	frontendFS, err := fs.Sub(frontendAssets, "frontend/dist")
	if err != nil {
		log.Println("[DEV] 前端未编译，将以 API 模式运行")
		log.Println("[DEV] 请运行: cd frontend && npm run dev (开发模式端口5173)")
	} else {
		mux.Handle("/", http.FileServer(http.FS(frontendFS)))
	}

	// CORS中间件
	handler := corsMiddleware(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "58080"
	}
	port = ":" + port

	// 单实例检测：如果已有实例在跑，直接打开浏览器并退出
	if existing, err := net.DialTimeout("tcp", "127.0.0.1"+port, 200*time.Millisecond); err == nil {
		existing.Close()
		log.Println("♻️  已有实例在运行，复用它...")
		if os.Getenv("NO_BROWSER") == "" {
			openBrowser("http://localhost" + port)
		}
		return
	}

	log.Printf("🚀 喵喵云结算启动: http://localhost%s", port)

	if os.Getenv("NO_BROWSER") == "" {
		go func() {
			openBrowser("http://localhost" + port)
		}()
	}

	log.Fatal(http.ListenAndServe(port, handler))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(200)
			return
		}
		// 白名单接口不需要认证
		if strings.HasPrefix(r.URL.Path, "/api/auth/") {
			next.ServeHTTP(w, r)
			return
		}
		// 在线授权接口不需要 token（激活前无登录态）
		if r.URL.Path == "/api/license/check-online" || r.URL.Path == "/api/license/activate-online" || r.URL.Path == "/api/license/activate-license-data" {
			next.ServeHTTP(w, r)
			return
		}
		// API 接口需要认证
		if strings.HasPrefix(r.URL.Path, "/api/") {
			token := r.Header.Get("Authorization")
			token = strings.TrimPrefix(token, "Bearer ")
			if token == "" {
				token = r.URL.Query().Get("token")
			}
			if !verifyToken(token) {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(401)
				json.NewEncoder(w).Encode(map[string]string{"error": "未登录或登录已过期"})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func openBrowser(url string) {
	switch runtime.GOOS {
	case "windows":
		openBrowserWindows(url)
	case "darwin":
		exec.Command("open", url).Start()
	default:
		exec.Command("xdg-open", url).Start()
	}
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(data)
}

func getFileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func sanitizeFilename(name string) string {
	// 去掉扩展名并清理特殊字符
	name = strings.TrimSuffix(name, ".xlsx")
	name = strings.TrimSuffix(name, ".xls")
	replacer := strings.NewReplacer(" ", "_", "(", "", ")", "", "（", "", "）", "", " ", "_")
	return replacer.Replace(name)
}

// ========== 规则初始化（从 Python Zone 体系一键导入） ==========

// zoneRates 定义各区费率 (首重1kg价格, 续重元/kg)
type zoneRate struct {
	firstPrice float64
	contPrice  float64
}

// zoneProvinceMap 省份→分区映射（与 Python ZONE_MAP 完全一致）
var zoneProvinceMap = map[string]int{
	"浙江省": 1,
	"江苏省": 2, "上海": 2, "安徽省": 2,
	"山东省": 3, "福建省": 3, "江西省": 3, "湖北省": 3,
	"湖南省": 3, "河南省": 3, "北京": 3, "天津": 3,
	"河北省": 3, "山西省": 3, "广东省": 3,
	"陕西省": 4, "重庆": 4, "四川省": 4, "辽宁省": 4,
	"吉林省": 4, "黑龙江省": 4, "云南省": 4, "贵州省": 4,
	"甘肃省": 4, "广西壮族自治区": 4, "宁夏回族自治区": 4, "海南省": 4,
	"新疆维吾尔自治区": 5, "西藏自治区": 5, "内蒙古自治区": 5, "青海省": 5,
	"香港": 6, "澳门": 6, "台湾": 6,
}

// defaultZoneRates 默认规则各区费率（市场价）
var defaultZoneRates = map[int]zoneRate{
	1: {5.0, 1.0},
	2: {6.0, 2.0},
	3: {8.0, 3.0},
	4: {10.0, 4.0},
	5: {16.0, 8.0},
	6: {25.0, 15.0},
}

// customerZoneRates 蜜丝婷大客户协议价各区费率
var mstZoneRates = map[int]zoneRate{
	1: {2.8, 0.8},
	2: {3.2, 1.2},
	3: {3.8, 1.5},
	4: {4.5, 2.0},
	5: {6.0, 3.5},
	6: {12.0, 8.0},
}

func seedDefaultRules() int {
	// 先删掉所有非 default 规则，保留初始种子
	db.DB.Exec("DELETE FROM freight_rules WHERE rule_type != 'default' OR (rule_type='default' AND province != '')")
	// 删除旧的全局默认（会被新的分省默认替代）
	db.DB.Exec("DELETE FROM freight_rules WHERE rule_type='default' AND province=''")

	count := 0
	for province, zone := range zoneProvinceMap {
		// 1. 分省默认规则（兜底）
		if dr, ok := defaultZoneRates[zone]; ok {
			db.DB.Exec(`INSERT INTO freight_rules (rule_type,customer_name,province,cont_mode,calc_mode,first_weight,first_price,cont_price,min_fee,max_fee,is_enabled,remark)
				VALUES ('default','',?,'full_kg','simple',1.0,?,?,0,0,1,?)`, province, dr.firstPrice, dr.contPrice, fmt.Sprintf("Zone%d默认", zone))
			count++
		}
		// 2. 蜜丝婷客户规则
		if cr, ok := mstZoneRates[zone]; ok {
			db.DB.Exec(`INSERT INTO freight_rules (rule_type,customer_name,province,cont_mode,calc_mode,first_weight,first_price,cont_price,min_fee,max_fee,is_enabled,remark)
				VALUES ('customer','蜜丝婷',?,'full_kg','simple',1.0,?,?,2.0,0,1,?)`, province, cr.firstPrice, cr.contPrice, fmt.Sprintf("蜜丝婷Zone%d", zone))
			count++
		}
	}
	return count
}
