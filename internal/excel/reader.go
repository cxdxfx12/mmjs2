package excel

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// RowData 一行结算数据
type RowData struct {
	BusinessTime   string  `json:"business_time"`
	WaybillNo      string  `json:"waybill_no"`
	Weight         float64 `json:"weight"`
	Province       string  `json:"province"`
	VolWeight      float64 `json:"vol_weight"`
	Station        string  `json:"station"`
	PackageStation string  `json:"package_station"`
	Customer       string  `json:"customer"`
	Fee            float64 `json:"fee"`
	RuleLevel      string  `json:"rule_level"`
	ContMode       string  `json:"cont_mode"`
	CalcMode       string  `json:"calc_mode"`       // simple | bracket
	ZoneName       string  `json:"zone_name"`         // 所属区域名称
	AvgWeightMarkup float64 `json:"avg_weight_markup"` // 拉均重加价
}

// ExcelPreview Excel预览信息
type ExcelPreview struct {
	FileName  string     `json:"file_name"`
	TotalRows int        `json:"total_rows"`
	Columns   []string   `json:"columns"`
	Customers []string   `json:"customers"`
	Provinces []string   `json:"provinces"`
	Samples   [][]string `json:"samples"`
}

// ReadPreview 读取Excel文件预览
func ReadPreview(filePath string) (*ExcelPreview, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("无法打开文件: %v", err)
	}
	defer f.Close()

	sheet := f.GetSheetName(0)
	rows, err := f.Rows(sheet)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 必须先 Next() 才能获取第一行列数据（excelize 流式读取特性）
	if !rows.Next() {
		return nil, fmt.Errorf("excel 文件为空")
	}
	header, _ := rows.Columns()
	colMap := detectColumns(header)

	preview := &ExcelPreview{
		FileName: filePath,
		Columns:  header,
	}

	rowCount := 0
	customerSet := map[string]bool{}
	provSet := map[string]bool{}
	var samples [][]string

	// 表头行也作为样本第一行展示
	if rowCount < 10 {
		samples = append(samples, header)
	}
	if c := getCol(header, colMap["customer"]); c != "" {
		customerSet[c] = true
	}
	if p := getCol(header, colMap["province"]); p != "" {
		provSet[p] = true
	}
	rowCount++

	for rows.Next() {
		row, _ := rows.Columns()
		if rowCount < 10 {
			samples = append(samples, row)
		}
		if c := getCol(row, colMap["customer"]); c != "" {
			customerSet[c] = true
		}
		if p := getCol(row, colMap["province"]); p != "" {
			provSet[p] = true
		}
		rowCount++
	}

	preview.TotalRows = rowCount
	preview.Samples = samples
	for c := range customerSet {
		preview.Customers = append(preview.Customers, c)
	}
	for p := range provSet {
		preview.Provinces = append(preview.Provinces, p)
	}
	return preview, nil
}

func detectColumns(header []string) map[string]int {
	m := make(map[string]int)
	for i, h := range header {
		hLower := strings.ToLower(h)
		// 已设置过的高优先级列不覆盖
		switch {
		case strContains(h, "业务时间", "date"):
			// 高优先级：精确匹配业务时间
			if _, ok := m["date"]; !ok || strings.Contains(h, "业务时间") {
				m["date"] = i
			}
		case strContains(h, "时间", "日期", "time"):
			if _, ok := m["date"]; !ok {
				m["date"] = i
			}
		case strContains(h, "运单", "单号", "waybill"):
			m["waybill"] = i
		case strContains(h, "结算重量", "计费重量", "重量", "weight"):
			m["weight"] = i
		case strContains(h, "目的省"):
			// 高优先级：目的省份 > 签收省份 > 省份
			m["province"] = i
		case strings.Contains(hLower, "province"):
			if _, ok := m["province"]; !ok {
				m["province"] = i
			}
		case strContains(h, "省份", "签收省"):
			// 低优先级：只有在没有目的省份时才设置
			if _, ok := m["province"]; !ok {
				m["province"] = i
			}
		case strContains(hLower, "体积重", "体积", "vol"):
			m["vol_weight"] = i
		case strContains(h, "订单/面单网点", "订单网点", "面单网点", "station"):
			m["station"] = i
		case strContains(h, "集包网点", "集包", "package"):
			m["package_station"] = i
		case strContains(h, "客户"):
			// 精确匹配"客户"列，优先级高于"订单客户"
			if h == "客户" || strings.TrimSpace(h) == "客户" {
				m["customer"] = i
			} else if _, ok := m["customer"]; !ok {
				m["customer"] = i
			}
		case strContains(hLower, "customer", "client"):
			if _, ok := m["customer"]; !ok {
				m["customer"] = i
			}
		}
	}
	return m
}

