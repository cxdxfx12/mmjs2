package app

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"yunfei/internal/db"
	"yunfei/internal/excel"
	"yunfei/internal/freight"
	"yunfei/internal/license"
	"yunfei/internal/rules"
)

// App 应用核心
type App struct {
	ctx context.Context
}

func New() *App {
	return &App{}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	db.Init()
	// 初始化默认区域数据
	rules.InitDefaultZones()
	// 初始化默认客户和规则
	rules.InitDefaultCustomers()
}

func (a *App) Shutdown(ctx context.Context) {
	db.Close()
}

// ========== 授权相关 ==========

func (a *App) GetMachineCode() string {
	return license.GetMachineCode()
}

func (a *App) ImportLicense(b64 string) map[string]interface{} {
	ok, msg := license.ImportLicense(b64)
	return map[string]interface{}{"ok": ok, "msg": msg}
}

func (a *App) GetLicenseInfo() *license.LicenseInfo {
	return license.VerifyLicense(false)
}

// SetServerURL 设置授权服务器地址（主）
func (a *App) SetServerURL(url string) {
	license.SetServerURL(url)
}

// SetBackupServerURL 设置备用授权服务器地址
func (a *App) SetBackupServerURL(url string) {
	license.SetBackupServerURL(url)
}

// GetServerInfo 返回主备授权服务器地址
func (a *App) GetServerInfo() map[string]interface{} {
	pri, bak := license.GetServerURLs()
	return map[string]interface{}{
		"server_url":        pri,
		"backup_server_url": bak,
	}
}

// CheckOnlineLicense 在线检查授权（启动时调用）
func (a *App) CheckOnlineLicense() map[string]interface{} {
	mc := a.GetMachineCode()
	// 先检查本地缓存（7天有效）
	cached := license.GetCachedOnlineLicense(mc)
	if cached != nil && cached.Valid {
		return map[string]interface{}{
			"valid": true, "cached": true,
			"customer_name": cached.CustomerName,
			"expires_at": cached.ExpiresAt,
			"days_left": cached.DaysLeft,
		}
	}
	// 联网验证
	result := license.CheckOnlineLicense(mc)
	return map[string]interface{}{
		"valid":       result.Valid,
		"error":       result.Error,
		"cached":      false,
		"customer_name": result.CustomerName,
		"expires_at":  result.ExpiresAt,
		"days_left":   result.DaysLeft,
	}
}

// ActivateOnline 在线激活授权（旧版YF激活码）
func (a *App) ActivateOnline(licenseKey string) map[string]interface{} {
	mc := a.GetMachineCode()
	result := license.ActivateOnline(licenseKey, mc)
	return map[string]interface{}{
		"ok":   result.OK,
		"msg":  result.Msg,
		"customer_name": result.CustomerName,
		"expires_at":    result.ExpiresAt,
		"days_left":     result.DaysLeft,
	}
}

// ActivateWithLicenseData 使用加密授权数据激活（新版RSA+AES）
func (a *App) ActivateWithLicenseData(licenseData string) map[string]interface{} {
	mc := a.GetMachineCode()
	result := license.ActivateWithLicenseData(licenseData, mc)
	return map[string]interface{}{
		"ok":   result.OK,
		"msg":  result.Msg,
		"customer_name": result.CustomerName,
		"expires_at":    result.ExpiresAt,
		"days_left":     result.DaysLeft,
	}
}

// GetApiSecret 获取当前 API 密钥
func (a *App) GetApiSecret() string {
	return license.GetApiSecret()
}

// SetApiSecret 设置 API 密钥
func (a *App) SetApiSecret(secret string) {
	license.SetApiSecret(secret)
}

// ResetLicense 清除所有授权信息（恢复未激活状态）
func (a *App) ResetLicense() {
	db.DB.Exec("DELETE FROM license_info")
	db.DB.Exec("DELETE FROM app_settings WHERE key='online_license_cache'")
}

// ========== 全局规则 ==========

func (a *App) GetGlobalRules() *rules.GlobalRule {
	return rules.GetGlobalRules()
}

func (a *App) SaveGlobalRules(g *rules.GlobalRule) bool {
	return rules.SaveGlobalRules(g) == nil
}

// ========== 规则管理 ==========

func (a *App) GetRules() []rules.FreightRule {
	list, _ := rules.GetAll()
	if list == nil {
		list = []rules.FreightRule{}
	}
	return list
}

