package export

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"tp-programacion-concurente/internal/normalize"
)

func WriteCSV(path string, rows []normalize.ConsolidatedRecord) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir output: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create csv: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	header := []string{
		"dataset_id",
		"source_sheet",
		"source_row",
		"expediente",
		"tipo_expediente",
		"fecha_presentacion",
		"denunciante",
		"denunciado",
		"ruc",
		"materia",
		"resumen",
		"raw_column_count",
	}

	if err := w.Write(header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	for _, row := range rows {
		record := []string{
			row.DatasetID,
			row.SourceSheet,
			strconv.Itoa(row.SourceRow),
			row.Expediente,
			row.TipoExpediente,
			row.FechaPresentacion,
			row.Denunciante,
			row.Denunciado,
			row.RUC,
			row.Materia,
			row.Resumen,
			strconv.Itoa(row.RawColumnCount),
		}
		if err := w.Write(record); err != nil {
			return fmt.Errorf("write record: %w", err)
		}
	}

	return nil
}