func strContains(s string, keys ...string) bool {
	for _, k := range keys {
		if strings.Contains(s, k) {
			return true
		}
	}
	return false
}

func getCol(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

// normalizeProvince 标准化省份名称（去掉"省"、"市"、"自治区"等后缀）
func normalizeProvince(province string) string {
	province = strings.TrimSpace(province)
	// 先处理长后缀（避免被短后缀截断）
	if strings.HasSuffix(province, "维吾尔自治区") {
		province = strings.TrimSuffix(province, "维吾尔自治区")
	} else if strings.HasSuffix(province, "回族自治区") {
		province = strings.TrimSuffix(province, "回族自治区")
	} else if strings.HasSuffix(province, "壮族自治区") {
		province = strings.TrimSuffix(province, "壮族自治区")
	} else if strings.HasSuffix(province, "自治区") {
		province = strings.TrimSuffix(province, "自治区")
	}
	// 然后处理短后缀
	province = strings.TrimSuffix(province, "省")
	province = strings.TrimSuffix(province, "市")
	province = strings.TrimSuffix(province, "特别行政区")
	province = strings.TrimSuffix(province, "地区")
	province = strings.TrimSpace(province)
	return province
}

func getColFloat(row []string, idx int) float64 {
	s := getCol(row, idx)
	if s == "" {
		return 0
	}
	var v float64
	fmt.Sscanf(s, "%f", &v)
	return v
}

func getColDate(row []string, idx int) string {
	s := getCol(row, idx)
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

// ReadPreviewFast 快速预览（直接读 ZIP 原始 XML，不经过 excelize，大文件秒开）
// 大文件场景避免 excelize.OpenFile 解析全部共享字符串表导致的数分钟延迟
func ReadPreviewFast(filePath string, maxSample int) (*ExcelPreview, error) {
	zr, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("无法打开文件: %v", err)
	}
	defer zr.Close()

	// 找到第一个 sheet 文件路径
	sheetPath, sharedStringsPath := findSheetPaths(zr)
	if sheetPath == "" {
		return nil, fmt.Errorf("找不到工作表")
	}

	// Step 1: 从 sheet XML 读维度（regex，不解析 XML）
	totalRows := readSheetDimensionRaw(zr, sheetPath)

	// Step 2: 流式读取前 maxSample 行，收集需要的共享字符串索引
	neededSST := map[int]bool{}
	headers, headerRaw, rawRows := readFirstNRowsRaw(zr, sheetPath, maxSample, neededSST)

	if len(headers) == 0 {
		return nil, fmt.Errorf("excel 文件为空")
	}

	// Step 3: 只加载需要的共享字符串（流式，不全量解析）
	sst := loadNeededSST(zr, sharedStringsPath, neededSST)

	// Step 4: 用 SST 重新解析表头（之前 raw 解析是 SST 索引值，无法匹配中文列名）
	var resolvedHeaders []string
	if len(headerRaw) > 0 {
		resolvedHeaders = resolveRowRaw(headerRaw, sst)
	} else {
		resolvedHeaders = headers
	}
	colMap := detectColumns(resolvedHeaders)
	previewColumns := resolvedHeaders // 用于展示和后续计算

	var samples [][]string
	samples = append(samples, previewColumns)

	customerSet := map[string]bool{}
	provSet := map[string]bool{}

	if c := getCol(previewColumns, colMap["customer"]); c != "" {
		customerSet[c] = true
	}
	if p := getCol(previewColumns, colMap["province"]); p != "" {
		provSet[p] = true
	}

	for _, raw := range rawRows {
		row := resolveRow(raw, sst)
		samples = append(samples, row)
		if c := getCol(row, colMap["customer"]); c != "" {
			customerSet[c] = true
		}
		if p := getCol(row, colMap["province"]); p != "" {
			provSet[p] = true
		}
	}

	preview := &ExcelPreview{
		FileName:  filePath,
		Columns:   headers,
		TotalRows: totalRows,
		Samples:   samples,
	}
	for c := range customerSet {
		preview.Customers = append(preview.Customers, c)
	}
	for p := range provSet {
		preview.Provinces = append(preview.Provinces, p)
	}

	return preview, nil
}

// ========== 以下为 ZIP 直读快速预览内部工具函数 ==========

type rawCell struct {
	ref string // A1, B2...
	t   string // cell type: "s"=string, ""=number, "str"=inline string
	v   string // cell value
}

// findSheetPaths 从 workbook.xml 和 rels 找到第一个 sheet 的路径和 sharedStrings 路径
func findSheetPaths(zr *zip.ReadCloser) (sheetPath, sstPath string) {
	type workbookXML struct {
		Sheets []struct {
			Name    string `xml:"name,attr"`
			RID     string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
		} `xml:"sheets>sheet"`
	}

	// 读取 workbook.xml 获取第一个 sheet 的 rId
	var wb workbookXML
	decodeZipXML(zr, "xl/workbook.xml", &wb)
	if len(wb.Sheets) == 0 {
		return "", ""
	}
	rID := wb.Sheets[0].RID

	// 读取 workbook.xml.rels 获取对应路径
	type relsXML struct {
		Relationships []struct {
			ID     string `xml:"Id,attr"`
			Target string `xml:"Target,attr"`
		} `xml:"Relationship"`
	}
	var rels relsXML
	decodeZipXML(zr, "xl/_rels/workbook.xml.rels", &rels)
	for _, r := range rels.Relationships {
		if r.ID == rID {
			sheetPath = "xl/" + r.Target
			break
		}
	}

	// sharedStrings
	// 尝试打开 xl/sharedStrings.xml，存在就用
	if _, err := findZipFile(zr, "xl/sharedStrings.xml"); err == nil {
		sstPath = "xl/sharedStrings.xml"
	}

	return
}

// decodeZipXML 从 ZIP 中读取一个 XML 文件并解码到结构体
func decodeZipXML(zr *zip.ReadCloser, name string, v interface{}) {
	f, err := findZipFile(zr, name)
	if err != nil {
		return
	}
	defer f.Close()
	xml.NewDecoder(f).Decode(v)
}

// findZipFile 在 ZIP 中查找文件
func findZipFile(zr *zip.ReadCloser, name string) (io.ReadCloser, error) {
	for _, f := range zr.File {
		if f.Name == name {
			return f.Open()
		}
	}
	return nil, fmt.Errorf("not found: %s", name)
}

// readSheetDimensionRaw 用 regex 从原始 sheet XML 中提取 <dimension ref="A1:Z500000"/>
// 不解析整个 XML，仅用 buf 扫描
func readSheetDimensionRaw(zr *zip.ReadCloser, sheetPath string) int {
	f, err := findZipFile(zr, sheetPath)
	if err != nil {
		return 0
	}
	defer f.Close()

	// 只读取前 4KB（dimension 元素通常在文件开头附近）
	buf := make([]byte, 4096)
	n, _ := io.ReadFull(f, buf)
	if n == 0 {
		return 0
	}

	re := regexp.MustCompile(`<dimension[^>]*ref="([^"]+)"`)
	m := re.FindSubmatch(buf[:n])
	if len(m) < 2 {
		return 0
	}

	ref := string(m[1])
	// ref 格式: "A1:K500000" → 取最后一组数字
	numRe := regexp.MustCompile(`\d+$`)
	if nm := numRe.FindString(ref); nm != "" {
		n, _ := strconv.Atoi(nm)
		return n
	}
	return 0
}

// readFirstNRowsRaw 流式解析 sheet XML，读取前 N 行的原始 cell 数据
// 返回: resolvedHeaders(用nil SST解析的表头), headerRaw(原始cells供后续SST解析), dataRows
func readFirstNRowsRaw(zr *zip.ReadCloser, sheetPath string, maxRows int, neededSST map[int]bool) (headers []string, headerRaw []rawCell, rows [][]rawCell) {
	f, err := findZipFile(zr, sheetPath)
	if err != nil {
		return nil, nil, nil
	}
	defer f.Close()

	decoder := xml.NewDecoder(f)

	type sheetDataState int
	const (
		stateRoot      sheetDataState = iota
		stateSheetData                // 在 <sheetData> 内
		stateRow                      // 在 <row> 内
		stateCell                     // 在 <c> 内
	)

	state := stateRoot
	var currentRow []rawCell
	var currentCell rawCell
	var rowCount int
	var charBuf []byte

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}

		switch t := token.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "sheetData":
				state = stateSheetData
			case "row":
				if state == stateSheetData {
					state = stateRow
					currentRow = nil
					rowCount++
				}
			case "c":
				if state == stateRow {
					state = stateCell
					currentCell = rawCell{}
					for _, attr := range t.Attr {
						switch attr.Name.Local {
						case "r":
							currentCell.ref = attr.Value
						case "t":
							currentCell.t = attr.Value
						}
					}
					charBuf = nil
				}
			case "v":
				if state == stateCell {
					charBuf = nil
				}
			}

		case xml.CharData:
			if state == stateCell {
				charBuf = append(charBuf, t...)
			}

		case xml.EndElement:
			switch t.Name.Local {
			case "v":
				if state == stateCell {
					currentCell.v = string(charBuf)
				}
			case "c":
				if state == stateCell {
					currentRow = append(currentRow, currentCell)
					// 收集 SST 索引
					if currentCell.t == "s" {
						if idx, err := strconv.Atoi(currentCell.v); err == nil {
							neededSST[idx] = true
						}
					}
					state = stateRow
				}
			case "row":
				if state == stateRow {
				if rowCount == 1 {
					headers = resolveRowRaw(currentRow, nil)
					headerRaw = currentRow // 保存原始 cells，后续用 SST 重新解析
				} else {
						rows = append(rows, currentRow)
						if rowCount > maxRows {
							return // 采样够了，提前退出
						}
					}
					state = stateSheetData
				}
			case "sheetData":
				return // 读完 sheetData
			}
		}
	}

	return
}