func (a *App) GetRulesByCustomer(customerName string) []rules.FreightRule {
	list, _ := rules.GetByCustomer(customerName)
	if list == nil {
		list = []rules.FreightRule{}
	}
	return list
}

type RuleSaveReq struct {
	ID            int64               `json:"id"`
	RuleType      string              `json:"rule_type"`
	CustomerName  string              `json:"customer_name"`
	Province      string              `json:"province"`
	ContMode      string              `json:"cont_mode"`
	FirstWeight   float64             `json:"first_weight"`
	FirstPrice    float64             `json:"first_price"`
	ContPrice     float64             `json:"cont_price"`
	MinFee        float64             `json:"min_fee"`
	MaxFee        float64             `json:"max_fee"`
	Surcharge     float64             `json:"surcharge"`
	CampaignName  string              `json:"campaign_name"`
	CampaignStart string              `json:"campaign_start"`
	CampaignEnd   string              `json:"campaign_end"`
	IsEnabled     int                 `json:"is_enabled"`
	Remark        string              `json:"remark"`
	CalcMode      string              `json:"calc_mode"` // simple | bracket
	ZoneID        int64               `json:"zone_id"`
	Brackets      []rules.WeightBracket `json:"brackets"`
}

func (a *App) SaveRule(r RuleSaveReq) int64 {
	rule := rules.FreightRule{
		ID:            r.ID,
		RuleType:      r.RuleType,
		CustomerName:  r.CustomerName,
		Province:      r.Province,
		ContMode:      r.ContMode,
		FirstWeight:   r.FirstWeight,
		FirstPrice:    r.FirstPrice,
		ContPrice:     r.ContPrice,
		MinFee:        r.MinFee,
		MaxFee:        r.MaxFee,
		Surcharge:     r.Surcharge,
		CampaignName:  r.CampaignName,
		CampaignStart: r.CampaignStart,
		CampaignEnd:   r.CampaignEnd,
		IsEnabled:     r.IsEnabled,
		Remark:        r.Remark,
		CalcMode:      r.CalcMode,
		ZoneID:        r.ZoneID,
		Brackets:      r.Brackets,
	}
	id, _ := rules.Save(&rule)
	return id
}

func (a *App) DeleteRule(id int64) bool {
	err := rules.Delete(id)
	return err == nil
}

func (a *App) DeleteRulesBatch(ids []int64) bool {
	err := rules.DeleteBatch(ids)
	return err == nil
}

// ========== 客户管理 ==========

func (a *App) GetCustomers() []rules.CustomerInfo {
	list, _ := rules.GetCustomers()
	if list == nil {
		list = []rules.CustomerInfo{}
	}
	return list
}

func (a *App) DeleteCustomer(name string) bool {
	err := rules.DeleteByCustomer(name)
	return err == nil
}

// CopyCustomerRules 将一个客户的全部规则复制到另一个客户
func (a *App) CopyCustomerRules(fromCustomer, toCustomer string) int {
	srcRules, err := rules.GetByCustomer(fromCustomer)
	if err != nil || len(srcRules) == 0 {
		return 0
	}
	count := 0
	for _, r := range srcRules {
		r.ID = 0            // 新记录
		r.CustomerName = toCustomer
		r.Remark = r.Remark + " (从" + fromCustomer + "复制)"
		if _, err := rules.Save(&r); err == nil {
			count++
		}
	}
	return count
}

