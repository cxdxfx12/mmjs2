//go:build ignore

package main

import (
	"fmt"
	"os"

	"github.com/xuri/excelize/v2"
)

func main() {
	path := `C:\Users\ccc\AppData\Local\Temp\yunfei_uploads\蜜丝婷-4月发件账单表1.xlsx`
	f, err := excelize.OpenFile(path)
	if err != nil {
		fmt.Println("ERROR OpenFile:", err)
		os.Exit(1)
	}
	defer f.Close()

	sheet := f.GetSheetName(0)
	fmt.Println("Sheet:", sheet)

	rows, err := f.Rows(sheet)
	if err != nil {
		fmt.Println("ERROR Rows:", err)
		os.Exit(1)
	}
	defer rows.Close()

	// 必须先 Next() 才能获取第一行列数据
	if !rows.Next() {
		fmt.Println("ERROR: empty file")
		os.Exit(1)
	}
	header, err := rows.Columns()
	if err != nil {
		fmt.Println("ERROR Columns:", err)
		os.Exit(1)
	}

	fmt.Println("Header columns:", len(header))
	for i, h := range header {
		fmt.Printf("  [%d] %q\n", i, h)
	}

	count := 1 // header row already consumed
	for rows.Next() {
		count++
	}
	fmt.Println("Total rows (incl header):", count)
}
