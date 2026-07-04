package freight

import (
	"math"
	"yunfei/internal/rules"
)

// ApplyGlobalMarkup 对计算后的运费应用全局加价规则
// 返回: (最终运费, 原始运费, 加价金额)
func ApplyGlobalMarkup(fee float64, gr *rules.GlobalRule) (float64, float64, float64) {
	if gr == nil {
		return fee, 0, 0
	}
	markup := 0.0
	if gr.MarkupFixed > 0 {
		markup += gr.MarkupFixed
	}
	if gr.MarkupPercent > 0 {
		markup += math.Round(fee*gr.MarkupPercent) / 100
	}
	return math.Round((fee+markup)*100) / 100, fee, math.Round(markup*100) / 100
}

// CalcSingle 计算单笔运费（兼容旧接口）
func CalcSingle(weight float64, customer, province string, allRules []rules.FreightRule) (float64, *rules.RuleResult) {
	fee, _, _, _, best := CalcSingleWithGlobal(weight, customer, province, allRules, nil)
	return fee, best
}

// CalcSingleWithIndex 基于预建索引计算单笔运费（O(1)规则查找，批量计算用）
func CalcSingleWithIndex(weight float64, customer, province string, idx *rules.RuleIndex, gr *rules.GlobalRule, bracketMap map[int64][]rules.WeightBracket) (float64, float64, float64, float64, *rules.RuleResult) {
	best := idx.Find(customer, province)
	return doCalcSingle(weight, province, best, gr, bracketMap, nil)
}

// CalcSingleWithIndexFast 基于预建索引+预加载省份加价map计算（批量计算最快路径）
func CalcSingleWithIndexFast(weight float64, customer, province string, idx *rules.RuleIndex, gr *rules.GlobalRule, bracketMap map[int64][]rules.WeightBracket, provSurchargeMap map[string]float64) (float64, float64, float64, float64, *rules.RuleResult) {
	best := idx.Find(customer, province)
	return doCalcSingle(weight, province, best, gr, bracketMap, provSurchargeMap)
}

// CalcSingleWithGlobal 计算单笔运费（支持全局保底和加价规则）
func CalcSingleWithGlobal(weight float64, customer, province string, allRules []rules.FreightRule, gr *rules.GlobalRule) (float64, float64, float64, float64, *rules.RuleResult) {
	best := rules.FindBestRule(customer, province, allRules)
	return doCalcSingle(weight, province, best, gr, nil, nil)
}

// doCalcSingle 核心计算逻辑（共享实现）
// bracketMap: 规则ID -> 重量区间列表（批量计算时预加载，可为nil）
// provSurchargeMap: 省份 -> 加价金额（批量计算时预加载，可为nil，nil时走数据库查询）
func doCalcSingle(weight float64, province string, best *rules.RuleResult, gr *rules.GlobalRule, bracketMap map[int64][]rules.WeightBracket, provSurchargeMap map[string]float64) (float64, float64, float64, float64, *rules.RuleResult) {
	billWeight := math.Max(weight, 0.01)

	var r rules.FreightRule
	if best != nil && best.Rule.ID > 0 {
		r = best.Rule
	} else if gr != nil {
		// 无匹配规则时，使用全局保底
		r = rules.FreightRule{
			RuleType:    "fallback",
			ContMode:    "full_kg",
			CalcMode:    "simple",
			FirstWeight: gr.DefaultFirstWeight,
			FirstPrice:  gr.DefaultFirstPrice,
			ContPrice:   gr.DefaultContPrice,
			MinFee:      gr.DefaultMinFee,
		}
		if best == nil {
			best = &rules.RuleResult{Rule: r, RuleLevel: "fallback"}
		}
	} else {
		return 5.0, 0, 0, 5.0, nil
	}

	// 获取省份加价（优先用预加载map，找不到再查数据库）
	getProvSurcharge := func(p string) float64 {
		provKey := rules.NormalizeProvince(p)
		if provSurchargeMap != nil {
			if v, ok := provSurchargeMap[provKey]; ok {
				return v
			}
			return 0
		}
		return rules.GetProvinceSurcharge(provKey)
	}

	// 零重量保护：重量为正但极小时用 no_weight_price
	if weight <= 0 && gr != nil && gr.NoWeightPrice > 0 {
		baseFee := gr.NoWeightPrice
		baseFee += getProvSurcharge(province)
		finalFee, rawFee, markup := ApplyGlobalMarkup(baseFee, gr)
		return finalFee, rawFee, markup, baseFee, best
	}

	var fee float64

	// 根据计费模式选择计算方式
	if r.CalcMode == "bracket" {
		// ===== 区间计费模式 =====
		var brackets []rules.WeightBracket
		if bracketMap != nil {
			brackets = bracketMap[r.ID]
		}
		if len(brackets) == 0 && len(r.Brackets) > 0 {
			brackets = r.Brackets
		}
		fee = calcByBracket(billWeight, brackets, r)
	} else {
		// ===== 传统首重续重模式（simple，默认） =====
		fee = calcBySimple(billWeight, r)
	}

	// 偏远附加费
	fee += r.Surcharge

	// 全局省份加价（按目的省份每票加收）
	fee += getProvSurcharge(province)

	// 最低/最高限制（保底价优先于全局 min_fee）
	if r.MinFee > 0 && fee < r.MinFee {
		fee = r.MinFee
	}
	if r.MaxFee > 0 && fee > r.MaxFee {
		fee = r.MaxFee
	}

	rawFee := fee
	baseFee := fee

	// 应用全局加价
	var markup float64
	if gr != nil {
		fee, _, markup = ApplyGlobalMarkup(fee, gr)
	}

	return math.Round(fee*100) / 100, rawFee, markup, baseFee, best
}

// calcBySimple 传统首重续重计费
func calcBySimple(billWeight float64, r rules.FreightRule) float64 {
	firstW := r.FirstWeight
	if firstW <= 0 {
		firstW = 1.0
	}

	if billWeight <= firstW {
		return r.FirstPrice
	}

	excess := billWeight - firstW
	fee := r.FirstPrice
	if r.ContMode == "hundred_gram" {
		units := math.Ceil(excess * 10)
		fee += units * r.ContPrice
	} else if r.ContMode == "actual_weight" {
		fee += excess * r.ContPrice
	} else {
		units := math.Ceil(excess)
		fee += units * r.ContPrice
	}
	return fee
}

// calcByBracket 重量区间计费
func calcByBracket(billWeight float64, brackets []rules.WeightBracket, r rules.FreightRule) float64 {
	if len(brackets) == 0 {
		// 没有区间数据，降级为 simple 模式
		return calcBySimple(billWeight, r)
	}

	// 查找匹配的区间
	bracket := rules.FindBracket(billWeight, brackets)
	if bracket == nil {
		return calcBySimple(billWeight, r)
	}

	if bracket.CalcType == "fixed" {
		// 一口价
		return bracket.FixedPrice
	}

	// first_cont 首重续重模式
	firstW := bracket.FirstWeight
	if firstW <= 0 {
		firstW = bracket.WeightFrom
	}

	contMode := bracket.ContMode
	if contMode == "" {
		contMode = r.ContMode
	}

	if billWeight <= firstW {
		return bracket.FirstPrice
	}

	excess := billWeight - firstW
	fee := bracket.FirstPrice
	if contMode == "hundred_gram" {
		units := math.Ceil(excess * 10)
		fee += units * bracket.ContPrice
	} else if contMode == "actual_weight" {
		fee += excess * bracket.ContPrice
	} else {
		units := math.Ceil(excess)
		fee += units * bracket.ContPrice
	}
	return fee
}