// ImportCustomerRules 批量导入客户规则
// 简单模板（1列）: 客户名称 —— 自动用默认规则参数创建客户规则
// 新模板（14+列）: 客户名称|省份|计费模式|续重模式|首重|首重单价|续重单价|保底价|最高价|附加费|区域名称|规则类型|启用|备注
// 兼容旧模板（9列）: 客户名称|省份|续重模式|首重|首重单价|续重单价|保底价|最高价|附加费
func (a *App) ImportCustomerRules(records [][]string) (int, string) {
	if len(records) < 2 {
		return 0, "模板至少需要表头+1条数据"
	}
	count := 0
	for i, row := range records {
		if i == 0 {
			continue
		}
		if len(row) < 1 {
			continue
		}
		customerName := row[0]
		if customerName == "" {
			continue
		}

		// 只有客户名称一列 → 简单模式：用默认六区价格表生成客户规则
		if len(row) == 1 || (len(row) == 2 && row[1] == "") {
			priceTable := rules.GetSamplePriceTable()
			zoneRules, err := rules.GenerateZoneRules(customerName, priceTable, "actual_weight", "bracket")
			if err != nil {
				continue
			}
			for j := range zoneRules {
				zoneRules[j].Remark = "批量导入"
				if _, err := rules.Save(&zoneRules[j]); err == nil {
					count++
				}
			}
			continue
		}

		if len(row) < 4 {
			continue
		}

		isNewFormat := false
		if len(row) > 2 {
			col3 := row[2]
			if col3 == "simple" || col3 == "bracket" {
				isNewFormat = true
			}
		}

		var province, calcMode, contMode, ruleType, remark, zoneName string
		var firstWeight, firstPrice, contPrice, minFee, maxFee, surcharge float64
		isEnabled := 1
		var brackets []rules.WeightBracket
		var zoneID int64

		if isNewFormat {
			province = rules.NormalizeProvince(getCol(row, 1))
			calcMode = getCol(row, 2)
			if calcMode == "" {
				calcMode = "simple"
			}
			contMode = getCol(row, 3)
			if contMode == "" {
				contMode = "full_kg"
			}
			firstWeight = parseFloat(getCol(row, 4), 1.0)
			firstPrice = parseFloat(getCol(row, 5), 5.0)
			contPrice = parseFloat(getCol(row, 6), 2.0)
			minFee = parseFloat(getCol(row, 7), 0)
			maxFee = parseFloat(getCol(row, 8), 0)
			surcharge = parseFloat(getCol(row, 9), 0)
			zoneName = getCol(row, 10)
			ruleType = getCol(row, 11)
			if ruleType == "" {
				ruleType = "customer"
			}
			enableStr := getCol(row, 12)
			if enableStr == "0" {
				isEnabled = 0
			}
			remark = getCol(row, 13)
			if remark == "" {
				remark = "批量导入"
			}

			if calcMode == "bracket" {
				brackets = []rules.WeightBracket{
					{WeightFrom: 0, WeightTo: 0.5, CalcType: "fixed", FixedPrice: parseFloat(getCol(row, 14), 0), SortOrder: 1},
					{WeightFrom: 0.5, WeightTo: 1, CalcType: "fixed", FixedPrice: parseFloat(getCol(row, 15), 0), SortOrder: 2},
					{WeightFrom: 1, WeightTo: 2, CalcType: "fixed", FixedPrice: parseFloat(getCol(row, 16), 0), SortOrder: 3},
					{WeightFrom: 2, WeightTo: 3, CalcType: "fixed", FixedPrice: parseFloat(getCol(row, 17), 0), SortOrder: 4},
					{WeightFrom: 3, WeightTo: 30, CalcType: "first_cont", FirstWeight: firstWeight, FirstPrice: parseFloat(getCol(row, 18), 0), ContPrice: parseFloat(getCol(row, 19), 0), ContMode: contMode, SortOrder: 5},
					{WeightFrom: 30, WeightTo: 0, CalcType: "first_cont", FirstWeight: firstWeight, FirstPrice: parseFloat(getCol(row, 20), 0), ContPrice: parseFloat(getCol(row, 21), 0), ContMode: contMode, SortOrder: 6},
				}
			}
		} else {
			province = rules.NormalizeProvince(getCol(row, 1))
			contMode = getCol(row, 2)
			if contMode == "" {
				contMode = "full_kg"
			}
			calcMode = "simple"
			firstWeight = parseFloat(getCol(row, 3), 1.0)
			firstPrice = parseFloat(getCol(row, 4), 5.0)
			contPrice = parseFloat(getCol(row, 5), 2.0)
			minFee = parseFloat(getCol(row, 6), 0)
			maxFee = parseFloat(getCol(row, 7), 0)
			surcharge = parseFloat(getCol(row, 8), 0)
			ruleType = "customer"
			remark = "批量导入"
		}

		// 根据区域名称查找zone_id
		if zoneName != "" {
			if zone, _ := rules.GetZoneByName(zoneName); zone != nil {
				zoneID = zone.ID
			}
		}

		rule := rules.FreightRule{
			RuleType:     ruleType,
			CustomerName: customerName,
			Province:     province,
			ContMode:     contMode,
			CalcMode:     calcMode,
			FirstWeight:  firstWeight,
			FirstPrice:   firstPrice,
			ContPrice:    contPrice,
			MinFee:       minFee,
			MaxFee:       maxFee,
			Surcharge:    surcharge,
			IsEnabled:    isEnabled,
			Remark:       remark,
			Brackets:     brackets,
			ZoneID:       zoneID,
		}
		if _, err := rules.Save(&rule); err == nil {
			count++
		}
	}
	return count, ""
}

