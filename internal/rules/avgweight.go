package rules

import (
	"yunfei/internal/db"
)

// GetAvgWeightRules 获取所有拉均重规则
func GetAvgWeightRules() ([]AvgWeightRule, error) {
	rows, err := db.DB.Query(`SELECT id, scope_type, customer_name, base_weight, 
		weight_limit, step_weight, step_price, max_markup, round_mode, is_enabled, remark
		FROM avg_weight_rules ORDER BY scope_type, customer_name`)
	if err != nil {
		return []AvgWeightRule{}, err
	}
	defer rows.Close()
	list := make([]AvgWeightRule, 0)
	for rows.Next() {
		var r AvgWeightRule
		rows.Scan(&r.ID, &r.ScopeType, &r.CustomerName, &r.BaseWeight,
			&r.WeightLimit, &r.StepWeight, &r.StepPrice, &r.MaxMarkup, &r.RoundMode, &r.IsEnabled, &r.Remark)
		list = append(list, r)
	}
	return list, nil
}

// LoadAllAvgWeightRules 预加载所有拉均重规则（批量计算性能优化）
// 返回: 客户级规则map(客户名->规则), 全局规则(可能为nil)
func LoadAllAvgWeightRules() (map[string]*AvgWeightRule, *AvgWeightRule) {
	rows, err := db.DB.Query(`SELECT id, scope_type, customer_name, base_weight, 
		weight_limit, step_weight, step_price, max_markup, round_mode, is_enabled, remark
		FROM avg_weight_rules`)
	if err != nil {
		return nil, nil
	}
	defer rows.Close()

	customerRules := make(map[string]*AvgWeightRule)
	var globalRule *AvgWeightRule
	for rows.Next() {
		var r AvgWeightRule
		rows.Scan(&r.ID, &r.ScopeType, &r.CustomerName, &r.BaseWeight,
			&r.WeightLimit, &r.StepWeight, &r.StepPrice, &r.MaxMarkup, &r.RoundMode, &r.IsEnabled, &r.Remark)
		if r.ScopeType == "customer" && r.CustomerName != "" {
			custKey := NormalizeCustomerName(r.CustomerName)
			customerRules[custKey] = &r
		} else if r.ScopeType == "global" {
			globalRule = &r
		}
	}
	return customerRules, globalRule
}

// GetAvgWeightRuleByCustomer 获取客户的拉均重规则（优先客户级，再找全局，都没有返回nil）
func GetAvgWeightRuleByCustomer(customer string) *AvgWeightRule {
	custKey := NormalizeCustomerName(customer)
	var r AvgWeightRule
	err := db.DB.QueryRow(`SELECT id, scope_type, customer_name, base_weight, 
		weight_limit, step_weight, step_price, max_markup, round_mode, is_enabled, remark
		FROM avg_weight_rules WHERE scope_type='customer' AND customer_name=?
		LIMIT 1`, custKey).Scan(
		&r.ID, &r.ScopeType, &r.CustomerName, &r.BaseWeight,
		&r.WeightLimit, &r.StepWeight, &r.StepPrice, &r.MaxMarkup, &r.RoundMode, &r.IsEnabled, &r.Remark)
	if err == nil && r.ID > 0 {
		return &r
	}
	err = db.DB.QueryRow(`SELECT id, scope_type, customer_name, base_weight, 
		weight_limit, step_weight, step_price, max_markup, round_mode, is_enabled, remark
		FROM avg_weight_rules WHERE scope_type='global'
		LIMIT 1`).Scan(
		&r.ID, &r.ScopeType, &r.CustomerName, &r.BaseWeight,
		&r.WeightLimit, &r.StepWeight, &r.StepPrice, &r.MaxMarkup, &r.RoundMode, &r.IsEnabled, &r.Remark)
	if err == nil && r.ID > 0 {
		return &r
	}
	return nil
}

// SaveAvgWeightRule 保存拉均重规则
func SaveAvgWeightRule(r *AvgWeightRule) (int64, error) {
	r.CustomerName = NormalizeCustomerName(r.CustomerName)
	if r.ID > 0 {
		_, err := db.DB.Exec(`UPDATE avg_weight_rules SET scope_type=?, customer_name=?, 
			base_weight=?, weight_limit=?, step_weight=?, step_price=?, max_markup=?, round_mode=?, 
			is_enabled=?, remark=? WHERE id=?`,
			r.ScopeType, r.CustomerName, r.BaseWeight, r.WeightLimit, r.StepWeight, r.StepPrice,
			r.MaxMarkup, r.RoundMode, r.IsEnabled, r.Remark, r.ID)
		return r.ID, err
	}
	res, err := db.DB.Exec(`INSERT INTO avg_weight_rules 
		(scope_type, customer_name, base_weight, weight_limit, step_weight, step_price, max_markup, round_mode, is_enabled, remark)
		VALUES (?,?,?,?,?,?,?,?,?,?)`,
		r.ScopeType, r.CustomerName, r.BaseWeight, r.WeightLimit, r.StepWeight, r.StepPrice,
		r.MaxMarkup, r.RoundMode, r.IsEnabled, r.Remark)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// DeleteAvgWeightRule 删除拉均重规则
func DeleteAvgWeightRule(id int64) error {
	_, err := db.DB.Exec("DELETE FROM avg_weight_rules WHERE id=?", id)
	return err
}

// ToggleAvgWeight 切换拉均重规则启用状态
func ToggleAvgWeight(id int64, isEnabled int) error {
	_, err := db.DB.Exec("UPDATE avg_weight_rules SET is_enabled=? WHERE id=?", isEnabled, id)
	return err
}

// InitDefaultAvgWeightRule 初始化默认拉均重规则
func InitDefaultAvgWeightRule() error {
	var cnt int
	db.DB.QueryRow("SELECT COUNT(*) FROM avg_weight_rules").Scan(&cnt)
	if cnt > 0 {
		return nil
	}
	_, err := db.DB.Exec(`INSERT INTO avg_weight_rules 
		(scope_type, customer_name, base_weight, weight_limit, step_weight, step_price, max_markup, round_mode, is_enabled, remark)
		VALUES ('global', '', 0.5, 1, 0.1, 0.1, 0, 'ceil', 0, '系统默认拉均重规则（默认关闭）')`)
	return err
}
