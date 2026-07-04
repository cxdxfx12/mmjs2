package rules

type FreightRule struct {
	ID            int64   `json:"id"`
	RuleType      string  `json:"rule_type"`
	CustomerName  string  `json:"customer_name"`
	Province      string  `json:"province"`
	ContMode      string  `json:"cont_mode"` // hundred_gram | full_kg
	FirstWeight   float64 `json:"first_weight"`
	FirstPrice    float64 `json:"first_price"`
	ContPrice     float64 `json:"cont_price"`
	MinFee        float64 `json:"min_fee"`
	MaxFee        float64 `json:"max_fee"`
	Surcharge     float64 `json:"surcharge"`
	CampaignName  string  `json:"campaign_name"`
	CampaignStart string  `json:"campaign_start"`
	CampaignEnd   string  `json:"campaign_end"`
	IsEnabled     int     `json:"is_enabled"`
	Remark        string  `json:"remark"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`

	// ===== 扩展字段 =====
	CalcMode  string          `json:"calc_mode"` // simple | bracket (简单首重续重 / 区间计费)
	ZoneID    int64           `json:"zone_id"`   // 关联区域ID，0=非区域规则
	ZoneName  string          `json:"zone_name"` // 区域名称（查询时JOIN）
	Brackets  []WeightBracket `json:"brackets,omitempty"`
}

// WeightBracket 重量区间（区间计费模式下，一个规则对应多个区间）
type WeightBracket struct {
	ID         int64   `json:"id"`
	RuleID     int64   `json:"rule_id"`
	WeightFrom float64 `json:"weight_from"` // 起始重量(kg)，包含
	WeightTo   float64 `json:"weight_to"`   // 结束重量(kg)，0表示不设上限
	CalcType   string  `json:"calc_type"`   // fixed(一口价) | first_cont(首重续重)
	FixedPrice float64 `json:"fixed_price"` // 一口价（calc_type=fixed时）
	FirstWeight float64 `json:"first_weight"` // 首重(kg)
	FirstPrice float64 `json:"first_price"`  // 首重价格
	ContPrice  float64 `json:"cont_price"`   // 续重单价
	ContMode   string  `json:"cont_mode"`    // hundred_gram | full_kg
	SortOrder  int     `json:"sort_order"`   // 排序
}

// Zone 区域（一区、二区...）
type Zone struct {
	ID        int64    `json:"id"`
	ZoneName  string   `json:"zone_name"`  // 一区/二区/...
	ZoneOrder int      `json:"zone_order"` // 排序
	Remark    string   `json:"remark"`
	Provinces []string `json:"provinces,omitempty"` // 包含的省份列表
}

// ZoneProvince 区域-省份映射
type ZoneProvince struct {
	ID           int64  `json:"id"`
	ZoneID       int64  `json:"zone_id"`
	ProvinceName string `json:"province_name"`
}

// AvgWeightRule 拉均重偏差加价规则
type AvgWeightRule struct {
	ID          int64   `json:"id"`
	ScopeType   string  `json:"scope_type"`   // global | customer
	CustomerName string `json:"customer_name"` // scope_type=customer时有效
	BaseWeight  float64 `json:"base_weight"`  // 基准重量(kg)，低于此值触发加价
	WeightLimit float64 `json:"weight_limit"` // 重量上限(kg)，超过此重量不适用拉均重，0=不限制
	StepWeight  float64 `json:"step_weight"`  // 偏差步长(kg)，每差这么多加一次价
	StepPrice   float64 `json:"step_price"`   // 每步加价(元/件)
	MaxMarkup   float64 `json:"max_markup"`   // 单件最高加价(0=不限制)
	RoundMode   string  `json:"round_mode"`   // ceil(向上取整) | floor | round
	IsEnabled   int     `json:"is_enabled"`
	Remark      string  `json:"remark"`
}

// AvgWeightResult 拉均重计算结果
type AvgWeightResult struct {
	Customer      string  `json:"customer"`
	AvgWeight     float64 `json:"avg_weight"`      // 实际平均重量
	BaseWeight    float64 `json:"base_weight"`     // 基准重量
	WeightLimit   float64 `json:"weight_limit"`    // 重量上限
	Deviation     float64 `json:"deviation"`       // 偏差 = 基准 - 实际（正数表示低于基准）
	Steps         int     `json:"steps"`           // 偏差步数
	StepPrice     float64 `json:"step_price"`      // 每步加价
	PerItemMarkup float64 `json:"per_item_markup"` // 单件加价
	ItemCount     int     `json:"item_count"`      // 包裹数
	TotalMarkup   float64 `json:"total_markup"`    // 总加价
}

type RuleResult struct {
	Rule      FreightRule `json:"rule"`
	RuleLevel string      `json:"rule_level"` // campaign / customer / global / default
}
