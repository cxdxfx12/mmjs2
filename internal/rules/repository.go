package rules

import (
	"database/sql"
	"yunfei/internal/db"
)

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
		if r.MinFee == 0 {
			r.MinFee = existing.MinFee
		}
		if r.MaxFee == 0 {
			r.MaxFee = existing.MaxFee
		}
		if r.Surcharge == 0 {
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
	for _, id := range ids {
		Delete(id)
	}
	return nil
}

// DeleteByCustomer 删除指定客户的所有规则
func DeleteByCustomer(customerName string) error {
	custKey := NormalizeCustomerName(customerName)
	// 先获取所有规则ID，再删除区间
	rows, _ := db.DB.Query("SELECT id FROM freight_rules WHERE customer_name=? AND rule_type IN ('customer','campaign')", custKey)
	var ids []int64
	for rows.Next() {
		var id int64
		rows.Scan(&id)
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		db.WriteExec("DELETE FROM freight_weight_brackets WHERE rule_id=?", id)
	}
	_, err := db.WriteExec("DELETE FROM freight_rules WHERE customer_name=? AND rule_type IN ('customer','campaign')", custKey)
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
	rows.Close()

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

// ========== RuleIndex: O(1) 规则查找 ==========
// 为批量计算预建索引，避免每行数据 O(R) 遍历

type RuleIndex struct {
	// customer -> province -> best rule（已按优先级 campaign > customer 排序）
	customerRules map[string]map[string]RuleResult
	// province -> best global rule
	globalRules map[string]RuleResult
	// 兜底默认规则
	defaultResult RuleResult
}

// BuildRuleIndex 预建规则索引（计算开始时调用一次）
func BuildRuleIndex(allRules []FreightRule, gr *GlobalRule) *RuleIndex {
	idx := &RuleIndex{
		customerRules: make(map[string]map[string]RuleResult),
		globalRules:   make(map[string]RuleResult),
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

	// 辅助函数：确保客户map存在
	ensureCustomer := func(cust string) {
		if cust == "" {
			return
		}
		custKey := NormalizeCustomerName(cust)
		if idx.customerRules[custKey] == nil {
			idx.customerRules[custKey] = make(map[string]RuleResult)
		}
	}

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
			ensureCustomer(r.CustomerName)
			custKey := NormalizeCustomerName(r.CustomerName)
			if r.Province == "" {
				idx.customerRules[custKey][""] = RuleResult{Rule: r, RuleLevel: "customer"}
			}
		}
	}
	// 再精确省份（覆盖通配）
	for _, r := range enabled {
		if r.RuleType == "customer" && r.CustomerName != "" && r.Province != "" {
			ensureCustomer(r.CustomerName)
			custKey := NormalizeCustomerName(r.CustomerName)
			provKey := NormalizeProvince(r.Province)
			idx.customerRules[custKey][provKey] = RuleResult{Rule: r, RuleLevel: "customer"}
		}
	}

	// 3) campaign 规则（最高优先级，覆盖 customer）
	// 先通配
	for _, r := range enabled {
		if r.RuleType == "campaign" && r.CustomerName != "" {
			ensureCustomer(r.CustomerName)
			custKey := NormalizeCustomerName(r.CustomerName)
			if r.Province == "" {
				idx.customerRules[custKey][""] = RuleResult{Rule: r, RuleLevel: "campaign"}
			}
		}
	}
	// 再精确省份（覆盖通配）
	for _, r := range enabled {
		if r.RuleType == "campaign" && r.CustomerName != "" && r.Province != "" {
			ensureCustomer(r.CustomerName)
			custKey := NormalizeCustomerName(r.CustomerName)
			provKey := NormalizeProvince(r.Province)
			idx.customerRules[custKey][provKey] = RuleResult{Rule: r, RuleLevel: "campaign"}
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

// Find 从索引中 O(1) 查找最佳规则
// 返回值可能为 nil（无匹配规则）
func (idx *RuleIndex) Find(customer, province string) *RuleResult {
	custKey := NormalizeCustomerName(customer)
	provKey := NormalizeProvince(province)
	// 1. 查找客户规则（含 campaign）
	if cm, ok := idx.customerRules[custKey]; ok {
		// 精确省份匹配
		if r, ok := cm[provKey]; ok {
			return &r
		}
		// 通配省份匹配
		if r, ok := cm[""]; ok {
			return &r
		}
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
