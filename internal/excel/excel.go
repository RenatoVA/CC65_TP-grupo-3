package excel

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/extrame/xls"
	"github.com/xuri/excelize/v2"
)

type SheetRows struct {
	Name   string
	Header []string
	Rows   []map[string]string
}

func ReadFile(path string) ([]SheetRows, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".xlsx":
		return readXLSX(path)
	case ".xls":
		return readXLS(path)
	default:
		return nil, fmt.Errorf("unsupported extension: %s", ext)
	}
}

func readXLSX(path string) ([]SheetRows, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("open xlsx: %w", err)
	}
	defer f.Close()

	var out []SheetRows
	for _, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err != nil || len(rows) == 0 {
			continue
		}
		headerIdx, header := detectHeaderRows(rows)
		var data []map[string]string
		for _, row := range rows[headerIdx+1:] {
			if isEmptyRow(row) {
				continue
			}
			m := make(map[string]string, len(header))
			empty := true
			for i, h := range header {
				if h == "" {
					continue
				}
				if i < len(row) {
					value := strings.TrimSpace(row[i])
					m[h] = value
					if value != "" {
						empty = false
					}
				} else {
					m[h] = ""
				}
			}
			if empty {
				continue
			}
			data = append(data, m)
		}
		out = append(out, SheetRows{Name: sheet, Header: header, Rows: data})
	}
	return out, nil
}

func readXLS(path string) ([]SheetRows, error) {
	wb, err := xls.Open(path, "utf-8")
	if err != nil {
		return nil, fmt.Errorf("open xls: %w", err)
	}

	var out []SheetRows
	for i := 0; i < wb.NumSheets(); i++ {
		sheet := wb.GetSheet(i)
		if sheet == nil || sheet.MaxRow == 0 {
			continue
		}

		matrix := make([][]string, 0, int(sheet.MaxRow)+1)
		for r := 0; r <= int(sheet.MaxRow); r++ {
			row := sheet.Row(r)
			if row == nil {
				matrix = append(matrix, []string{})
				continue
			}
			line := make([]string, row.LastCol())
			for c := 0; c < row.LastCol(); c++ {
				line[c] = normalizeCell(row.Col(c))
			}
			matrix = append(matrix, line)
		}

		headerIdx, header := detectHeaderRows(matrix)
		var data []map[string]string
		for _, row := range matrix[headerIdx+1:] {
			if isEmptyRow(row) {
				continue
			}
			m := make(map[string]string, len(header))
			empty := true
			for c, h := range header {
				if h == "" {
					continue
				}
				if c < len(row) {
					value := normalizeCell(row[c])
					m[h] = value
					if value != "" {
						empty = false
					}
				} else {
					m[h] = ""
				}
			}
			if empty {
				continue
			}
			data = append(data, m)
		}
		out = append(out, SheetRows{Name: sheet.Name, Header: header, Rows: data})
	}

	return out, nil
}

func normalizeHeader(row []string) []string {
	out := make([]string, len(row))
	for i, value := range row {
		out[i] = normalizeCell(value)
	}
	return out
}

func detectHeaderRows(rows [][]string) (int, []string) {
	bestIdx := 0
	bestScore := -1
	bestHeader := normalizeHeader(rows[0])

	limit := len(rows)
	if limit > 12 {
		limit = 12
	}

	for i := 0; i < limit; i++ {
		header := normalizeHeader(rows[i])
		score := headerScore(header)
		if score > bestScore {
			bestScore = score
			bestIdx = i
			bestHeader = header
		}
	}

	return bestIdx, bestHeader
}

func headerScore(header []string) int {
	score := 0
	for _, col := range header {
		col = normalizeCell(col)
		if col == "" {
			continue
		}
		score++
		low := strings.ToLower(col)
		if strings.Contains(low, "exped") || strings.Contains(low, "fecha") || strings.Contains(low, "denunci") || strings.Contains(low, "ruc") || strings.Contains(low, "sector") || strings.Contains(low, "mater") || strings.Contains(low, "resol") || strings.Contains(low, "tipo") || strings.Contains(low, "proced") {
			score += 3
		}
		if strings.Contains(low, "listado de expedientes") || strings.Contains(low, "indecopi") {
			score -= 2
		}
	}
	return score
}

func normalizeCell(s string) string {
	replacer := strings.NewReplacer(
		"\u00a0", " ",
		"\r", " ",
		"\n", " ",
		"\t", " ",
	)
	s = replacer.Replace(s)
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSpace(s)
}

func isEmptyRow(row []string) bool {
	for _, value := range row {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}