func getCol(row []string, idx int) string {
	if idx < len(row) {
		return row[idx]
	}
	return ""
}

func parseFloat(s string, def float64) float64 {
	if s == "" {
		return def
	}
	var f float64
	if _, err := fmt.Sscanf(s, "%f", &f); err == nil {
		return f
	}
	return def
}

// ========== Excel 读取 ==========

func (a *App) ReadExcelPreview(filePath string) *excel.ExcelPreview {
	// 快速预览：维度获取行数 O(1) + 采样前1000行
	preview, err := excel.ReadPreviewFast(filePath, 1000)
	if err != nil {
		return &excel.ExcelPreview{
			FileName: filePath,
			Columns:  []string{"ERROR: " + err.Error()},
		}
	}
	return preview
}

// ========== 运费计算 ==========

type CalcRequest struct {
	FilePath string `json:"file_path"`
}

type CalcResult struct {
	Data       []excel.RowData    `json:"data"`
	Summary    *excel.CalcSummary `json:"summary"`
	OutputPath string             `json:"output_path"`
	FileName   string             `json:"file_name"`
	Error      string             `json:"error,omitempty"`
}

// ProgressFn 进度回调
type ProgressFn func(phase string, current, total int, msg string)

func (a *App) CalculateFreight(req CalcRequest) *CalcResult {
	rowData, _, err := excel.ReadAllRows(req.FilePath, nil)
	if err != nil {
		return &CalcResult{Error: err.Error()}
	}
	return a.doCalc(rowData, nil, req.FilePath)
}

func (a *App) CalculateFreightWithProgress(req CalcRequest, progress ProgressFn) *CalcResult {
	// 阶段1: 读取
	progress("reading", 0, 100, "正在读取Excel文件...")
	rowData, _, err := excel.ReadAllRows(req.FilePath, func(cur, total int) {
		progress("reading", cur, total, fmt.Sprintf("读取数据中... %d 行", cur))
	})
	if err != nil {
		return &CalcResult{Error: "读取失败: " + err.Error()}
	}

	// 阶段2: 计算
	return a.doCalc(rowData, progress, req.FilePath)
}

