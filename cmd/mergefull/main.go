// mergefull: concatena enriched_full.csv y enriched_aug.csv en un único CSV de ~1M filas.
//
// Uso:
//   go run ./cmd/mergefull
//   go run ./cmd/mergefull -a=data/processed/enriched_full.csv -b=data/processed/enriched_aug.csv -output=data/processed/enriched_1m_real.csv
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
	fileA  := flag.String("a",      "data/processed/enriched_full.csv",      "Primer CSV")
	fileB  := flag.String("b",      "data/processed/enriched_aug.csv",        "Segundo CSV (sin header)")
	output := flag.String("output", "data/processed/enriched_1m_real.csv",    "CSV de salida")
	flag.Parse()

	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		log.Fatalf("crear directorio: %v", err)
	}

	out, err := os.Create(*output)
	if err != nil {
		log.Fatalf("crear output: %v", err)
	}
	defer out.Close()

	w := bufio.NewWriterSize(out, 4*1024*1024)

	total := copyFile(*fileA, w, false) // incluye header
	total += copyFile(*fileB, w, true)  // salta header

	if err := w.Flush(); err != nil {
		log.Fatalf("flush: %v", err)
	}

	info, _ := out.Stat()
	fmt.Printf("listo: %d filas | %.1f MB | %s\n", total, float64(info.Size())/1024/1024, *output)
}

func copyFile(path string, w *bufio.Writer, skipHeader bool) int {
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("abrir %s: %v", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	count := 0
	first := true
	for scanner.Scan() {
		if first && skipHeader {
			first = false
			continue
		}
		first = false
		w.WriteString(scanner.Text())
		w.WriteByte('\n')
		count++
	}
	log.Printf("  %s: %d filas copiadas", path, count)
	return count
}
