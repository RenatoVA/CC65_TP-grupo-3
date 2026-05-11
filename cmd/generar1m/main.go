// generar1m: genera data/processed/enriched_1m.csv repitiendo el dataset 100 veces.
// Reemplaza el script bash para que funcione en cualquier SO (incluido Windows).
//
// Uso:
//   go run ./cmd/generar1m
//   go run ./cmd/generar1m -input=data/processed/enriched_filled.csv -output=data/processed/enriched_1m.csv -reps=100
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func main() {
	input  := flag.String("input",  "data/processed/enriched_filled.csv", "CSV fuente")
	output := flag.String("output", "data/processed/enriched_1m.csv",      "CSV destino")
	reps   := flag.Int("reps",     100,                                    "Veces que se repite el dataset")
	flag.Parse()

	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		log.Fatalf("crear directorio: %v", err)
	}

	src, err := os.Open(*input)
	if err != nil {
		log.Fatalf("abrir input: %v", err)
	}
	defer src.Close()

	// Leer header
	scanner := bufio.NewScanner(src)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	if !scanner.Scan() {
		log.Fatal("input vacío o sin header")
	}
	header := scanner.Text()

	// Leer todas las filas de datos en memoria
	var rows []string
	for scanner.Scan() {
		if line := scanner.Text(); line != "" {
			rows = append(rows, line)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("leer input: %v", err)
	}
	src.Close()

	fmt.Printf("Filas fuente: %d | Repeticiones: %d | Total estimado: %d\n",
		len(rows), *reps, len(rows)**reps)
	fmt.Printf("Generando %s ...\n", *output)

	dst, err := os.Create(*output)
	if err != nil {
		log.Fatalf("crear output: %v", err)
	}
	defer dst.Close()

	w := bufio.NewWriterSize(dst, 4*1024*1024)

	fmt.Fprintln(w, header)

	written := 0
	for i := 1; i <= *reps; i++ {
		for _, row := range rows {
			fmt.Fprintln(w, row)
			written++
		}
		fmt.Printf("\r  bloque %3d/%d  (%d filas escritas)", i, *reps, written)
	}

	if err := w.Flush(); err != nil {
		log.Fatalf("flush: %v", err)
	}

	info, _ := dst.Stat()
	sizeMB := float64(info.Size()) / 1024 / 1024
	fmt.Printf("\nListo: %d filas | %.1f MB | %s\n", written, sizeMB, *output)
}