func (a *App) doCalc(rowData []excel.RowData, progress ProgressFn, inputFile string) *CalcResult {
	startTime := time.Now()
	total := len(rowData)

	allRules, err := rules.GetAll()
	if err != nil {
		allRules = nil
	}
	gr := rules.GetGlobalRules()
	ruleIdx := rules.BuildRuleIndex(allRules, gr)

	// 预加载重量区间数据（批量计算性能优化）
	bracketMap, _ := rules.LoadRuleBrackets(allRules)

	// 预加载省份加价数据（避免每行查数据库，大幅提升性能）
	provSurchargeMap := make(map[string]float64)
	if provList, err := rules.GetAllProvinceSurcharges(); err == nil {
		for _, p := range provList {
			provKey := rules.NormalizeProvince(p.ProvinceName)
			provSurchargeMap[provKey] = p.Surcharge
		}
	}

	// 预加载拉均重规则（避免每客户查数据库）
	avgCustomerRules, avgGlobalRule := rules.LoadAllAvgWeightRules()

	numWorkers := runtime.NumCPU()
	if numWorkers < 1 {
		numWorkers = 1
	}
	if numWorkers > 16 {
		numWorkers = 16
	}
	chunkSize := (total + numWorkers - 1) / numWorkers

	if progress != nil {
		progress("calculating", 0, total, fmt.Sprintf("启动 %d 核并行计算...", numWorkers))
	}

	var wg sync.WaitGroup
	var processed atomic.Int64
	var markupCents atomic.Int64

	for w := 0; w < numWorkers; w++ {
		start := w * chunkSize
		end := start + chunkSize
		if end > total {
			end = total
		}
		if start >= end {
			continue
		}
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
				if progress != nil && cur%2000 == 0 {
					progress("calculating", int(cur), total,
						fmt.Sprintf("正在计算... %d/%d (%.0f%%)", cur, total, float64(cur)*100/float64(total)))
				}
			}
		}(start, end)
	}
	wg.Wait()

	if progress != nil {
		progress("calculating", total, total, "正在计算拉均重加价...")
	}

	// 计算并应用拉均重偏差加价（使用预加载的规则，避免重复查数据库）
	avgResults, totalAvgMarkup := freight.CalcAvgWeightMarkupFast(rowData, avgCustomerRules, avgGlobalRule)
	if len(avgResults) > 0 && totalAvgMarkup > 0 {
		freight.ApplyAvgWeightToRows(rowData, avgResults)
	}

	if progress != nil {
		progress("calculating", total, total, "正在生成统计汇总...")
	}

	duration := math.Round(time.Since(startTime).Seconds()*100) / 100
	summary := excel.BuildSummary(rowData, duration)
	tMarkup := float64(markupCents.Load()) / 100.0
	if tMarkup > 0 {
		summary.TotalMarkup = math.Round(tMarkup*100) / 100
	}
	// 拉均重加价
	summary.TotalAvgMarkup = math.Round(totalAvgMarkup*100) / 100
	if len(avgResults) > 0 {
		// 转换为 map 数组（便于 Excel 导出和 JSON 序列化）
		avgInterfaces := make([]interface{}, len(avgResults))
		for i, r := range avgResults {
			avgInterfaces[i] = map[string]interface{}{
				"customer":       r.Customer,
				"avg_weight":     r.AvgWeight,
				"base_weight":    r.BaseWeight,
				"deviation":      r.Deviation,
				"steps":          r.Steps,
				"step_price":     r.StepPrice,
				"per_item_markup": r.PerItemMarkup,
				"item_count":     r.ItemCount,
				"total_markup":   r.TotalMarkup,
			}
		}
		summary.AvgWeightResults = avgInterfaces
	}

	ruleSummary := fmt.Sprintf("%d条规则", len(allRules))
	// 取文件名（不含路径）
	fileName := filepath.Base(inputFile)
	_, err = db.WriteExec(`INSERT INTO calc_history (input_file, total_count, total_fee, avg_fee, max_fee, min_fee, rule_summary, calc_duration)
		VALUES (?,?,?,?,?,?,?,?)`,
		fileName, summary.TotalCount, summary.TotalFee, summary.AvgFee, summary.MaxFee, summary.MinFee, ruleSummary, duration)
	if err != nil {
		println("[WARN] 保存计算历史失败:", err.Error())
	}

	if progress != nil {
		progress("done", total, total, "计算完成！")
	}

	return &CalcResult{Data: rowData, Summary: summary}
}

func (a *App) ExportResult(data []excel.RowData, outputPath string, summary *excel.CalcSummary) string {
	err := excel.WriteResult(outputPath, data, summary)
	if err != nil {
		return "导出失败: " + err.Error()
	}
	db.WriteExec("UPDATE calc_history SET output_file=? WHERE id=(SELECT MAX(id) FROM calc_history)", outputPath)
	return "ok"
}

// ========== 历史记录 ==========

type CalcHistory struct {
	ID          int64   `json:"id"`
	InputFile   string  `json:"input_file"`
	OutputFile  string  `json:"output_file"`
	TotalCount  int     `json:"total_count"`
	TotalFee    float64 `json:"total_fee"`
	AvgFee      float64 `json:"avg_fee"`
	MaxFee      float64 `json:"max_fee"`
	MinFee      float64 `json:"min_fee"`
	RuleSummary string  `json:"rule_summary"`
	Duration    float64 `json:"calc_duration"`
	CreatedAt   string  `json:"created_at"`
}

func (a *App) GetHistory() []CalcHistory {
	rows, err := db.DB.Query(`SELECT id,input_file,output_file,total_count,total_fee,avg_fee,max_fee,min_fee,rule_summary,calc_duration,created_at 
		FROM calc_history ORDER BY id DESC LIMIT 50`)
	if err != nil {
		return []CalcHistory{}
	}
	defer rows.Close()
	list := []CalcHistory{}
	for rows.Next() {
		var h CalcHistory
		var outFile *string
		rows.Scan(&h.ID, &h.InputFile, &outFile, &h.TotalCount, &h.TotalFee, &h.AvgFee, &h.MaxFee, &h.MinFee, &h.RuleSummary, &h.Duration, &h.CreatedAt)
		if outFile != nil {
			h.OutputFile = *outFile
		}
		list = append(list, h)
	}
	return list
}

// ========== 区域管理 ==========

