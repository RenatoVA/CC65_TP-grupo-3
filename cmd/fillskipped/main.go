package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"tp-programacion-concurente/internal/enricher"
	"tp-programacion-concurente/internal/normalize"
)

func main() {
	input := flag.String("input", "data/processed/enriched.csv", "CSV con filas a rellenar")
	output := flag.String("output", "data/processed/enriched_filled.csv", "CSV de salida")
	flag.Parse()

	inFile, err := os.Open(*input)
	if err != nil {
		log.Fatalf("abrir input: %v", err)
	}
	defer inFile.Close()

	outFile, err := os.Create(*output)
	if err != nil {
		log.Fatalf("crear output: %v", err)
	}
	defer outFile.Close()

	reader := csv.NewReader(inFile)
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	writer := csv.NewWriter(outFile)
	defer writer.Flush()

	// Copiar header tal cual
	header, err := reader.Read()
	if err != nil {
		log.Fatalf("leer header: %v", err)
	}
	if err := writer.Write(header); err != nil {
		log.Fatalf("escribir header: %v", err)
	}

	total := 0
	patched := 0
	alreadyFilled := 0

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("error leyendo fila %d: %v (saltando)", total+1, err)
			continue
		}

		total++

		// Columna 12 (índice 12) es queja
		queja := ""
		if len(row) > 12 {
			queja = strings.TrimSpace(row[12])
		}

		if queja == "" || queja == "ERROR" {
			rec := parseRow(row)
			queja = enricher.FillFromTemplates(rec)
			log.Printf("[fila %d] FILL queja=%q", total, queja)
			patched++
		} else {
			alreadyFilled++
		}

		// Asegurar que la fila tenga exactamente 13 columnas
		for len(row) < 12 {
			row = append(row, "")
		}
		if len(row) == 12 {
			row = append(row, queja)
		} else {
			row[12] = queja
		}

		if err := writer.Write(row); err != nil {
			log.Fatalf("escribir fila %d: %v", total, err)
		}

		if total%1000 == 0 {
			writer.Flush()
			log.Printf("--- progreso: %d filas | rellenadas=%d ya_tenían=%d ---", total, patched, alreadyFilled)
		}
	}

	writer.Flush()
	fmt.Printf("\nlisto: total=%d rellenadas=%d ya_tenían_queja=%d\n", total, patched, alreadyFilled)
}

func parseRow(row []string) normalize.ConsolidatedRecord {
	get := func(i int) string {
		if i < len(row) {
			return strings.TrimSpace(row[i])
		}
		return ""
	}
	sourceRow, _ := strconv.Atoi(get(2))
	rawCount, _ := strconv.Atoi(get(11))
	return normalize.ConsolidatedRecord{
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
}