// resolveRowRaw 将原始 cell 列表解析为字符串数组（不查 SST，t="s" 保留原始索引值）
func resolveRowRaw(cells []rawCell, sst []string) []string {
	result := make([]string, len(cells))
	for i, c := range cells {
		if c.t == "s" && sst != nil {
			if idx, err := strconv.Atoi(c.v); err == nil && idx >= 0 && idx < len(sst) {
				result[i] = sst[idx]
			} else {
				result[i] = c.v
			}
		} else if c.t == "str" {
			result[i] = c.v
		} else if c.t == "inlineStr" {
			result[i] = c.v
		} else {
			// 数字或空类型，直接用 v
			result[i] = c.v
		}
	}
	return result
}

// resolveRow 用 SST 解析 raw cells
func resolveRow(cells []rawCell, sst []string) []string {
	return resolveRowRaw(cells, sst)
}

// loadNeededSST 流式解析 sharedStrings.xml，只加载 neededSST 中需要的字符串
// 避免全量加载数百万共享字符串到内存
func loadNeededSST(zr *zip.ReadCloser, sstPath string, neededSST map[int]bool) []string {
	if sstPath == "" || len(neededSST) == 0 {
		return nil
	}

	f, err := findZipFile(zr, sstPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	decoder := xml.NewDecoder(f)
	var result []string
	var inT bool
	var currentString string
	var sstIdx int
	var inSI bool

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}

		switch t := token.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "si":
				inSI = true
			case "t":
				if inSI {
					inT = true
					currentString = ""
				}
			case "r":
				// 富文本模式，跳过 r 元素（实际上也是 t 子元素）
			}

		case xml.CharData:
			if inT {
				currentString += string(t)
			}

		case xml.EndElement:
			switch t.Name.Local {
			case "t":
				inT = false
			case "si":
				inSI = false
				// 只保存我们需要的 SST 条目
				if neededSST[sstIdx] {
					// 扩展 result 到足够大小（按需）
					for len(result) <= sstIdx {
						result = append(result, "")
					}
					result[sstIdx] = currentString
				}
				sstIdx++
			}
		}
	}

	return result
}

