package freight

import (
	"math"
	"yunfei/internal/excel"
	"yunfei/internal/rules"
)

// CalcAvgWeightMarkup 计算拉均重偏差加价
// 对所有行数据按客户分组，计算每组的平均重量，低于基准的按偏差步长加价
// 返回: 按客户分组的加价详情, 总加价金额
func CalcAvgWeightMarkup(rowData []excel.RowData) ([]rules.AvgWeightResult, float64) {
	// 按客户分组收集重量
	customerWeights := make(map[string][]float64)
	for _, row := range rowData {
		if row.Customer == "" {
			row.Customer = "默认客户"
		}
		customerWeights[row.Customer] = append(customerWeights[row.Customer], row.Weight)
	}

	var results []rules.AvgWeightResult
	totalMarkup := 0.0

	for customer, weights := range customerWeights {
		// 获取该客户的拉均重规则（客户级 > 全局）
		rule := rules.GetAvgWeightRuleByCustomer(customer)
		if rule == nil || rule.IsEnabled == 0 || rule.BaseWeight <= 0 {
			continue
		}

		result := calcOneCustomer(customer, weights, rule)
		results = append(results, result)
		totalMarkup += result.TotalMarkup
	}

	return results, math.Round(totalMarkup*100) / 100
}

func calcOneCustomer(customer string, weights []float64, rule *rules.AvgWeightRule) rules.AvgWeightResult {
	count := len(weights)
	if count == 0 {
		return rules.AvgWeightResult{Customer: customer}
	}

	// 过滤超过重量上限的包裹（如果设置了上限）
	filteredWeights := make([]float64, 0)
	for _, w := range weights {
		if rule.WeightLimit > 0 && w > rule.WeightLimit {
			continue
		}
		filteredWeights = append(filteredWeights, w)
	}

	if len(filteredWeights) == 0 {
		return rules.AvgWeightResult{Customer: customer}
	}

	// 计算平均重量（仅基于过滤后的包裹）
	totalWeight := 0.0
	for _, w := range filteredWeights {
		totalWeight += math.Max(w, 0.01)
	}
	avgWeight := totalWeight / float64(len(filteredWeights))
	avgWeight = math.Round(avgWeight*1000) / 1000

	result := rules.AvgWeightResult{
		Customer:     customer,
		AvgWeight:    avgWeight,
		BaseWeight:   rule.BaseWeight,
		WeightLimit:  rule.WeightLimit,
		StepPrice:    rule.StepPrice,
		ItemCount:    len(filteredWeights),
	}

	// 平均重量 >= 基准，不加价
	if avgWeight >= rule.BaseWeight {
		return result
	}

	// 计算偏差
	deviation := rule.BaseWeight - avgWeight
	deviation = math.Round(deviation*1000) / 1000
	result.Deviation = deviation

	// 计算步数
	if rule.StepWeight <= 0 {
		rule.StepWeight = 0.1
	}
	rawSteps := deviation / rule.StepWeight
	var steps int
	switch rule.RoundMode {
	case "floor":
		steps = int(math.Floor(rawSteps))
	case "round":
		steps = int(math.Round(rawSteps))
	default: // ceil
		steps = int(math.Ceil(rawSteps))
	}
	if steps < 0 {
		steps = 0
	}
	result.Steps = steps

	// 单件加价
	perItem := float64(steps) * rule.StepPrice
	if rule.MaxMarkup > 0 && perItem > rule.MaxMarkup {
		perItem = rule.MaxMarkup
	}
	perItem = math.Round(perItem*100) / 100
	result.PerItemMarkup = perItem

	// 总加价
	result.TotalMarkup = math.Round(perItem*float64(len(filteredWeights))*100) / 100

	return result
}

// ApplyAvgWeightToRows 将拉均重加价应用到每行数据上
// 直接修改 rowData 中的 Fee 和 AvgWeightMarkup 字段
// 只有重量在 weight_limit 范围内的包裹才会被加价
func ApplyAvgWeightToRows(rowData []excel.RowData, avgResults []rules.AvgWeightResult) {
	if len(avgResults) == 0 {
		return
	}

	// 构建客户 -> 单件加价和重量上限的映射
	type markupInfo struct {
		markup       float64
		weightLimit  float64
	}
	markupMap := make(map[string]markupInfo)
	for _, r := range avgResults {
		markupMap[r.Customer] = markupInfo{
			markup:       r.PerItemMarkup,
			weightLimit:  r.WeightLimit,
		}
	}

	for i := range rowData {
		customer := rowData[i].Customer
		if customer == "" {
			customer = "默认客户"
		}
		if info, ok := markupMap[customer]; ok && info.markup > 0 {
			// 检查重量是否在范围内（如果设置了上限）
			if info.weightLimit > 0 && rowData[i].Weight > info.weightLimit {
				continue
			}
			rowData[i].AvgWeightMarkup = info.markup
			rowData[i].Fee = math.Round((rowData[i].Fee + info.markup) * 100) / 100
		}
	}
}
