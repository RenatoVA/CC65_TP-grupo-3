// augment: genera variantes adicionales del dataset sin duplicar filas.
//
// Toma enriched_full.csv y produce una segunda queja por fila usando un salt
// distinto en el hash de FillFromTemplates → misma metadata real, texto diferente.
// El output se puede concatenar con enriched_full.csv para llegar a ~1M filas únicas.
//
// Uso:
//   go run ./cmd/augment
//   go run ./cmd/augment -input=data/processed/enriched_full.csv -output=data/processed/enriched_aug.csv -salt=v2
//
// Para obtener el dataset de 1M:
//   go run ./cmd/augment
//   (luego usar enriched_full.csv + enriched_aug.csv como inputs del detector)
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"tp-programacion-concurente/internal/enricher"
	"tp-programacion-concurente/internal/normalize"
)

func main() {
	input  := flag.String("input",  "data/processed/enriched_full.csv",  "CSV fuente")
	output := flag.String("output", "data/processed/enriched_aug.csv",    "CSV de salida con variantes")
	salt   := flag.String("salt",   "v2",                                  "Salt para diferenciar el hash (cambiar para más variantes)")
	flag.Parse()

	inFile, err := os.Open(*input)
	if err != nil {
		log.Fatalf("abrir input: %v", err)
	}
	defer inFile.Close()

	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		log.Fatalf("crear directorio: %v", err)
	}
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

	// Leer y escribir header
	header, err := reader.Read()
	if err != nil {
		log.Fatalf("leer header: %v", err)
	}
	_ = writer.Write(header)

	total := 0
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

		// Generar queja con salt distinto → combinación diferente de plantilla
		queja := fillWithSalt(rec, *salt)

		outRow := make([]string, len(row))
		copy(outRow, row)
		if len(outRow) >= 13 {
			outRow[12] = queja
		} else {
			outRow = append(outRow, queja)
		}
		_ = writer.Write(outRow)

		total++
		if total%50000 == 0 {
			writer.Flush()
			log.Printf("  %d variantes generadas", total)
		}
	}

	writer.Flush()
	fmt.Printf("\nlisto:\n")
	fmt.Printf("  variantes generadas: %d\n", total)
	fmt.Printf("  salt usado:          %s\n", *salt)
	fmt.Printf("  output:              %s\n", *output)
	fmt.Printf("\nPara combinar con el dataset original:\n")
	fmt.Printf("  El detector acepta un solo CSV — concatena manualmente o usa -input con enriched_full.csv\n")
	fmt.Printf("  Para un dataset de ~1M: usa go run ./cmd/mergefull\n")
}

// fillWithSalt genera una queja usando un seed derivado del salt.
// Mismo registro + distinto salt = distinta combinación de plantilla.
func fillWithSalt(r normalize.ConsolidatedRecord, salt string) string {
	h := fnv.New64a()
	h.Write([]byte(r.DatasetID + "|" + r.SourceSheet + "|" + strconv.Itoa(r.SourceRow) + "|" + salt))
	seed := int64(h.Sum64())
	rng := rand.New(rand.NewSource(seed))
	return enricher.FillFromTemplatesWithRng(r, rng)
}
