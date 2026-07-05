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
	// 预加载所有拉均重规则
	customerRules, globalRule := rules.LoadAllAvgWeightRules()
	return calcAvgWeightWithRules(rowData, customerRules, globalRule)
}

// CalcAvgWeightMarkupFast 使用预加载的规则计算拉均重加价
func CalcAvgWeightMarkupFast(rowData []excel.RowData, customerRules map[string]*rules.AvgWeightRule, globalRule *rules.AvgWeightRule) ([]rules.AvgWeightResult, float64) {
	return calcAvgWeightWithRules(rowData, customerRules, globalRule)
}

func calcAvgWeightWithRules(rowData []excel.RowData, customerRules map[string]*rules.AvgWeightRule, globalRule *rules.AvgWeightRule) ([]rules.AvgWeightResult, float64) {
	// 按客户分组收集重量
	customerWeights := make(map[string][]float64)
	for _, row := range rowData {
		custKey := rules.NormalizeCustomerName(row.Customer)
		if custKey == "" {
			custKey = "默认客户"
		}
		customerWeights[custKey] = append(customerWeights[custKey], row.Weight)
	}

	var results []rules.AvgWeightResult
	totalMarkup := 0.0

	// 查找客户规则的辅助函数（客户级优先，全局兜底）
	findRule := func(customer string) *rules.AvgWeightRule {
		if customerRules != nil {
			if r, ok := customerRules[customer]; ok && r != nil {
				return r
			}
		}
		return globalRule
	}

	for customer, weights := range customerWeights {
		rule := findRule(customer)
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

	baseWeight := rule.BaseWeight
	weightLimit := rule.WeightLimit
	stepWeight := rule.StepWeight
	stepPrice := rule.StepPrice
	if stepWeight <= 0 {
		stepWeight = 0.1
	}
	if stepPrice <= 0 {
		stepPrice = 0.1
	}

	// 过滤超过重量上限的包裹（原地过滤，零额外内存分配）
	filteredIdx := 0
	for _, w := range weights {
		billW := math.Max(w, 0.01)
		if weightLimit > 0 && billW > weightLimit {
			continue
		}
		weights[filteredIdx] = billW
		filteredIdx++
	}
	filteredWeights := weights[:filteredIdx]

	if len(filteredWeights) == 0 {
		return rules.AvgWeightResult{Customer: customer}
	}

	// 计算平均重量（仅基于过滤后的包裹）
	totalWeight := 0.0
	for i := 0; i < len(filteredWeights); i++ {
		totalWeight += filteredWeights[i]
	}
	avgWeight := totalWeight / float64(len(filteredWeights))
	avgWeight = math.Round(avgWeight*1000) / 1000

	result := rules.AvgWeightResult{
		Customer:    customer,
		AvgWeight:   avgWeight,
		BaseWeight:  baseWeight,
		WeightLimit: weightLimit,
		StepPrice:   stepPrice,
		ItemCount:   len(filteredWeights),
	}

	// 平均重量 >= 基准，不加价（低于基准才加价）
	if avgWeight >= baseWeight {
		return result
	}

	// 计算偏差（超基准部分）
	deviation := avgWeight - baseWeight
	deviation = math.Round(deviation*1000) / 1000
	result.Deviation = deviation

	if deviation <= 0 {
		return result
	}

	steps := deviation / stepWeight
	switch rule.RoundMode {
	case "floor":
		steps = math.Floor(steps)
	case "round":
		steps = math.Round(steps)
	default:
		steps = math.Ceil(steps)
	}
	if steps < 0 {
		steps = 0
	}
	result.Steps = int(steps)

	perItem := steps * stepPrice
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
// 平均重量模式：只有重量在范围内（≤重量上限）的包裹参与计算并分摊加价
func ApplyAvgWeightToRows(rowData []excel.RowData, avgResults []rules.AvgWeightResult) {
	if len(avgResults) == 0 {
		return
	}

	// 构建客户 -> 单件加价和重量上限的映射
	type markupInfo struct {
		markup      float64
		weightLimit float64
	}
	markupMap := make(map[string]markupInfo)
	for _, r := range avgResults {
		custKey := rules.NormalizeCustomerName(r.Customer)
		markupMap[custKey] = markupInfo{
			markup:      r.PerItemMarkup,
			weightLimit: r.WeightLimit,
		}
	}

	for i := range rowData {
		customer := rowData[i].Customer
		custKey := rules.NormalizeCustomerName(customer)
		if custKey == "" {
			custKey = "默认客户"
		}
		if info, ok := markupMap[custKey]; ok && info.markup > 0 {
			// 超过重量上限的不加价（不参与拉均重计算）
			if info.weightLimit > 0 && rowData[i].Weight > info.weightLimit {
				continue
			}
			rowData[i].AvgWeightMarkup = info.markup
			rowData[i].Fee = math.Round((rowData[i].Fee+info.markup)*100) / 100
		}
	}
}