// ProgressCallback 进度回调
type ProgressCallback func(current, total int)

// ReadAllRows 流式读取所有行
func ReadAllRows(filePath string, progress ProgressCallback) ([]RowData, string, error) {
	return ReadAllRowsWithProgress(filePath, progress)
}

// safeColumns 安全读取一行列数据，避免 excelize 库在处理损坏共享字符串索引时 panic 导致整个进程崩溃
func safeColumns(rows *excelize.Rows) []string {
	var cols []string
	func() {
		defer func() {
			recover()
		}()
		cols, _ = rows.Columns()
	}()
	return cols
}

// ReadAllRowsWithProgress 流式读取所有行（带进度回调）
func ReadAllRowsWithProgress(filePath string, progress ProgressCallback) ([]RowData, string, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()

	sheet := f.GetSheetName(0)
	rows, err := f.Rows(sheet)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	// 必须先 Next() 才能获取第一行列数据（excelize 流式读取特性）
	if !rows.Next() {
		return nil, "", fmt.Errorf("excel 文件为空")
	}
	header := safeColumns(rows)
	colMap := detectColumns(header)

	var data []RowData
	count := 0
	for rows.Next() {
		row := safeColumns(rows)
		if row == nil {
			count++
			continue
		}
		weight := getColFloat(row, colMap["weight"])
		volWeight := getColFloat(row, colMap["vol_weight"])
		billWeight := math.Max(weight, volWeight)
		if billWeight <= 0 {
			billWeight = 0.01
		}

		rd := RowData{
			BusinessTime:   getColDate(row, colMap["date"]),
			WaybillNo:      getCol(row, colMap["waybill"]),
			Weight:         billWeight,
			Province:       normalizeProvince(getCol(row, colMap["province"])), // 标准化省份名称
			VolWeight:      volWeight,
			Station:        getCol(row, colMap["station"]),
			PackageStation: getCol(row, colMap["package_station"]),
			Customer:       getCol(row, colMap["customer"]),
		}
		data = append(data, rd)
		count++
		// 每1000行报告一次进度
		if progress != nil && count%1000 == 0 {
			progress(count, 0)
		}
	}
	return data, sheet, nil
}