// GetZones 获取所有区域（含省份）
func (a *App) GetZones() []rules.Zone {
	zones, err := rules.GetAllZones()
	if err != nil {
		return []rules.Zone{}
	}
	return zones
}

// SaveZone 保存区域
func (a *App) SaveZone(z rules.Zone) int64 {
	id, _ := rules.SaveZone(&z)
	return id
}

// DeleteZone 删除区域
func (a *App) DeleteZone(id int64) bool {
	return rules.DeleteZone(id) == nil
}

// GetZoneTemplates 获取默认区域模板
func (a *App) GetZoneTemplates() []rules.ZoneTemplate {
	return rules.GetDefaultZoneTemplates()
}

// ========== 重量区间 ==========

// GetRuleBrackets 获取规则的重量区间
func (a *App) GetRuleBrackets(ruleID int64) []rules.WeightBracket {
	brackets, err := rules.GetBracketsByRuleID(ruleID)
	if err != nil {
		return []rules.WeightBracket{}
	}
	return brackets
}

// SaveRuleBrackets 保存规则的重量区间
func (a *App) SaveRuleBrackets(ruleID int64, brackets []rules.WeightBracket) bool {
	return rules.SaveBrackets(ruleID, brackets) == nil
}

// ========== 拉均重规则 ==========

// GetAvgWeightRules 获取所有拉均重规则
func (a *App) GetAvgWeightRules() []rules.AvgWeightRule {
	list, err := rules.GetAvgWeightRules()
	if err != nil {
		return []rules.AvgWeightRule{}
	}
	return list
}

// SaveAvgWeightRule 保存拉均重规则
func (a *App) SaveAvgWeightRule(r rules.AvgWeightRule) int64 {
	id, _ := rules.SaveAvgWeightRule(&r)
	return id
}

// DeleteAvgWeightRule 删除拉均重规则
func (a *App) DeleteAvgWeightRule(id int64) bool {
	return rules.DeleteAvgWeightRule(id) == nil
}

// ========== 区域规则模板生成 ==========

// GenerateZoneRulesByTemplate 按区域模板+示例价格表为客户生成规则
// customerName: 客户名称
// contMode: 续重模式 actual_weight | full_kg | hundred_gram
// calcMode: 定价模式 bracket | simple
// 覆盖该客户已有的区域型规则
func (a *App) GenerateZoneRulesByTemplate(customerName string, contMode string, calcMode string, priceTable map[string]rules.ZonePriceScheme) map[string]interface{} {
	if customerName == "" {
		return map[string]interface{}{"ok": false, "msg": "客户名称不能为空"}
	}
	priceMap := make(map[string]rules.ZonePriceScheme)
	if priceTable != nil && len(priceTable) > 0 {
		priceMap = priceTable
	} else {
		priceMap = rules.GetSamplePriceTable()
	}

	oldRules, _ := rules.GetByCustomer(customerName)
	for _, r := range oldRules {
		if r.ZoneID > 0 || r.CalcMode == "bracket" {
			rules.Delete(r.ID)
		}
	}

	newRules, err := rules.GenerateZoneRules(customerName, priceMap, contMode, calcMode)
	if err != nil {
		return map[string]interface{}{"ok": false, "msg": err.Error()}
	}

	count := 0
	for _, r := range newRules {
		if _, err := rules.Save(&r); err == nil {
			count++
		}
	}

	return map[string]interface{}{
		"ok":      true,
		"count":   count,
		"msg":     fmt.Sprintf("成功生成 %d 条规则", count),
	}
}

// GetSamplePriceTable 获取示例价格表
func (a *App) GetSamplePriceTable() map[string]rules.ZonePriceScheme {
	return rules.GetSamplePriceTable()
}

// ImportSamplePriceFromExcel 从 Excel 行数据导入六区参考价
func (a *App) ImportSamplePriceFromExcel(rows [][]string) (map[string]rules.ZonePriceScheme, string) {
	table, err := rules.ParsePriceTableFromRows(rows)
	if err != nil {
		return nil, err.Error()
	}
	return table, ""
}

// ========== 规则详情（含区间） ==========

// GetRuleDetail 获取规则详情（含重量区间）
func (a *App) GetRuleDetail(id int64) *rules.FreightRule {
	rule, err := rules.GetByID(id)
	if err != nil || rule == nil {
		return nil
	}
	return rule
}

// ========== 全局省份加价 ==========

