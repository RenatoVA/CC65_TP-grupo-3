package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"tp-programacion-concurente/internal/enricher"
	"tp-programacion-concurente/internal/normalize"
)

func main() {
	input := flag.String("input", "data/processed/consolidated.csv", "CSV de entrada")
	output := flag.String("output", "data/processed/enriched.csv", "CSV de salida")
	model := flag.String("model", "", "Modelo OpenRouter (requerido)")
	baseURL := flag.String("base-url", "https://openrouter.ai/api/v1", "Base URL de la API")
	delayMs := flag.Int("delay-ms", 150, "Pausa entre llamadas API (ms)")
	limit := flag.Int("limit", 0, "Máx. filas a procesar en esta corrida (0 = sin límite)")
	flag.Parse()

	if *model == "" {
		log.Fatal("--model es requerido (ej: --model=openai/gpt-4o-mini)")
	}

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENROUTER_API_KEY no está definida")
	}

	client := enricher.NewClient(apiKey, *baseURL, *model)
	delay := time.Duration(*delayMs) * time.Millisecond

	// Resolver resume: contar filas ya procesadas en el output
	alreadyDone := countOutputRows(*output)
	if alreadyDone > 0 {
		log.Printf("resume: %d filas ya procesadas en %s, continuando desde ahí", alreadyDone, *output)
	}

	// Abrir input
	inFile, err := os.Open(*input)
	if err != nil {
		log.Fatalf("abrir input: %v", err)
	}
	defer inFile.Close()

	reader := csv.NewReader(inFile)
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	// Leer y descartar header del input
	header, err := reader.Read()
	if err != nil {
		log.Fatalf("leer header: %v", err)
	}
	_ = header

	// Saltar filas ya procesadas
	for i := 0; i < alreadyDone; i++ {
		if _, err := reader.Read(); err != nil {
			log.Fatalf("saltar fila %d: %v", i+1, err)
		}
	}

	// Abrir output (append o nuevo)
	outFile, err := openOutput(*output, alreadyDone == 0)
	if err != nil {
		log.Fatalf("abrir output: %v", err)
	}
	defer outFile.Close()

	writer := csv.NewWriter(outFile)
	defer writer.Flush()

	if alreadyDone == 0 {
		outHeader := []string{
			"dataset_id", "source_sheet", "source_row", "expediente", "tipo_expediente",
			"fecha_presentacion", "denunciante", "denunciado", "ruc", "materia",
			"resumen", "raw_column_count", "queja",
		}
		if err := writer.Write(outHeader); err != nil {
			log.Fatalf("escribir header: %v", err)
		}
		writer.Flush()
	}

	ctx := context.Background()
	start := time.Now()
	total := 0
	generated := 0
	skipped := 0
	errors := 0

	for {
		if *limit > 0 && total >= *limit {
			log.Printf("límite de %d filas alcanzado, deteniendo", *limit)
			break
		}

		row, err := reader.Read()
		if err != nil {
			break // EOF u otro error terminal
		}

		rec := parseRow(row)
		rowNum := alreadyDone + total + 1
		empresa := rec.Denunciado
		if empresa == "" {
			empresa = rec.Materia
		}
		if empresa == "" {
			empresa = rec.DatasetID
		}
		log.Printf("[fila %d] procesando | expediente=%q empresa=%q tipo=%q",
			rowNum, rec.Expediente, empresa, rec.TipoExpediente)

		queja, enrichErr := enricher.Enrich(ctx, client, rec)
		if enrichErr != nil {
			log.Printf("[fila %d] ERROR llamada API: %v", rowNum, enrichErr)
			queja = "ERROR"
			errors++
		} else if queja == "" {
			log.Printf("[fila %d] SKIP sin datos suficientes (empresa y sector vacíos)", rowNum)
			skipped++
		} else {
			log.Printf("[fila %d] OK queja=%q", rowNum, queja)
			generated++
		}

		outRow := append(rowFields(rec), queja)
		if err := writer.Write(outRow); err != nil {
			log.Fatalf("escribir fila: %v", err)
		}

		total++

		if total%500 == 0 {
			writer.Flush()
			elapsed := time.Since(start)
			rate := float64(total) / elapsed.Seconds()
			log.Printf("--- progreso: %d filas (+%d previas) | generadas=%d skipped=%d errores=%d | %.1f filas/s ---",
				total, alreadyDone, generated, skipped, errors, rate)
		}

		if delay > 0 {
			time.Sleep(delay)
		}
	}

	writer.Flush()
	elapsed := time.Since(start)
	fmt.Printf("\nlisto: total=%d generadas=%d skipped=%d errores=%d tiempo=%s\n",
		total, generated, skipped, errors, elapsed.Round(time.Second))
}

func countOutputRows(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0 // archivo no existe
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		count++
	}
	if count > 0 {
		count-- // descontar header
	}
	return count
}

func openOutput(path string, create bool) (*os.File, error) {
	if create {
		return os.Create(path)
	}
	return os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
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

func rowFields(r normalize.ConsolidatedRecord) []string {
	return []string{
		r.DatasetID,
		r.SourceSheet,
		strconv.Itoa(r.SourceRow),
		r.Expediente,
		r.TipoExpediente,
		r.FechaPresentacion,
		r.Denunciante,
		r.Denunciado,
		r.RUC,
		r.Materia,
		r.Resumen,
		strconv.Itoa(r.RawColumnCount),
	}
}
