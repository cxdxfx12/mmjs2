package rules

import (
	"yunfei/internal/db"
)

// GlobalRule 全局保底+加价规则，单行表
type GlobalRule struct {
	DefaultFirstWeight float64 `json:"default_first_weight"`
	DefaultFirstPrice  float64 `json:"default_first_price"`
	DefaultContPrice   float64 `json:"default_cont_price"`
	DefaultMinFee      float64 `json:"default_min_fee"`
	NoWeightPrice      float64 `json:"no_weight_price"`
	MarkupFixed        float64 `json:"markup_fixed"`
	MarkupPercent      float64 `json:"markup_percent"`
}

func GetGlobalRules() *GlobalRule {
	var g GlobalRule
	err := db.DB.QueryRow(`SELECT default_first_weight,default_first_price,default_cont_price,
		default_min_fee,no_weight_price,markup_fixed,markup_percent FROM global_rules WHERE id=1`).Scan(
		&g.DefaultFirstWeight, &g.DefaultFirstPrice, &g.DefaultContPrice,
		&g.DefaultMinFee, &g.NoWeightPrice, &g.MarkupFixed, &g.MarkupPercent)
	if err != nil {
		return &GlobalRule{
			DefaultFirstWeight: 1.0,
			DefaultFirstPrice:  5.0,
			DefaultContPrice:   2.0,
			NoWeightPrice:      5.0,
		}
	}
	return &g
}

func SaveGlobalRules(g *GlobalRule) error {
	_, err := db.DB.Exec(`UPDATE global_rules SET default_first_weight=?,default_first_price=?,
		default_cont_price=?,default_min_fee=?,no_weight_price=?,markup_fixed=?,markup_percent=?,
		updated_at=datetime('now','localtime') WHERE id=1`,
		g.DefaultFirstWeight, g.DefaultFirstPrice, g.DefaultContPrice,
		g.DefaultMinFee, g.NoWeightPrice, g.MarkupFixed, g.MarkupPercent)
	return err
}

// ===== 全局省份加价 =====

// ProvinceSurcharge 省份加价规则
type ProvinceSurcharge struct {
	ID          int64   `json:"id"`
	ProvinceName string `json:"province_name"`
	Surcharge   float64 `json:"surcharge"`
	Remark      string  `json:"remark"`
}

// GetAllProvinceSurcharges 获取所有省份加价
func GetAllProvinceSurcharges() ([]ProvinceSurcharge, error) {
	rows, err := db.DB.Query(`SELECT id, province_name, surcharge, remark 
		FROM global_province_surcharges ORDER BY province_name`)
	if err != nil {
		return []ProvinceSurcharge{}, err
	}
	defer rows.Close()
	list := make([]ProvinceSurcharge, 0)
	for rows.Next() {
		var p ProvinceSurcharge
		rows.Scan(&p.ID, &p.ProvinceName, &p.Surcharge, &p.Remark)
		list = append(list, p)
	}
	return list, nil
}

// GetProvinceSurcharge 根据省份获取加价金额（找不到返回0）
func GetProvinceSurcharge(province string) float64 {
	var surcharge float64
	err := db.DB.QueryRow(`SELECT surcharge FROM global_province_surcharges 
		WHERE province_name=? LIMIT 1`, province).Scan(&surcharge)
	if err != nil {
		return 0
	}
	return surcharge
}

// SaveProvinceSurcharge 保存省份加价
func SaveProvinceSurcharge(p *ProvinceSurcharge) (int64, error) {
	if p.ID > 0 {
		_, err := db.DB.Exec(`UPDATE global_province_surcharges SET province_name=?, surcharge=?, remark=? WHERE id=?`,
			p.ProvinceName, p.Surcharge, p.Remark, p.ID)
		return p.ID, err
	}
	res, err := db.DB.Exec(`INSERT INTO global_province_surcharges (province_name, surcharge, remark) VALUES (?,?,?)`,
		p.ProvinceName, p.Surcharge, p.Remark)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// DeleteProvinceSurcharge 删除省份加价
func DeleteProvinceSurcharge(id int64) error {
	_, err := db.DB.Exec("DELETE FROM global_province_surcharges WHERE id=?", id)
	return err
}