func (a *App) GetProvinceSurcharges() []rules.ProvinceSurcharge {
	list, err := rules.GetAllProvinceSurcharges()
	if err != nil {
		return []rules.ProvinceSurcharge{}
	}
	return list
}

func (a *App) SaveProvinceSurcharge(p rules.ProvinceSurcharge) int64 {
	id, _ := rules.SaveProvinceSurcharge(&p)
	return id
}

func (a *App) DeleteProvinceSurcharge(id int64) bool {
	return rules.DeleteProvinceSurcharge(id) == nil
}

// ========== 规则快速测试 ==========

// TestRule 测试规则计算
// 输入客户名、省份、重量，返回匹配的规则和计算结果
func (a *App) TestRule(customer, province string, weight float64) map[string]interface{} {
	allRules, _ := rules.GetAll()
	gr := rules.GetGlobalRules()

	bracketMap, _ := rules.LoadRuleBrackets(allRules)
	idx := rules.BuildRuleIndex(allRules, gr)

	fee, rawFee, markup, baseFee, best := freight.CalcSingleWithIndex(weight, customer, province, idx, gr, bracketMap)

	result := map[string]interface{}{
		"fee":     fee,
		"raw_fee": rawFee,
		"markup":  markup,
		"base_fee": baseFee,
	}

	if best != nil {
		result["rule_level"] = best.RuleLevel
		result["rule_id"] = best.Rule.ID
		result["cont_mode"] = best.Rule.ContMode
		result["calc_mode"] = best.Rule.CalcMode
		result["zone_name"] = best.Rule.ZoneName
		result["first_weight"] = best.Rule.FirstWeight
		result["first_price"] = best.Rule.FirstPrice
		result["cont_price"] = best.Rule.ContPrice
		result["surcharge"] = best.Rule.Surcharge
		result["min_fee"] = best.Rule.MinFee
		result["max_fee"] = best.Rule.MaxFee
		result["province_surcharge"] = rules.GetProvinceSurcharge(province)
		result["global_markup_fixed"] = gr.MarkupFixed
		result["global_markup_percent"] = gr.MarkupPercent

		if best.Rule.CalcMode == "bracket" && len(best.Rule.Brackets) > 0 {
			bracketsInfo := make([]map[string]interface{}, 0)
			for _, b := range best.Rule.Brackets {
				bi := map[string]interface{}{
					"weight_from": b.WeightFrom,
					"weight_to":   b.WeightTo,
					"calc_type":   b.CalcType,
				}
				if b.CalcType == "fixed" {
					bi["fixed_price"] = b.FixedPrice
				} else {
					bi["first_weight"] = b.FirstWeight
					bi["first_price"] = b.FirstPrice
					bi["cont_price"] = b.ContPrice
					bi["cont_mode"] = b.ContMode
				}
				bracketsInfo = append(bracketsInfo, bi)
			}
			result["brackets"] = bracketsInfo
		}
	}

	avgRule := rules.GetAvgWeightRuleByCustomer(customer)
	if avgRule != nil {
		result["avg_weight_rule"] = map[string]interface{}{
			"is_enabled":    avgRule.IsEnabled,
			"base_weight":   avgRule.BaseWeight,
			"weight_limit":  avgRule.WeightLimit,
			"step_weight":   avgRule.StepWeight,
			"step_price":    avgRule.StepPrice,
			"round_mode":    avgRule.RoundMode,
			"max_markup":    avgRule.MaxMarkup,
			"scope_type":    avgRule.ScopeType,
		}
	}

	return result
}

