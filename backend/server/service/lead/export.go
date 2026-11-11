package lead

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/xuri/excelize/v2"

	leadModel "github.com/flipped-aurora/gin-vue-admin/server/model/lead"
)

// writeLeadXlsx 把线索写成 Excel，返回文件路径
func writeLeadXlsx(leads []leadModel.Lead) (string, error) {
	file := excelize.NewFile()
	defer file.Close()

	sheet := "线索"
	index, _ := file.NewSheet(sheet)
	file.SetActiveSheet(index)
	file.DeleteSheet("Sheet1")

	headers := []string{"ID", "客户", "等级", "评分", "来源", "关键词", "地区", "联系方式", "微信", "标签", "备注", "话术", "采集时间"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		file.SetCellValue(sheet, cell, h)
	}

	for rowIndex, lead := range leads {
		row := rowIndex + 2
		values := []string{
			fmt.Sprintf("%d", derefInt64(lead.Id)),
			derefString(lead.ContactName),
			lead.Grade,
			fmt.Sprintf("%d", lead.Score),
			derefString(lead.Source),
			derefString(lead.Keyword),
			derefString(lead.City),
			derefString(lead.Phone),
			derefString(lead.Wechat),
			derefString(lead.Industry),
			derefString(lead.Remark),
			lead.DmScript,
			derefString(lead.CreatedAt),
		}

		for i, v := range values {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			file.SetCellValue(sheet, cell, v)
		}
	}

	style, _ := file.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Alignment: &excelize.Alignment{Vertical: "center"},
	})
	file.SetCellStyle(sheet, "A1", "M1", style)
	file.SetColWidth(sheet, "B", "B", 20)
	file.SetColWidth(sheet, "K", "K", 40)
	file.SetColWidth(sheet, "L", "L", 50)

	dir := `E:\ai\data\export`
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	path := filepath.Join(dir, "leads-"+time.Now().Format("20060102-150405")+".xlsx")

	if err := file.SaveAs(path); err != nil {
		return "", err
	}

	return path, nil
}

func derefInt64(v *int64) int64 {
	if v == nil {
		return 0
	}

	return *v
}
