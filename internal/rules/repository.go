package rules

import (
	"database/sql"
	"fmt"
	"time"

	"yunfei/internal/db"
)

func GetDefaultRule() *FreightRule {
	var def FreightRule
	var zoneID sql.NullInt64
	var zoneName sql.NullString
	db.DB.QueryRow(`SELECT r.id,r.rule_type,r.customer_name,r.province,r.cont_mode,r.first_weight,r.first_price,r.cont_price,
		r.min_fee,r.max_fee,r.surcharge,r.campaign_name,r.campaign_start,r.campaign_end,r.is_enabled,r.remark,r.created_at,r.updated_at,
		r.calc_mode, r.zone_id, z.zone_name
		FROM freight_rules r
		LEFT JOIN freight_zones z ON r.zone_id = z.id
		WHERE r.rule_type='default' LIMIT 1`).Scan(
		&def.ID, &def.RuleType, &def.CustomerName, &def.Province, &def.ContMode,
		&def.FirstWeight, &def.FirstPrice, &def.ContPrice, &def.MinFee, &def.MaxFee, &def.Surcharge,
		&def.CampaignName, &def.CampaignStart, &def.CampaignEnd, &def.IsEnabled, &def.Remark, &def.CreatedAt, &def.UpdatedAt,
		&def.CalcMode, &zoneID, &zoneName)
	if def.CalcMode == "" {
		def.CalcMode = "simple"
		def.ContMode = "full_kg"
		def.FirstWeight = 1.0
		def.FirstPrice = 5.0
		def.ContPrice = 2.0
		def.IsEnabled = 1
	}
	return &def
}

func GetAll() ([]FreightRule, error) {
	rows, err := db.DB.Query(`SELECT r.id,r.rule_type,r.customer_name,r.province,r.cont_mode,r.first_weight,r.first_price,r.cont_price,
		r.min_fee,r.max_fee,r.surcharge,r.campaign_name,r.campaign_start,r.campaign_end,r.is_enabled,r.remark,r.created_at,r.updated_at,
		r.calc_mode, r.zone_id, z.zone_name
		FROM freight_rules r
		LEFT JOIN freight_zones z ON r.zone_id = z.id
		ORDER BY r.rule_type, r.customer_name, r.province`)
	if err != nil {
		return []FreightRule{}, err
	}
	defer rows.Close()

	list := make([]FreightRule, 0)
	for rows.Next() {
		var r FreightRule
		var zoneID sql.NullInt64
		var zoneName sql.NullString
		rows.Scan(&r.ID, &r.RuleType, &r.CustomerName, &r.Province, &r.ContMode,
			&r.FirstWeight, &r.FirstPrice, &r.ContPrice, &r.MinFee, &r.MaxFee, &r.Surcharge,
			&r.CampaignName, &r.CampaignStart, &r.CampaignEnd, &r.IsEnabled, &r.Remark, &r.CreatedAt, &r.UpdatedAt,
			&r.CalcMode, &zoneID, &zoneName)
		if zoneID.Valid {
			r.ZoneID = zoneID.Int64
		}
		if zoneName.Valid {
			r.ZoneName = zoneName.String
		}
		if r.CalcMode == "" {
			r.CalcMode = "simple"
		}
		list = append(list, r)
	}
	return list, nil
}

func GetByID(id int64) (*FreightRule, error) {
	var r FreightRule
	var zoneID sql.NullInt64
	var zoneName sql.NullString
	err := db.DB.QueryRow(`SELECT r.id,r.rule_type,r.customer_name,r.province,r.cont_mode,r.first_weight,r.first_price,r.cont_price,
		r.min_fee,r.max_fee,r.surcharge,r.campaign_name,r.campaign_start,r.campaign_end,r.is_enabled,r.remark,r.created_at,r.updated_at,
		r.calc_mode, r.zone_id, z.zone_name
		FROM freight_rules r
		LEFT JOIN freight_zones z ON r.zone_id = z.id
		WHERE r.id=?`, id).Scan(
		&r.ID, &r.RuleType, &r.CustomerName, &r.Province, &r.ContMode,
		&r.FirstWeight, &r.FirstPrice, &r.ContPrice, &r.MinFee, &r.MaxFee, &r.Surcharge,
		&r.CampaignName, &r.CampaignStart, &r.CampaignEnd, &r.IsEnabled, &r.Remark, &r.CreatedAt, &r.UpdatedAt,
		&r.CalcMode, &zoneID, &zoneName)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if zoneID.Valid {
		r.ZoneID = zoneID.Int64
	}
	if zoneName.Valid {
		r.ZoneName = zoneName.String
	}
	if r.CalcMode == "" {
		r.CalcMode = "simple"
	}
	// 加载重量区间
	if r.CalcMode == "bracket" {
		brackets, _ := GetBracketsByRuleID(r.ID)
		r.Brackets = brackets
	}
	return &r, err
}