func (a *App) TestRuleBatch(customer string, province string, weights []float64) map[string]interface{} {
	allRules, _ := rules.GetAll()
	gr := rules.GetGlobalRules()

	bracketMap, _ := rules.LoadRuleBrackets(allRules)
	idx := rules.BuildRuleIndex(allRules, gr)

	results := make([]map[string]interface{}, 0)

	if province == "" || province == "all" {
		provinces := []string{
			"北京", "天津", "上海", "重庆",
			"河北", "山西", "辽宁", "吉林", "黑龙江",
			"江苏", "浙江", "安徽", "福建", "江西", "山东",
			"河南", "湖北", "湖南", "广东",
			"四川", "贵州", "云南", "陕西", "甘肃", "青海",
			"广西", "内蒙古", "宁夏", "新疆", "西藏", "海南",
			"香港", "澳门", "台湾",
		}
		for _, p := range provinces {
			for _, w := range weights {
				fee, rawFee, markup, _, best := freight.CalcSingleWithIndex(w, customer, p, idx, gr, bracketMap)
				r := map[string]interface{}{
					"province": p,
					"weight":   w,
					"fee":      fee,
					"raw_fee":  rawFee,
					"markup":   markup,
				}
				if best != nil {
					r["rule_level"] = best.RuleLevel
					r["zone_name"] = best.Rule.ZoneName
					r["calc_mode"] = best.Rule.CalcMode
					r["cont_mode"] = best.Rule.ContMode
				}
				results = append(results, r)
			}
		}
	} else {
		for _, w := range weights {
			fee, rawFee, markup, _, best := freight.CalcSingleWithIndex(w, customer, province, idx, gr, bracketMap)
			r := map[string]interface{}{
				"province": province,
				"weight":   w,
				"fee":      fee,
				"raw_fee":  rawFee,
				"markup":   markup,
			}
			if best != nil {
				r["rule_level"] = best.RuleLevel
				r["zone_name"] = best.Rule.ZoneName
				r["calc_mode"] = best.Rule.CalcMode
				r["cont_mode"] = best.Rule.ContMode
			}
			results = append(results, r)
		}
	}

	return map[string]interface{}{
		"ok":      true,
		"count":   len(results),
		"results": results,
	}
}

// ========== 导出客户规则 ==========

// ExportCustomerRules 导出客户规则为 [][]string（供 Excel 写入）
func (a *App) ExportCustomerRules(customerName string) ([][]string, string) {
	var list []rules.FreightRule
	var err error
	if customerName != "" {
		list, err = rules.GetByCustomer(customerName)
	} else {
		list, err = rules.GetAll()
	}
	if err != nil || len(list) == 0 {
		return nil, "没有可导出的规则"
	}

	bracketMap, _ := rules.LoadRuleBrackets(list)

	headers := []string{"客户名称", "省份(空=全国)", "计费模式(simple/bracket)", "续重模式(actual_weight/full_kg/hundred_gram)",
		"首重(kg)", "首重单价(元)", "续重单价(元)", "保底价(元)", "最高价(元)", "附加费(元)",
		"区域名称", "规则类型", "启用(1/0)", "备注",
		"0-0.5kg价", "0.5-1kg价", "1-2kg价", "2-3kg价",
		"3-30kg首重", "3-30kg续重", "30kg以上首重", "30kg以上续重"}

	rows := [][]string{headers}
	for _, r := range list {
		row := []string{
			r.CustomerName,
			r.Province,
			r.CalcMode,
			r.ContMode,
			fmt.Sprintf("%g", r.FirstWeight),
			fmt.Sprintf("%g", r.FirstPrice),
			fmt.Sprintf("%g", r.ContPrice),
			fmt.Sprintf("%g", r.MinFee),
			fmt.Sprintf("%g", r.MaxFee),
			fmt.Sprintf("%g", r.Surcharge),
			r.ZoneName,
			r.RuleType,
			fmt.Sprintf("%d", r.IsEnabled),
			r.Remark,
			"", "", "", "", "", "", "", "",
		}

		if r.CalcMode == "bracket" {
			if brackets, ok := bracketMap[r.ID]; ok {
				for _, b := range brackets {
					if b.CalcType == "fixed" {
						switch {
						case b.WeightFrom == 0 && b.WeightTo == 0.5:
							row[14] = fmt.Sprintf("%g", b.FixedPrice)
						case b.WeightFrom == 0.5 && b.WeightTo == 1:
							row[15] = fmt.Sprintf("%g", b.FixedPrice)
						case b.WeightFrom == 1 && b.WeightTo == 2:
							row[16] = fmt.Sprintf("%g", b.FixedPrice)
						case b.WeightFrom == 2 && b.WeightTo == 3:
							row[17] = fmt.Sprintf("%g", b.FixedPrice)
						}
					} else if b.CalcType == "first_cont" {
						switch {
						case b.WeightFrom == 3 && b.WeightTo == 30:
							row[18] = fmt.Sprintf("%g", b.FirstPrice)
							row[19] = fmt.Sprintf("%g", b.ContPrice)
						case b.WeightFrom == 30 && b.WeightTo == 0:
							row[20] = fmt.Sprintf("%g", b.FirstPrice)
							row[21] = fmt.Sprintf("%g", b.ContPrice)
						}
					}
				}
			}
		}
		rows = append(rows, row)
	}
	return rows, ""
}
