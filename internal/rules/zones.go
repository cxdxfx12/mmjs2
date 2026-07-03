package rules

import (
	"yunfei/internal/db"
)

// GetAllZones 获取所有区域（含省份）
func GetAllZones() ([]Zone, error) {
	rows, err := db.DB.Query(`SELECT id, zone_name, zone_order, remark 
		FROM freight_zones ORDER BY zone_order, id`)
	if err != nil {
		return []Zone{}, err
	}
	zones := make([]Zone, 0)
	for rows.Next() {
		var z Zone
		rows.Scan(&z.ID, &z.ZoneName, &z.ZoneOrder, &z.Remark)
		zones = append(zones, z)
	}
	rows.Close()

	// 加载每个区域的省份（在rows关闭后执行，避免连接死锁）
	for i := range zones {
		provinces, _ := GetZoneProvinces(zones[i].ID)
		zones[i].Provinces = provinces
	}

	return zones, nil
}

// GetZoneProvinces 获取区域的省份列表
func GetZoneProvinces(zoneID int64) ([]string, error) {
	rows, err := db.DB.Query(`SELECT province_name FROM freight_zone_provinces 
		WHERE zone_id=? ORDER BY id`, zoneID)
	if err != nil {
		return []string{}, err
	}
	defer rows.Close()
	list := make([]string, 0)
	for rows.Next() {
		var p string
		rows.Scan(&p)
		list = append(list, p)
	}
	return list, nil
}

// GetZoneByProvince 根据省份查找所在区域
func GetZoneByProvince(province string) (*Zone, error) {
	provKey := NormalizeProvince(province)
	var z Zone
	err := db.DB.QueryRow(`SELECT z.id, z.zone_name, z.zone_order, z.remark 
		FROM freight_zones z 
		INNER JOIN freight_zone_provinces zp ON z.id = zp.zone_id 
		WHERE zp.province_name=? LIMIT 1`, provKey).Scan(
		&z.ID, &z.ZoneName, &z.ZoneOrder, &z.Remark)
	if err != nil {
		return nil, err
	}
	return &z, nil
}

// GetZoneByName 根据区域名称查找区域
func GetZoneByName(zoneName string) (*Zone, error) {
	var z Zone
	err := db.DB.QueryRow(`SELECT id, zone_name, zone_order, remark 
		FROM freight_zones WHERE zone_name=? LIMIT 1`, zoneName).Scan(
		&z.ID, &z.ZoneName, &z.ZoneOrder, &z.Remark)
	if err != nil {
		return nil, err
	}
	return &z, nil
}

// SaveZone 保存区域（新增或更新）
func SaveZone(z *Zone) (int64, error) {
	if z.ID > 0 {
		_, err := db.WriteExec(`UPDATE freight_zones SET zone_name=?, zone_order=?, remark=? WHERE id=?`,
			z.ZoneName, z.ZoneOrder, z.Remark, z.ID)
		if err != nil {
			return 0, err
		}
		// 更新省份：先删后插
		db.WriteExec("DELETE FROM freight_zone_provinces WHERE zone_id=?", z.ID)
		for _, p := range z.Provinces {
			provName := NormalizeProvince(p)
			db.WriteExec("INSERT INTO freight_zone_provinces (zone_id, province_name) VALUES (?, ?)", z.ID, provName)
		}
		return z.ID, nil
	}
	res, err := db.WriteExec(`INSERT INTO freight_zones (zone_name, zone_order, remark) VALUES (?,?,?)`,
		z.ZoneName, z.ZoneOrder, z.Remark)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	for _, p := range z.Provinces {
		provName := NormalizeProvince(p)
		db.WriteExec("INSERT INTO freight_zone_provinces (zone_id, province_name) VALUES (?, ?)", id, provName)
	}
	return id, nil
}

// DeleteZone 删除区域
func DeleteZone(id int64) error {
	_, err := db.WriteExec("DELETE FROM freight_zone_provinces WHERE zone_id=?", id)
	if err != nil {
		return err
	}
	_, err = db.WriteExec("DELETE FROM freight_zones WHERE id=?", id)
	return err
}

// ZoneTemplate 区域模板（用于生成规则）
type ZoneTemplate struct {
	ZoneName   string   `json:"zone_name"`
	Provinces  []string `json:"provinces"`
	ZoneOrder  int      `json:"zone_order"`
}