// WriteResult 写入结算结果到 xlsx
func WriteResult(outputPath string, data []RowData, summary *CalcSummary) error {
	f := excelize.NewFile()
	sheetName := "结算结果"
	f.SetSheetName("Sheet1", sheetName)

	sw, err := f.NewStreamWriter(sheetName)
	if err != nil {
		return err
	}

	// 表头
	header := []interface{}{"业务时间", "运单号", "结算重量(kg)", "目的省份", "体积重(kg)", "订单/面单网点", "集包网点", "客户",
		"运费(元)", "拉均重加价(元)", "规则级别", "续重模式", "计费模式", "所属区域"}
	sw.SetRow("A1", header)

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#4472C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	f.SetCellStyle(sheetName, "A1", "N1", headerStyle)

	for i, d := range data {
		row := i + 2
		cell, _ := excelize.CoordinatesToCellName(1, row)
		sw.SetRow(cell, []interface{}{
			d.BusinessTime, d.WaybillNo, d.Weight, d.Province, d.VolWeight,
			d.Station, d.PackageStation, d.Customer, d.Fee, d.AvgWeightMarkup,
			d.RuleLevel, d.ContMode, d.CalcMode, d.ZoneName,
		})
	}
	if err := sw.Flush(); err != nil {
		return err
	}

	// 汇总 Sheet
	if summary != nil {
		writeSummarySheet(f, summary)
	}

	// 列宽
	colWidths := map[string]float64{"A": 14, "B": 20, "C": 14, "D": 10, "E": 12, "F": 20, "G": 18, "H": 14,
		"I": 12, "J": 14, "K": 10, "L": 10, "M": 10, "N": 10}
	for col, w := range colWidths {
		f.SetColWidth(sheetName, col, col, w)
	}

	return f.SaveAs(outputPath)
}

