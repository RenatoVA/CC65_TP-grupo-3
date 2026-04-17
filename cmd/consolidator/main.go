package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"tp-programacion-concurente/internal/catalog"
	"tp-programacion-concurente/internal/downloader"
	"tp-programacion-concurente/internal/excel"
	"tp-programacion-concurente/internal/export"
	"tp-programacion-concurente/internal/normalize"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	datasets := catalog.DefaultCatalog()
	dl := downloader.New()

	var rows []normalize.ConsolidatedRecord
	var profiles []export.DatasetProfile
	warnings := make([]string, 0)
	filesDownloaded := 0
	duplicateKeys := map[string]int{}

	for _, ds := range datasets {
		for idx, url := range ds.ResourceURLs {
			filename := filepath.Base(strings.Split(url, "?")[0])
			if filename == "." || filename == "/" || filename == "" {
				filename = fmt.Sprintf("%s_%d.bin", ds.ID, idx+1)
			}

			dst := filepath.Join("data", "raw", ds.ID, filename)
			log.Printf("descargando %s -> %s", url, dst)
			if err := dl.Download(ctx, url, dst); err != nil {
				warnings = append(warnings, fmt.Sprintf("fallo descarga %s: %v", url, err))
				continue
			}
			filesDownloaded++

			sheets, err := excel.ReadFile(dst)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("fallo lectura %s: %v", dst, err))
				continue
			}

			for _, sheet := range sheets {
				columnNames := compactColumns(sheet.Header)
				if shouldSkipSheet(sheet.Name, columnNames, len(sheet.Rows)) {
					warnings = append(warnings, fmt.Sprintf("hoja omitida %s (%s)", ds.ID, sheet.Name))
					continue
				}
				profiles = append(profiles, export.DatasetProfile{
					DatasetID:   ds.ID,
					SourceFile:  dst,
					SheetName:   sheet.Name,
					RowCount:    len(sheet.Rows),
					ColumnCount: len(columnNames),
					ColumnNames: columnNames,
				})

				for rowIdx, values := range sheet.Rows {
					raw := normalize.RawRecord{
						DatasetID: ds.ID,
						SheetName: sheet.Name,
						RowIndex:  rowIdx + 2,
						Values:    values,
					}
					row := normalize.FromMap(raw)
					rows = append(rows, row)

					dupKey := strings.TrimSpace(row.Expediente) + "|" + strings.TrimSpace(row.FechaPresentacion) + "|" + strings.TrimSpace(row.RUC)
					if dupKey != "||" {
						duplicateKeys[dupKey]++
					}
				}
			}
		}
	}

	dupCount := 0
	for _, count := range duplicateKeys {
		if count > 1 {
			dupCount++
		}
	}

	if err := export.WriteCSV(filepath.Join("data", "processed", "consolidated.csv"), rows); err != nil {
		log.Fatalf("error escribiendo csv: %v", err)
	}

	if err := export.WriteProfiles(filepath.Join("data", "processed", "profiles.json"), profiles); err != nil {
		log.Fatalf("error escribiendo perfiles: %v", err)
	}

	report := export.Report{
		DatasetsConfigured: len(datasets),
		FilesDownloaded:    filesDownloaded,
		RowsConsolidated:   len(rows),
		DuplicateKeys:      dupCount,
		Warnings:           warnings,
	}

	if err := export.WriteReport(filepath.Join("data", "processed", "report.json"), report); err != nil {
		log.Fatalf("error escribiendo reporte: %v", err)
	}

	log.Printf("listo, filas=%d, descargas=%d, perfiles=%d, warnings=%d", len(rows), filesDownloaded, len(profiles), len(warnings))
}

func compactColumns(header []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(header))
	for _, col := range header {
		col = strings.TrimSpace(col)
		if col == "" {
			continue
		}
		if _, ok := seen[col]; ok {
			continue
		}
		seen[col] = struct{}{}
		out = append(out, col)
	}
	sort.Strings(out)
	return out
}

func shouldSkipSheet(sheetName string, columnNames []string, rowCount int) bool {
	name := strings.ToLower(strings.TrimSpace(sheetName))
	if name == "" {
		return false
	}
	if strings.Contains(name, "graf") || strings.Contains(name, "chart") {
		return true
	}
	if rowCount == 0 {
		return true
	}
	if len(columnNames) <= 1 {
		joined := strings.ToLower(strings.Join(columnNames, " | "))
		if strings.Contains(joined, "listado") || strings.Contains(joined, "expedientes ingresados") || strings.Contains(joined, "registros otorgados") || strings.Contains(joined, "total expedientes") {
			return true
		}
	}
	return false
}