// GetDefaultZoneTemplates 获取默认的6区模板（含港澳台六区）
func GetDefaultZoneTemplates() []ZoneTemplate {
	return []ZoneTemplate{
		{
			ZoneName:  "一区",
			ZoneOrder: 1,
			Provinces: []string{"江苏", "浙江", "安徽", "上海", "山东", "广东"},
		},
		{
			ZoneName:  "二区",
			ZoneOrder: 2,
			Provinces: []string{"福建", "北京", "河南", "湖北", "湖南", "江西", "天津", "河北"},
		},
		{
			ZoneName:  "三区",
			ZoneOrder: 3,
			Provinces: []string{"山西", "广西", "四川", "重庆", "陕西", "贵州", "辽宁", "吉林", "黑龙江", "云南", "海南"},
		},
		{
			ZoneName:  "四区",
			ZoneOrder: 4,
			Provinces: []string{"甘肃", "青海", "内蒙古", "宁夏"},
		},
		{
			ZoneName:  "五区",
			ZoneOrder: 5,
			Provinces: []string{"新疆", "西藏"},
		},
		{
			ZoneName:  "六区",
			ZoneOrder: 6,
			Provinces: []string{"香港", "澳门", "台湾"},
		},
	}
}

// InitDefaultZones 初始化默认6个区域（如果不存在）
func InitDefaultZones() error {
	var cnt int
	db.DB.QueryRow("SELECT COUNT(*) FROM freight_zones").Scan(&cnt)
	if cnt > 0 {
		return nil // 已有区域，不初始化
	}

	templates := GetDefaultZoneTemplates()
	for _, t := range templates {
		z := Zone{
			ZoneName:  t.ZoneName,
			ZoneOrder: t.ZoneOrder,
			Provinces: t.Provinces,
			Remark:    "系统默认区域",
		}
		SaveZone(&z)
	}
	return nil
}

// GenerateZoneRules 为指定客户按区域模板+价格表生成规则
// priceTable: key=区域名称, value=该区域的6个区间价格
// calcMode: simple | bracket（标准首重续重 / 区间计费）
// contMode: actual_weight | full_kg | hundred_gram
func GenerateZoneRules(customerName string, priceTable map[string]ZonePriceScheme, contMode string, calcMode string) ([]FreightRule, error) {
	zones, err := GetAllZones()
	if err != nil {
		return []FreightRule{}, err
	}

	result := make([]FreightRule, 0)
	for _, z := range zones {
		scheme, ok := priceTable[z.ZoneName]
		if !ok {
			continue
		}
		for _, prov := range z.Provinces {
			if calcMode == "bracket" {
				rule := FreightRule{
					RuleType:     "customer",
					CustomerName: customerName,
					Province:     prov,
					CalcMode:     "bracket",
					ContMode:     contMode,
					ZoneID:       z.ID,
					ZoneName:     z.ZoneName,
					IsEnabled:    1,
					Remark:       z.ZoneName + " - " + prov,
					Brackets:     scheme.ToBrackets(contMode),
				}
				result = append(result, rule)
			} else {
				rule := FreightRule{
					RuleType:     "customer",
					CustomerName: customerName,
					Province:     prov,
					CalcMode:     "simple",
					ContMode:     contMode,
					ZoneID:       z.ID,
					ZoneName:     z.ZoneName,
					FirstWeight:  1.0,
					FirstPrice:   scheme.Price05_1,
					ContPrice:    scheme.Cont3_30,
					IsEnabled:    1,
					Remark:       z.ZoneName + " - " + prov,
				}
				result = append(result, rule)
			}
		}
	}
	return result, nil
}

// ZonePriceScheme 一个区域的价格方案（6个区间 + 两个首重续重段）
type ZonePriceScheme struct {
	Price0_05  float64 `json:"price_0_05"`  // 0-0.5kg
	Price05_1  float64 `json:"price_05_1"`  // 0.51-1kg
	Price1_2   float64 `json:"price_1_2"`   // 1-2kg
	Price2_3   float64 `json:"price_2_3"`   // 2-3kg
	First3_30  float64 `json:"first_3_30"`  // 3-30kg 首重
	Cont3_30   float64 `json:"cont_3_30"`   // 3-30kg 续重
	First30up  float64 `json:"first_30up"`  // 30kg以上 首重
	Cont30up   float64 `json:"cont_30up"`   // 30kg以上 续重
}

// ToBrackets 将价格方案转换为重量区间数组
func (s ZonePriceScheme) ToBrackets(contMode string) []WeightBracket {
	return []WeightBracket{
		{
			WeightFrom: 0,
			WeightTo:   0.5,
			CalcType:   "fixed",
			FixedPrice: s.Price0_05,
			SortOrder:  1,
		},
		{
			WeightFrom: 0.5,
			WeightTo:   1.0,
			CalcType:   "fixed",
			FixedPrice: s.Price05_1,
			SortOrder:  2,
		},
		{
			WeightFrom: 1.0,
			WeightTo:   2.0,
			CalcType:   "fixed",
			FixedPrice: s.Price1_2,
			SortOrder:  3,
		},
		{
			WeightFrom: 2.0,
			WeightTo:   3.0,
			CalcType:   "fixed",
			FixedPrice: s.Price2_3,
			SortOrder:  4,
		},
		{
			WeightFrom:  3.0,
			WeightTo:    30.0,
			CalcType:    "first_cont",
			FirstWeight: 3.0,
			FirstPrice:  s.First3_30,
			ContPrice:   s.Cont3_30,
			ContMode:    contMode,
			SortOrder:   5,
		},
		{
			WeightFrom:  30.0,
			WeightTo:    0, // 0 = 不设上限
			CalcType:    "first_cont",
			FirstWeight: 3.0,
			FirstPrice:  s.First30up,
			ContPrice:   s.Cont30up,
			ContMode:    contMode,
			SortOrder:   6,
		},
	}
}