// CalcSummary 计算摘要
type CalcSummary struct {
	TotalCount        int                `json:"total_count"`
	TotalFee          float64            `json:"total_fee"`
	AvgFee            float64            `json:"avg_fee"`
	MaxFee            float64            `json:"max_fee"`
	MinFee            float64            `json:"min_fee"`
	TotalMarkup       float64            `json:"total_markup"`
	TotalAvgMarkup    float64            `json:"total_avg_markup"`   // 拉均重总加价
	AvgWeightResults  []interface{}      `json:"avg_weight_results"` // 拉均重计算结果
	ByProvince        map[string]ProvSum `json:"by_province"`
	ByCustomer        map[string]CustSum `json:"by_customer"`
	ByRuleLevel       map[string]RLSum   `json:"by_rule_level"`
	ByZone            map[string]ZoneSum `json:"by_zone"`            // 按区域汇总
	ByCalcMode        map[string]ModeSum `json:"by_calc_mode"`       // 按计费模式汇总
	DurationSec       float64            `json:"duration_sec"`
}

type ProvSum struct {
	Count int     `json:"count"`
	Fee   float64 `json:"fee"`
}
type CustSum struct {
	Count int     `json:"count"`
	Fee   float64 `json:"fee"`
}
type RLSum struct {
	Count int     `json:"count"`
	Fee   float64 `json:"fee"`
}
type ZoneSum struct {
	Count int     `json:"count"`
	Fee   float64 `json:"fee"`
}
type ModeSum struct {
	Count int     `json:"count"`
	Fee   float64 `json:"fee"`
}

func writeSummarySheet(f *excelize.File, s *CalcSummary) {
	sheet := "汇总统计"
	f.NewSheet(sheet)
	set := func(cell string, v interface{}) { f.SetCellValue(sheet, cell, v) }

	set("A1", "指标"); set("B1", "数值")
	set("A2", "总件数"); set("B2", s.TotalCount)
	set("A3", "总运费"); set("B3", s.TotalFee)
	set("A4", "平均运费"); set("B4", s.AvgFee)
	set("A5", "最高运费"); set("B5", s.MaxFee)
	set("A6", "最低运费"); set("B6", s.MinFee)
	set("A7", "拉均重加价合计"); set("B7", s.TotalAvgMarkup)
	set("A8", "计算耗时(秒)"); set("B8", s.DurationSec)

	set("D1", "省份"); set("E1", "件数"); set("F1", "运费")
	row := 2
	for prov, ps := range s.ByProvince {
		set(fmt.Sprintf("D%d", row), prov)
		set(fmt.Sprintf("E%d", row), ps.Count)
		set(fmt.Sprintf("F%d", row), ps.Fee)
		row++
	}

	set("H1", "客户"); set("I1", "件数"); set("J1", "运费")
	row = 2
	for cust, cs := range s.ByCustomer {
		set(fmt.Sprintf("H%d", row), cust)
		set(fmt.Sprintf("I%d", row), cs.Count)
		set(fmt.Sprintf("J%d", row), cs.Fee)
		row++
	}

	// 区域汇总
	set("L1", "区域"); set("M1", "件数"); set("N1", "运费")
	row = 2
	for zone, zs := range s.ByZone {
		set(fmt.Sprintf("L%d", row), zone)
		set(fmt.Sprintf("M%d", row), zs.Count)
		set(fmt.Sprintf("N%d", row), zs.Fee)
		row++
	}

	// 计费模式汇总
	set("P1", "计费模式"); set("Q1", "件数"); set("R1", "运费")
	row = 2
	for mode, ms := range s.ByCalcMode {
		modeName := mode
		if mode == "simple" {
			modeName = "传统首重续重"
		} else if mode == "bracket" {
			modeName = "重量区间计费"
		}
		set(fmt.Sprintf("P%d", row), modeName)
		set(fmt.Sprintf("Q%d", row), ms.Count)
		set(fmt.Sprintf("R%d", row), ms.Fee)
		row++
	}

	// 拉均重明细
	if len(s.AvgWeightResults) > 0 {
		set("T1", "拉均重明细")
		set("T2", "客户"); set("U2", "平均重量"); set("V2", "基准重量"); set("W2", "偏差")
		set("X2", "偏差步数"); set("Y2", "单件加价"); set("Z2", "件数"); set("AA2", "总加价")
		for i, r := range s.AvgWeightResults {
			row = i + 3
			// 通过 fmt 提取字段（兼容不同类型）
			if m, ok := r.(map[string]interface{}); ok {
				set(fmt.Sprintf("T%d", row), m["customer"])
				set(fmt.Sprintf("U%d", row), m["avg_weight"])
				set(fmt.Sprintf("V%d", row), m["base_weight"])
				set(fmt.Sprintf("W%d", row), m["deviation"])
				set(fmt.Sprintf("X%d", row), m["steps"])
				set(fmt.Sprintf("Y%d", row), m["per_item_markup"])
				set(fmt.Sprintf("Z%d", row), m["item_count"])
				set(fmt.Sprintf("AA%d", row), m["total_markup"])
			} else {
				// 结构体类型，通过 %v 格式化（兜底方案）
				set(fmt.Sprintf("T%d", row), fmt.Sprintf("%v", r))
			}
		}
	}

	f.SetColWidth(sheet, "A", "B", 18)
	f.SetColWidth(sheet, "D", "F", 12)
	f.SetColWidth(sheet, "H", "J", 14)
	f.SetColWidth(sheet, "L", "N", 12)
	f.SetColWidth(sheet, "P", "R", 14)
	f.SetColWidth(sheet, "T", "AA", 12)
}

