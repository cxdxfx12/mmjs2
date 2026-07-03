package rules

import "strings"

func NormalizeProvince(province string) string {
	province = strings.TrimSpace(province)
	if strings.HasSuffix(province, "维吾尔自治区") {
		province = strings.TrimSuffix(province, "维吾尔自治区")
	} else if strings.HasSuffix(province, "回族自治区") {
		province = strings.TrimSuffix(province, "回族自治区")
	} else if strings.HasSuffix(province, "壮族自治区") {
		province = strings.TrimSuffix(province, "壮族自治区")
	} else if strings.HasSuffix(province, "自治区") {
		province = strings.TrimSuffix(province, "自治区")
	}
	province = strings.TrimSuffix(province, "省")
	province = strings.TrimSuffix(province, "市")
	province = strings.TrimSuffix(province, "特别行政区")
	province = strings.TrimSuffix(province, "地区")
	province = strings.TrimSpace(province)
	return province
}

func NormalizeCustomerName(name string) string {
	return strings.TrimSpace(name)
}