// GetSamplePriceTable 获取示例价格表（基于图片中的报价）
func GetSamplePriceTable() map[string]ZonePriceScheme {
	return map[string]ZonePriceScheme{
		"一区": {
			Price0_05: 2.26,
			Price05_1: 2.46,
			Price1_2:  3.56,
			Price2_3:  4.76,
			First3_30: 3.76,
			Cont3_30:  0.8,
			First30up: 3.86,
			Cont30up:  0.8,
		},
		"二区": {
			Price0_05: 2.26,
			Price05_1: 2.46,
			Price1_2:  3.56,
			Price2_3:  4.76,
			First3_30: 3.76,
			Cont3_30:  1.1,
			First30up: 4.06,
			Cont30up:  1.3,
		},
		"三区": {
			Price0_05: 2.26,
			Price05_1: 2.46,
			Price1_2:  3.56,
			Price2_3:  4.76,
			First3_30: 3.76,
			Cont3_30:  1.5,
			First30up: 4.06,
			Cont30up:  1.6,
		},
		"四区": {
			Price0_05: 2.56,
			Price05_1: 3.56,
			Price1_2:  4.06,
			Price2_3:  5.06,
			First3_30: 3.76,
			Cont3_30:  2.5,
			First30up: 4.06,
			Cont30up:  4.3,
		},
		"五区": {
			Price0_05: 10,
			Price05_1: 13,
			Price1_2:  20,
			Price2_3:  25,
			First3_30: 15,
			Cont3_30:  15,
			First30up: 15,
			Cont30up:  15,
		},
		"六区": {
			Price0_05: 25,
			Price05_1: 35,
			Price1_2:  50,
			Price2_3:  70,
			First3_30: 60,
			Cont3_30:  25,
			First30up: 60,
			Cont30up:  20,
		},
	}
}

// DefaultCustomerPreset 默认客户预设
type DefaultCustomerPreset struct {
	Name      string
	ContMode  string
	PriceMult float64
}

// GetDefaultCustomerPresets 获取5个默认客户预设
func GetDefaultCustomerPresets() []DefaultCustomerPreset {
	return []DefaultCustomerPreset{
		{Name: "中通快递", ContMode: "actual_weight", PriceMult: 1.0},
		{Name: "圆通速递", ContMode: "actual_weight", PriceMult: 0.95},
		{Name: "申通快递", ContMode: "actual_weight", PriceMult: 1.05},
		{Name: "韵达快递", ContMode: "actual_weight", PriceMult: 0.9},
		{Name: "顺丰速运", ContMode: "actual_weight", PriceMult: 1.5},
	}
}

// InitDefaultCustomers 初始化默认客户和规则（仅当没有客户时）
func InitDefaultCustomers() error {
	var cnt int
	db.DB.QueryRow(`SELECT COUNT(DISTINCT customer_name) FROM freight_rules 
		WHERE customer_name != '' AND rule_type IN ('customer','campaign')`).Scan(&cnt)
	if cnt > 0 {
		return nil
	}

	baseTable := GetSamplePriceTable()
	presets := GetDefaultCustomerPresets()

	for _, preset := range presets {
		priceTable := make(map[string]ZonePriceScheme)
		for zone, scheme := range baseTable {
			priceTable[zone] = ZonePriceScheme{
				Price0_05: round2(scheme.Price0_05 * preset.PriceMult),
				Price05_1: round2(scheme.Price05_1 * preset.PriceMult),
				Price1_2:  round2(scheme.Price1_2 * preset.PriceMult),
				Price2_3:  round2(scheme.Price2_3 * preset.PriceMult),
				First3_30: round2(scheme.First3_30 * preset.PriceMult),
				Cont3_30:  round2(scheme.Cont3_30 * preset.PriceMult),
				First30up: round2(scheme.First30up * preset.PriceMult),
				Cont30up:  round2(scheme.Cont30up * preset.PriceMult),
			}
		}

		rules, err := GenerateZoneRules(preset.Name, priceTable, preset.ContMode, "bracket")
		if err != nil {
			continue
		}
		for i := range rules {
			Save(&rules[i])
		}
	}
	return nil
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