func BuildSummary(data []RowData, durationSec float64) *CalcSummary {
	s := &CalcSummary{
		TotalCount:  len(data),
		ByProvince:  make(map[string]ProvSum),
		ByCustomer:  make(map[string]CustSum),
		ByRuleLevel: make(map[string]RLSum),
		ByZone:      make(map[string]ZoneSum),
		ByCalcMode:  make(map[string]ModeSum),
		DurationSec: durationSec,
	}
	if len(data) == 0 {
		return s
	}
	s.MinFee = data[0].Fee
	for _, d := range data {
		s.TotalFee += d.Fee
		if d.Fee > s.MaxFee {
			s.MaxFee = d.Fee
		}
		if d.Fee < s.MinFee {
			s.MinFee = d.Fee
		}
		if d.AvgWeightMarkup > 0 {
			s.TotalAvgMarkup += d.AvgWeightMarkup
		}

		ps := s.ByProvince[d.Province]
		ps.Count++
		ps.Fee += d.Fee
		s.ByProvince[d.Province] = ps

		cs := s.ByCustomer[d.Customer]
		cs.Count++
		cs.Fee += d.Fee
		s.ByCustomer[d.Customer] = cs

		rl := s.ByRuleLevel[d.RuleLevel]
		rl.Count++
		rl.Fee += d.Fee
		s.ByRuleLevel[d.RuleLevel] = rl

		zone := d.ZoneName
		if zone == "" {
			zone = "未分区"
		}
		zs := s.ByZone[zone]
		zs.Count++
		zs.Fee += d.Fee
		s.ByZone[zone] = zs

		cm := d.CalcMode
		if cm == "" {
			cm = "simple"
		}
		ms := s.ByCalcMode[cm]
		ms.Count++
		ms.Fee += d.Fee
		s.ByCalcMode[cm] = ms
	}
	s.TotalFee = math.Round(s.TotalFee*100) / 100
	s.TotalAvgMarkup = math.Round(s.TotalAvgMarkup*100) / 100
	s.AvgFee = math.Round(s.TotalFee/float64(s.TotalCount)*100) / 100
	return s
}


