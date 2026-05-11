// enrichsynthetic: genera quejas para el dataset completo (538k filas) combinando:
//   - Quejas LLM ya existentes en enriched_filled.csv (se conservan tal cual)
//   - Quejas de plantillas combinatorias para el resto (sin costo de API)
//
// Resultado: data/processed/enriched_full.csv con ~538k filas únicas, sin repetición.
//
// Uso:
//   go run ./cmd/enrichsynthetic
//   go run ./cmd/enrichsynthetic -enriched=data/processed/enriched_filled.csv -consolidated=data/processed/consolidated.csv -output=data/processed/enriched_full.csv
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"tp-programacion-concurente/internal/enricher"
	"tp-programacion-concurente/internal/normalize"
)

func main() {
	enrichedPath    := flag.String("enriched",    "data/processed/enriched_filled.csv", "CSV con quejas LLM existentes")
	consolidatedPath := flag.String("consolidated", "data/processed/consolidated.csv",   "CSV completo de INDECOPI")
	outputPath      := flag.String("output",      "data/processed/enriched_full.csv",   "CSV de salida combinado")
	flag.Parse()

	// ── Paso 1: cargar quejas LLM existentes en memoria ──────────────────────
	log.Printf("cargando quejas existentes de %s ...", *enrichedPath)
	existing, err := loadExistingQuejas(*enrichedPath)
	if err != nil {
		log.Fatalf("error cargando enriched: %v", err)
	}
	log.Printf("%d quejas LLM cargadas", len(existing))

	// ── Paso 2: procesar consolidated.csv y generar quejas faltantes ──────────
	log.Printf("procesando %s ...", *consolidatedPath)

	inFile, err := os.Open(*consolidatedPath)
	if err != nil {
		log.Fatalf("abrir consolidated: %v", err)
	}
	defer inFile.Close()

	if err := os.MkdirAll(filepath.Dir(*outputPath), 0o755); err != nil {
		log.Fatalf("crear directorio output: %v", err)
	}
	outFile, err := os.Create(*outputPath)
	if err != nil {
		log.Fatalf("crear output: %v", err)
	}
	defer outFile.Close()

	reader := csv.NewReader(inFile)
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	writer := csv.NewWriter(outFile)
	defer writer.Flush()

	// Escribir header con columna queja
	if _, err := reader.Read(); err != nil {
		log.Fatalf("leer header: %v", err)
	}
	_ = writer.Write([]string{
		"dataset_id", "source_sheet", "source_row", "expediente", "tipo_expediente",
		"fecha_presentacion", "denunciante", "denunciado", "ruc", "materia",
		"resumen", "raw_column_count", "queja",
	})

	total := 0
	usedLLM := 0
	usedTemplate := 0

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		get := func(i int) string {
			if i < len(row) {
				return strings.TrimSpace(row[i])
			}
			return ""
		}

		key := get(0) + "|" + get(1) + "|" + get(2) // dataset_id|source_sheet|source_row

		var queja string
		if q, ok := existing[key]; ok && q != "" && q != "ERROR" {
			queja = q
			usedLLM++
		} else {
			sourceRow, _ := strconv.Atoi(get(2))
			rawCount, _ := strconv.Atoi(get(11))
			rec := normalize.ConsolidatedRecord{
				DatasetID:         get(0),
				SourceSheet:       get(1),
				SourceRow:         sourceRow,
				Expediente:        get(3),
				TipoExpediente:    get(4),
				FechaPresentacion: get(5),
				Denunciante:       get(6),
				Denunciado:        get(7),
				RUC:               get(8),
				Materia:           get(9),
				Resumen:           get(10),
				RawColumnCount:    rawCount,
			}
			queja = enricher.FillFromTemplates(rec)
			usedTemplate++
		}

		outRow := append(row[:12], queja) // conservar las 12 columnas originales + queja
		if len(row) < 12 {
			for len(outRow) < 13 {
				outRow = append(outRow, "")
			}
			outRow[12] = queja
		}
		_ = writer.Write(outRow[:13])

		total++
		if total%50000 == 0 {
			writer.Flush()
			log.Printf("  %d filas procesadas (llm=%d plantilla=%d)", total, usedLLM, usedTemplate)
		}
	}

	writer.Flush()
	fmt.Printf("\nlisto:\n")
	fmt.Printf("  total filas:      %d\n", total)
	fmt.Printf("  quejas LLM:       %d (%.1f%%)\n", usedLLM, 100*float64(usedLLM)/float64(total))
	fmt.Printf("  quejas plantilla: %d (%.1f%%)\n", usedTemplate, 100*float64(usedTemplate)/float64(total))
	fmt.Printf("  output:           %s\n", *outputPath)
}

// loadExistingQuejas carga el mapa key→queja del CSV enriquecido.
// key = "dataset_id|source_sheet|source_row"
func loadExistingQuejas(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.LazyQuotes = true
	r.FieldsPerRecord = -1
	r.Read() // header

	result := make(map[string]string, 15000)
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		get := func(i int) string {
			if i < len(row) {
				return strings.TrimSpace(row[i])
			}
			return ""
		}
		key := get(0) + "|" + get(1) + "|" + get(2)
		queja := get(12)
		if key != "||" && queja != "" {
			result[key] = queja
		}
	}
	return result, nil
}