func Save(r *FreightRule) (int64, error) {
	r.Province = NormalizeProvince(r.Province)
	r.CustomerName = NormalizeCustomerName(r.CustomerName)
	// 活动规则校验：活动名称与日期必须存在，且结束日期不早于开始日期
	if r.RuleType == "campaign" {
		if r.CampaignName == "" {
			return 0, fmt.Errorf("campaign_name required for campaign rules")
		}
		if r.CampaignStart == "" || r.CampaignEnd == "" {
			return 0, fmt.Errorf("campaign_start and campaign_end are required for campaign rules")
		}
		// 尝试解析多种可能的日期格式，最终比较日期部分
		parseDate := func(s string) (time.Time, error) {
			if t, err := time.Parse("2006-01-02", s); err == nil {
				return t, nil
			}
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				return t, nil
			}
			if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
				return t, nil
			}
			return time.Time{}, fmt.Errorf("invalid date format: %s", s)
		}
		sd, err1 := parseDate(r.CampaignStart)
		ed, err2 := parseDate(r.CampaignEnd)
		if err1 != nil || err2 != nil {
			return 0, fmt.Errorf("invalid campaign date: %v %v", err1, err2)
		}
		// 只比较日期部分
		y1, m1, d1 := sd.Date()
		y2, m2, d2 := ed.Date()
		ds := time.Date(y1, m1, d1, 0, 0, 0, 0, time.Local)
		de := time.Date(y2, m2, d2, 0, 0, 0, 0, time.Local)
		if de.Before(ds) {
			return 0, fmt.Errorf("campaign_end before campaign_start")
		}
		// 规范化为 YYYY-MM-DD 存储
		r.CampaignStart = ds.Format("2006-01-02")
		r.CampaignEnd = de.Format("2006-01-02")
	}
	if r.ID > 0 {
		var existing FreightRule
		db.DB.QueryRow(`SELECT rule_type, customer_name, province, cont_mode, first_weight, first_price,
		cont_price, min_fee, max_fee, surcharge, campaign_name, campaign_start, campaign_end,
		is_enabled, remark, calc_mode, zone_id FROM freight_rules WHERE id=?`, r.ID).Scan(
			&existing.RuleType, &existing.CustomerName, &existing.Province, &existing.ContMode,
			&existing.FirstWeight, &existing.FirstPrice, &existing.ContPrice, &existing.MinFee,
			&existing.MaxFee, &existing.Surcharge, &existing.CampaignName, &existing.CampaignStart,
			&existing.CampaignEnd, &existing.IsEnabled, &existing.Remark, &existing.CalcMode, &existing.ZoneID)

		if r.ContMode == "" {
			r.ContMode = existing.ContMode
		}
		if r.CalcMode == "" {
			r.CalcMode = existing.CalcMode
		}
		if r.ZoneID == 0 {
			r.ZoneID = existing.ZoneID
		}
		if r.RuleType == "" {
			r.RuleType = existing.RuleType
		}
		if r.CustomerName == "" {
			r.CustomerName = existing.CustomerName
		}
		if r.FirstWeight == 0 {
			r.FirstWeight = existing.FirstWeight
		}
		if r.FirstPrice == 0 {
			r.FirstPrice = existing.FirstPrice
		}
		if r.ContPrice == 0 {
			r.ContPrice = existing.ContPrice
		}
		// 使用 -1 作为"未设置"标记，允许将字段重置为 0
		if r.MinFee < 0 {
			r.MinFee = existing.MinFee
		}
		if r.MaxFee < 0 {
			r.MaxFee = existing.MaxFee
		}
		if r.Surcharge < 0 {
			r.Surcharge = existing.Surcharge
		}
		if r.CampaignName == "" {
			r.CampaignName = existing.CampaignName
		}
		if r.CampaignStart == "" {
			r.CampaignStart = existing.CampaignStart
		}
		if r.CampaignEnd == "" {
			r.CampaignEnd = existing.CampaignEnd
		}
		if r.Remark == "" {
			r.Remark = existing.Remark
		}

		_, err := db.WriteExec(`UPDATE freight_rules SET rule_type=?,customer_name=?,province=?,cont_mode=?,
			first_weight=?,first_price=?,cont_price=?,min_fee=?,max_fee=?,surcharge=?,
			campaign_name=?,campaign_start=?,campaign_end=?,is_enabled=?,remark=?,
			calc_mode=?, zone_id=?,
			updated_at=datetime('now','localtime') WHERE id=?`,
			r.RuleType, r.CustomerName, r.Province, r.ContMode,
			r.FirstWeight, r.FirstPrice, r.ContPrice, r.MinFee, r.MaxFee, r.Surcharge,
			r.CampaignName, r.CampaignStart, r.CampaignEnd, r.IsEnabled, r.Remark,
			r.CalcMode, r.ZoneID, r.ID)
		if err != nil {
			return 0, err
		}
		if r.CalcMode == "bracket" && len(r.Brackets) > 0 {
			SaveBrackets(r.ID, r.Brackets)
		} else if existing.CalcMode == "bracket" && r.CalcMode != "bracket" {
			// 从 bracket 切换到 simple 时，清除旧的重量区间数据
			db.WriteExec("DELETE FROM freight_weight_brackets WHERE rule_id=?", r.ID)
		}
		return r.ID, err
	}
	if r.CalcMode == "" {
		r.CalcMode = "simple"
	}
	res, err := db.WriteExec(`INSERT INTO freight_rules (rule_type,customer_name,province,cont_mode,
		first_weight,first_price,cont_price,min_fee,max_fee,surcharge,
		campaign_name,campaign_start,campaign_end,is_enabled,remark,calc_mode,zone_id)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		r.RuleType, r.CustomerName, r.Province, r.ContMode,
		r.FirstWeight, r.FirstPrice, r.ContPrice, r.MinFee, r.MaxFee, r.Surcharge,
		r.CampaignName, r.CampaignStart, r.CampaignEnd, r.IsEnabled, r.Remark,
		r.CalcMode, r.ZoneID)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	r.ID = id
	// 保存重量区间
	if r.CalcMode == "bracket" && len(r.Brackets) > 0 {
		SaveBrackets(id, r.Brackets)
	}
	return id, nil
}

func Delete(id int64) error {
	// 不允许删除默认规则
	var rt string
	db.DB.QueryRow("SELECT rule_type FROM freight_rules WHERE id=?", id).Scan(&rt)
	if rt == "default" {
		return nil
	}
	// 先删重量区间
	_, err := db.WriteExec("DELETE FROM freight_weight_brackets WHERE rule_id=?", id)
	if err != nil {
		return err
	}
	_, err = db.WriteExec("DELETE FROM freight_rules WHERE id=?", id)
	return err
}

// DeleteBatch 批量删除规则
func DeleteBatch(ids []int64) error {
	var lastErr error
	for _, id := range ids {
		if err := Delete(id); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// DeleteByCustomer 删除指定客户的所有规则
func DeleteByCustomer(customerName string) error {
	custKey := NormalizeCustomerName(customerName)
	// 先获取所有规则ID，再删除区间
	rows, err := db.DB.Query("SELECT id FROM freight_rules WHERE customer_name=? AND rule_type IN ('customer','campaign')", custKey)
	if err != nil {
		return err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}
	for _, id := range ids {
		db.WriteExec("DELETE FROM freight_weight_brackets WHERE rule_id=?", id)
	}
	_, err = db.WriteExec("DELETE FROM freight_rules WHERE customer_name=? AND rule_type IN ('customer','campaign')", custKey)
	return err
}

// GetByCustomer 获取指定客户的所有规则
func GetByCustomer(customerName string) ([]FreightRule, error) {
	custKey := NormalizeCustomerName(customerName)
	rows, err := db.DB.Query(`SELECT r.id,r.rule_type,r.customer_name,r.province,r.cont_mode,r.first_weight,r.first_price,r.cont_price,
		r.min_fee,r.max_fee,r.surcharge,r.campaign_name,r.campaign_start,r.campaign_end,r.is_enabled,r.remark,r.created_at,r.updated_at,
		r.calc_mode, r.zone_id, z.zone_name
		FROM freight_rules r
		LEFT JOIN freight_zones z ON r.zone_id = z.id
		WHERE r.customer_name=? AND r.rule_type IN ('customer','campaign')
		ORDER BY r.province`, custKey)
	if err != nil {
		return []FreightRule{}, err
	}
	defer rows.Close()

	list := make([]FreightRule, 0)
	for rows.Next() {
		var r FreightRule
		var zoneID sql.NullInt64
		var zoneName sql.NullString
		if err := rows.Scan(&r.ID, &r.RuleType, &r.CustomerName, &r.Province, &r.ContMode,
			&r.FirstWeight, &r.FirstPrice, &r.ContPrice, &r.MinFee, &r.MaxFee, &r.Surcharge,
			&r.CampaignName, &r.CampaignStart, &r.CampaignEnd, &r.IsEnabled, &r.Remark, &r.CreatedAt, &r.UpdatedAt,
			&r.CalcMode, &zoneID, &zoneName); err != nil {
			continue
		}
		if zoneID.Valid {
			r.ZoneID = zoneID.Int64
		}
		if zoneName.Valid {
			r.ZoneName = zoneName.String
		}
		if r.CalcMode == "" {
			r.CalcMode = "simple"
		}
		list = append(list, r)
	}

	// 批量加载bracket规则的重量区间（在rows关闭后执行，避免连接死锁）
	bracketMap, _ := LoadRuleBrackets(list)
	for i := range list {
		if list[i].CalcMode == "bracket" {
			list[i].Brackets = bracketMap[list[i].ID]
		}
	}

	return list, nil
}

// CustomerInfo 客户信息
type CustomerInfo struct {
	Name      string `json:"name"`
	RuleCount int    `json:"rule_count"`
	CalcMode  string `json:"calc_mode"` // 主要计费模式
}

// GetCustomers 获取所有客户（从规则表中去重）
func GetCustomers() ([]CustomerInfo, error) {
	rows, err := db.DB.Query(`SELECT customer_name, COUNT(*) as cnt 
		FROM freight_rules 
		WHERE customer_name != '' AND rule_type IN ('customer','campaign')
		GROUP BY customer_name ORDER BY customer_name`)
	if err != nil {
		return []CustomerInfo{}, err
	}
	defer rows.Close()
	list := make([]CustomerInfo, 0)
	for rows.Next() {
		var c CustomerInfo
		rows.Scan(&c.Name, &c.RuleCount)
		list = append(list, c)
	}
	return list, nil
}

// SaveBatch 批量保存规则（导入用）
func SaveBatch(rules []FreightRule) (int, error) {
	count := 0
	for _, r := range rules {
		_, err := Save(&r)
		if err == nil {
			count++
		}
	}
	return count, nil
}

// FindBestRule 按优先级查找最匹配的规则
// 优先级：campaign > customer > global > default
// 同级别内：精确省份 > 通配省份（空=全国）
func FindBestRule(customer, province string, allRules []FreightRule) *RuleResult {
	if allRules == nil {
		allRules, _ = GetAll()
	}
	// 过滤启用的规则
	var enabled []FreightRule
	for _, r := range allRules {
		if r.IsEnabled == 1 {
			enabled = append(enabled, r)
		}
	}

	custKey := NormalizeCustomerName(customer)
	provKey := NormalizeProvince(province)

	// 辅助函数：在指定级别的规则中查找，先精确省份再通配省份
	findInLevel := func(ruleType string) *RuleResult {
		// 第一轮：精确省份匹配
		for _, r := range enabled {
			if r.RuleType == ruleType {
				if ruleType != "global" && NormalizeCustomerName(r.CustomerName) != custKey {
					continue
				}
				if ruleType == "campaign" && !isCampaignActive(r) {
					continue
				}
				if r.Province != "" && NormalizeProvince(r.Province) == provKey {
					return &RuleResult{Rule: r, RuleLevel: ruleType}
				}
			}
		}
		// 第二轮：通配省份匹配（空=全国通用）
		for _, r := range enabled {
			if r.RuleType == ruleType {
				if ruleType != "global" && NormalizeCustomerName(r.CustomerName) != custKey {
					continue
				}
				if ruleType == "campaign" && !isCampaignActive(r) {
					continue
				}
				if r.Province == "" {
					return &RuleResult{Rule: r, RuleLevel: ruleType}
				}
			}
		}
		return nil
	}

	// 1. 查找活动规则（最高优先级）
	if r := findInLevel("campaign"); r != nil {
		return r
	}

	// 2. 查找客户规则
	if r := findInLevel("customer"); r != nil {
		return r
	}

	// 3. 查找全局规则
	if r := findInLevel("global"); r != nil {
		return r
	}

	// 4. 兜底默认规则
	var def FreightRule
	var zoneID sql.NullInt64
	var zoneName sql.NullString
	db.DB.QueryRow(`SELECT r.id,r.rule_type,r.customer_name,r.province,r.cont_mode,r.first_weight,r.first_price,r.cont_price,
		r.min_fee,r.max_fee,r.surcharge,r.campaign_name,r.campaign_start,r.campaign_end,r.is_enabled,r.remark,r.created_at,r.updated_at,
		r.calc_mode, r.zone_id, z.zone_name
		FROM freight_rules r
		LEFT JOIN freight_zones z ON r.zone_id = z.id
		WHERE r.rule_type='default' LIMIT 1`).Scan(
		&def.ID, &def.RuleType, &def.CustomerName, &def.Province, &def.ContMode,
		&def.FirstWeight, &def.FirstPrice, &def.ContPrice, &def.MinFee, &def.MaxFee, &def.Surcharge,
		&def.CampaignName, &def.CampaignStart, &def.CampaignEnd, &def.IsEnabled, &def.Remark, &def.CreatedAt, &def.UpdatedAt,
		&def.CalcMode, &zoneID, &zoneName)
	if def.CalcMode == "" {
		def.CalcMode = "simple"
	}
	return &RuleResult{Rule: def, RuleLevel: "default"}
}

func matchProvince(ruleProv, targetProv string) bool {
	if ruleProv == "" {
		return true // 空=匹配所有
	}
	return ruleProv == targetProv
}

func isCampaignActive(r FreightRule) bool {
	now := time.Now()
	if r.CampaignStart != "" {
		start, err := time.ParseInLocation("2006-01-02", r.CampaignStart, time.Local)
		if err == nil && now.Before(start) {
			return false
		}
	}
	if r.CampaignEnd != "" {
		end, err := time.ParseInLocation("2006-01-02", r.CampaignEnd, time.Local)
		if err == nil {
			end = end.Add(24 * time.Hour)
			if now.After(end) {
				return false
			}
		}
	}
	return true
}

// ========== RuleIndex: O(1) 规则查找 ==========
// 为批量计算预建索引，避免每行数据 O(R) 遍历

type RuleIndex struct {
	// 扁平化存储："客户|省份" -> RuleResult（减少 map 层级，提升查找效率）
	flatRules map[string]RuleResult
	// province -> best global rule
	globalRules map[string]RuleResult
	// 兜底默认规则
	defaultResult RuleResult
}

// makeIndexKey 创建组合键（避免字符串拼接分配）
func makeIndexKey(customer, province string) string {
	// 预分配足够长度：客户名 + "|" + 省份名
	n := len(customer) + 1 + len(province)
	if n <= 128 {
		// 小键使用栈分配
		var buf [128]byte
		copy(buf[:], customer)
		buf[len(customer)] = '|'
		copy(buf[len(customer)+1:], province)
		return string(buf[:n])
	}
	// 大键使用堆分配
	buf := make([]byte, n)
	copy(buf, customer)
	buf[len(customer)] = '|'
	copy(buf[len(customer)+1:], province)
	return string(buf)
}

// BuildRuleIndex 预建规则索引（计算开始时调用一次）
func BuildRuleIndex(allRules []FreightRule) *RuleIndex {
	idx := &RuleIndex{
		flatRules:   make(map[string]RuleResult),
		globalRules: make(map[string]RuleResult),
	}

	// 收集所有启用的规则
	var enabled []FreightRule
	for _, r := range allRules {
		if r.IsEnabled == 1 {
			enabled = append(enabled, r)
		}
	}

	// 优先级：campaign > customer > global，高优先级覆盖低优先级
	// 同级别内：精确省份 > 通配省份（空=全国）
	// 策略：先加载通配（低优先级），再加载精确（高优先级，覆盖通配）

	// 1) global 规则（最低优先级）
	// 先通配
	for _, r := range enabled {
		if r.RuleType == "global" && r.Province == "" {
			idx.globalRules[""] = RuleResult{Rule: r, RuleLevel: "global"}
		}
	}
	// 再精确省份（覆盖通配）
	for _, r := range enabled {
		if r.RuleType == "global" && r.Province != "" {
			provKey := NormalizeProvince(r.Province)
			idx.globalRules[provKey] = RuleResult{Rule: r, RuleLevel: "global"}
		}
	}

	// 2) customer 规则（覆盖 global）
	// 先通配
	for _, r := range enabled {
		if r.RuleType == "customer" && r.CustomerName != "" {
			custKey := NormalizeCustomerName(r.CustomerName)
			if r.Province == "" {
				idx.flatRules[makeIndexKey(custKey, "")] = RuleResult{Rule: r, RuleLevel: "customer"}
			}
		}
	}
	// 再精确省份（覆盖通配）
	for _, r := range enabled {
		if r.RuleType == "customer" && r.CustomerName != "" && r.Province != "" {
			custKey := NormalizeCustomerName(r.CustomerName)
			provKey := NormalizeProvince(r.Province)
			idx.flatRules[makeIndexKey(custKey, provKey)] = RuleResult{Rule: r, RuleLevel: "customer"}
		}
	}

	// 3) campaign 规则（最高优先级，覆盖 customer）
	// 先通配
	for _, r := range enabled {
		if r.RuleType == "campaign" && r.CustomerName != "" {
			if !isCampaignActive(r) {
				continue
			}
			custKey := NormalizeCustomerName(r.CustomerName)
			if r.Province == "" {
				idx.flatRules[makeIndexKey(custKey, "")] = RuleResult{Rule: r, RuleLevel: "campaign"}
			}
		}
	}
	// 再精确省份（覆盖通配）
	for _, r := range enabled {
		if r.RuleType == "campaign" && r.CustomerName != "" && r.Province != "" {
			if !isCampaignActive(r) {
				continue
			}
			custKey := NormalizeCustomerName(r.CustomerName)
			provKey := NormalizeProvince(r.Province)
			idx.flatRules[makeIndexKey(custKey, provKey)] = RuleResult{Rule: r, RuleLevel: "campaign"}
		}
	}

	// 兜底默认规则
	var def FreightRule
	var zoneID sql.NullInt64
	var zoneName sql.NullString
	db.DB.QueryRow(`SELECT r.id,r.rule_type,r.customer_name,r.province,r.cont_mode,r.first_weight,r.first_price,r.cont_price,
		r.min_fee,r.max_fee,r.surcharge,r.campaign_name,r.campaign_start,r.campaign_end,r.is_enabled,r.remark,r.created_at,r.updated_at,
		r.calc_mode, r.zone_id, z.zone_name
		FROM freight_rules r
		LEFT JOIN freight_zones z ON r.zone_id = z.id
		WHERE r.rule_type='default' LIMIT 1`).Scan(
		&def.ID, &def.RuleType, &def.CustomerName, &def.Province, &def.ContMode,
		&def.FirstWeight, &def.FirstPrice, &def.ContPrice, &def.MinFee, &def.MaxFee, &def.Surcharge,
		&def.CampaignName, &def.CampaignStart, &def.CampaignEnd, &def.IsEnabled, &def.Remark, &def.CreatedAt, &def.UpdatedAt,
		&def.CalcMode, &zoneID, &zoneName)
	if def.CalcMode == "" {
		def.CalcMode = "simple"
	}
	idx.defaultResult = RuleResult{Rule: def, RuleLevel: "default"}

	return idx
}

// Find 从索引中 O(1) 查找最佳规则（使用预计算键版本更快）
// 如果 RowData 中有预计算的 CustKey/ProvKey，请使用 FindByKeys
func (idx *RuleIndex) Find(customer, province string) *RuleResult {
	custKey := NormalizeCustomerName(customer)
	provKey := NormalizeProvince(province)
	return idx.FindByKeys(custKey, provKey)
}

// FindByKeys 使用预计算的归一化键查找（零字符串分配，最高性能）
func (idx *RuleIndex) FindByKeys(custKey, provKey string) *RuleResult {
	// 1. 查找客户规则（含 campaign）- 单次 map 查找
	if r, ok := idx.flatRules[makeIndexKey(custKey, provKey)]; ok {
		return &r
	}
	// 回退到通配省份
	if r, ok := idx.flatRules[makeIndexKey(custKey, "")]; ok {
		return &r
	}
	// 2. 查找全局规则
	if r, ok := idx.globalRules[provKey]; ok {
		return &r
	}
	if r, ok := idx.globalRules[""]; ok {
		return &r
	}
	// 3. 兜底默认
	if idx.defaultResult.Rule.ID > 0 {
		return &idx.defaultResult
	}
	return nil
}
