// Package export writes Excel (.xlsx) downloads for listing endpoints.
package export

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

const batchSize = 1000

// rows writes an .xlsx attachment using a stream writer.
// The organisation letterhead (same fields as the ILR) sits above the table.
func rows(c fiber.Ctx, sheet string, filePrefix string, headers []any, write func(sw *excelize.StreamWriter, row int) (int, error)) error {
	f := excelize.NewFile()
	defer f.Close()

	f.SetSheetName("Sheet1", sheet)
	sw, err := f.NewStreamWriter(sheet)
	if err != nil {
		return err
	}
	row, err := writeSheetLetterhead(sw, sheet)
	if err != nil {
		return err
	}
	headerRow := row
	if err := sw.SetRow(fmt.Sprintf("A%d", row), headers); err != nil {
		return err
	}
	if _, err := write(sw, row+1); err != nil {
		return err
	}
	if err := sw.Flush(); err != nil {
		return err
	}
	applyPrintLayout(f, sheet, headerRow)

	name := fmt.Sprintf("%s_%s.xlsx", filePrefix, time.Now().Format("02012006_150405"))
	c.Attachment(name)
	c.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	return f.Write(c.Response().BodyWriter())
}

func writeSheetLetterhead(sw *excelize.StreamWriter, title string) (int, error) {
	head := resolveLetterhead(Letterhead{})
	row := 1
	for _, line := range head.Lines() {
		if err := sw.SetRow(fmt.Sprintf("A%d", row), []any{line}); err != nil {
			return 0, err
		}
		row++
	}
	row++
	if err := sw.SetRow(fmt.Sprintf("A%d", row), []any{title}); err != nil {
		return 0, err
	}
	row++
	if err := sw.SetRow(fmt.Sprintf("A%d", row), []any{"Generated " + time.Now().Format("02/01/2006 15:04")}); err != nil {
		return 0, err
	}
	return row + 2, nil
}

func applyPrintLayout(f *excelize.File, sheet string, headerRow int) {
	head := resolveLetterhead(Letterhead{})
	one := 1
	orient := "landscape"
	_ = f.SetPageLayout(sheet, &excelize.PageLayoutOptions{
		Orientation: &orient,
		FitToWidth:  &one,
	})
	_ = f.SetHeaderFooter(sheet, &excelize.HeaderFooterOptions{
		OddHeader: "&L" + head.DisplayName() + "&R&D &T",
		OddFooter: "&CPage &P of &N",
	})
	if headerRow > 0 {
		_ = f.SetPanes(sheet, &excelize.Panes{
			Freeze:      true,
			YSplit:      headerRow,
			TopLeftCell: fmt.Sprintf("A%d", headerRow+1),
			ActivePane:  "bottomLeft",
		})
		safe := strings.ReplaceAll(sheet, "'", "''")
		_ = f.SetDefinedName(&excelize.DefinedName{
			Name:     "_xlnm.Print_Titles",
			RefersTo: fmt.Sprintf("'%s'!$%d:$%d", safe, headerRow, headerRow),
			Scope:    sheet,
		})
	}
}

// Slice writes an in-memory table (aggregate reports) as an .xlsx download.
func Slice(c fiber.Ctx, sheet, filePrefix string, headers []any, data [][]any) error {
	return rows(c, sheet, filePrefix, headers, func(sw *excelize.StreamWriter, row int) (int, error) {
		for _, cells := range data {
			if err := sw.SetRow(fmt.Sprintf("A%d", row), cells); err != nil {
				return row, err
			}
			row++
		}
		return row, nil
	})
}

// Query streams rows from a GORM query into an .xlsx download.
// Uses OFFSET/LIMIT rather than FindInBatches: GORM batches always append the
// primary key to ORDER BY, and SQL Server rejects a column listed twice
// (list queries already add an ID tie-breaker for stable paging).
func Query[T any](c fiber.Ctx, q *gorm.DB, sheet, filePrefix string, headers []any, mapRow func(T) []any) error {
	return rows(c, sheet, filePrefix, headers, func(sw *excelize.StreamWriter, row int) (int, error) {
		offset := 0
		for {
			var batch []T
			if err := q.Session(&gorm.Session{}).Offset(offset).Limit(batchSize).Find(&batch).Error; err != nil {
				return row, err
			}
			if len(batch) == 0 {
				return row, nil
			}
			for _, item := range batch {
				if err := sw.SetRow(fmt.Sprintf("A%d", row), mapRow(item)); err != nil {
					return row, err
				}
				row++
			}
			if len(batch) < batchSize {
				return row, nil
			}
			offset += batchSize
		}
	})
}
