package rules

import (
	"yunfei/internal/db"
)

// GetBracketsByRuleID 获取规则的所有重量区间
func GetBracketsByRuleID(ruleID int64) ([]WeightBracket, error) {
	rows, err := db.DB.Query(`SELECT id, rule_id, weight_from, weight_to, calc_type, 
		fixed_price, first_weight, first_price, cont_price, cont_mode, sort_order
		FROM freight_weight_brackets WHERE rule_id=? ORDER BY sort_order, weight_from`, ruleID)
	if err != nil {
		return []WeightBracket{}, err
	}
	defer rows.Close()
	list := make([]WeightBracket, 0)
	for rows.Next() {
		var b WeightBracket
		rows.Scan(&b.ID, &b.RuleID, &b.WeightFrom, &b.WeightTo, &b.CalcType,
			&b.FixedPrice, &b.FirstWeight, &b.FirstPrice, &b.ContPrice, &b.ContMode, &b.SortOrder)
		list = append(list, b)
	}
	return list, nil
}

// SaveBrackets 保存规则的重量区间（先删后插）
func SaveBrackets(ruleID int64, brackets []WeightBracket) error {
	_, err := db.WriteExec("DELETE FROM freight_weight_brackets WHERE rule_id=?", ruleID)
	if err != nil {
		return err
	}
	for _, b := range brackets {
		_, err = db.WriteExec(`INSERT INTO freight_weight_brackets 
			(rule_id, weight_from, weight_to, calc_type, fixed_price, first_weight, first_price, cont_price, cont_mode, sort_order)
			VALUES (?,?,?,?,?,?,?,?,?,?)`,
			ruleID, b.WeightFrom, b.WeightTo, b.CalcType, b.FixedPrice,
			b.FirstWeight, b.FirstPrice, b.ContPrice, b.ContMode, b.SortOrder)
		if err != nil {
			return err
		}
	}
	return nil
}

// FindBracket 从区间列表中查找匹配的重量区间
func FindBracket(weight float64, brackets []WeightBracket) *WeightBracket {
	if len(brackets) == 0 {
		return nil
	}
	for _, b := range brackets {
		if weight >= b.WeightFrom {
			if b.WeightTo <= 0 || weight < b.WeightTo {
				return &b
			}
		}
	}
	// 重量小于所有区间起始值，返回第一个区间（最小的）
	if weight < brackets[0].WeightFrom {
		return &brackets[0]
	}
	// 重量大于所有区间（最后一个区间没有上限的情况不会走到这里），返回最后一个
	return &brackets[len(brackets)-1]
}

// LoadRuleBrackets 批量加载规则的区间数据（用于批量计算前预加载）
func LoadRuleBrackets(rules []FreightRule) (map[int64][]WeightBracket, error) {
	result := make(map[int64][]WeightBracket)
	if len(rules) == 0 {
		return result, nil
	}

	// 收集所有rule_id
	ids := make([]int64, 0, len(rules))
	for _, r := range rules {
		if r.CalcMode == "bracket" && r.ID > 0 {
			ids = append(ids, r.ID)
		}
	}
	if len(ids) == 0 {
		return result, nil
	}

	// 构造 IN 查询
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := `SELECT id, rule_id, weight_from, weight_to, calc_type, 
		fixed_price, first_weight, first_price, cont_price, cont_mode, sort_order
		FROM freight_weight_brackets WHERE rule_id IN (` + joinStrings(placeholders, ",") + `)
		ORDER BY rule_id, sort_order`

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var b WeightBracket
		rows.Scan(&b.ID, &b.RuleID, &b.WeightFrom, &b.WeightTo, &b.CalcType,
			&b.FixedPrice, &b.FirstWeight, &b.FirstPrice, &b.ContPrice, &b.ContMode, &b.SortOrder)
		result[b.RuleID] = append(result[b.RuleID], b)
	}
	return result, nil
}

func joinStrings(arr []string, sep string) string {
	result := ""
	for i, s := range arr {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
