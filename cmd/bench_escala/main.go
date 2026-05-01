// bench_escala: benchmark temporal sobre el dataset de ~1M registros.
//
// Ejecuta tres mediciones para comparar secuencial vs concurrente:
//
//  Parte A — Carga
//    Tiempo de leer y tokenizar todos los registros del CSV de 1M.
//
//  Parte B — TEXT_REPEAT (O(n²)) sobre muestra de los primeros N registros
//    El O(n²) completo sobre 1M filas requeriría ~500 mil millones de
//    comparaciones (inviable). Se usa la muestra para mostrar el speedup
//    manteniendo tiempos de ejecución razonables.
//
//  Parte C — TIMING_BURST + EXACT_DUPLICATE (O(n)) sobre el 1M completo
//    Estas fases escalan linealmente y se ejecutan sobre todos los registros.
//
// Uso:
//   go run ./cmd/bench_escala
//   go run ./cmd/bench_escala -input=data/processed/enriched_1m.csv -sample=5000 -workers=2 -runs=3
package main

import (
	"flag"
	"fmt"
	"log"
	"runtime"
	"time"

	"tp-programacion-concurente/internal/detector"
)

func main() {
	input   := flag.String("input",   "data/processed/enriched_1m.csv",    "CSV de 1M registros")
	sample  := flag.Int("sample",    5000,                                  "Registros para TEXT_REPEAT (O(n²))")
	workers := flag.Int("workers",   runtime.NumCPU(),                      "Workers concurrentes")
	runs    := flag.Int("runs",      3,                                      "Corridas por medición")
	flag.Parse()

	banner("BENCHMARK DE ESCALA — ~1M REGISTROS")
	fmt.Printf("Input:    %s\n", *input)
	fmt.Printf("CPUs:     %d | Workers: %d\n", runtime.NumCPU(), *workers)
	fmt.Printf("Corridas: %d | Sample TEXT_REPEAT: %d registros\n\n", *runs, *sample)

	opts := detector.DefaultOptions()

	// ─── Parte A: Carga ─────────────────────────────────────────────────────
	section("A — Carga del CSV completo")
	var records []detector.DetectorRecord
	var loadTimes []float64

	for r := 1; r <= *runs; r++ {
		t0 := time.Now()
		recs, err := detector.LoadCSV(*input)
		elapsed := time.Since(t0)
		if err != nil {
			log.Fatalf("error cargando CSV: %v", err)
		}
		records = recs
		loadTimes = append(loadTimes, float64(elapsed.Milliseconds()))
		fmt.Printf("  corrida %d: %s (%d registros cargados)\n", r, elapsed.Round(time.Millisecond), len(recs))
	}
	loadMean := detector.TrimmedMean(loadTimes, 0.10)
	fmt.Printf("  Media recortada (10%%): %.0f ms\n\n", loadMean)

	total := len(records)
	fmt.Printf("  Total registros: %d\n\n", total)

	// ─── Parte B: TEXT_REPEAT sobre muestra ─────────────────────────────────
	section(fmt.Sprintf("B — TEXT_REPEAT (O(n²)) sobre muestra de %d registros", *sample))
	fmt.Printf("  Pares a comparar: %d\n\n", (*sample)*(*sample-1)/2)

	muestra := records
	if *sample < total {
		muestra = records[:*sample]
	}

	var seqBTimes, concBTimes []float64

	fmt.Printf("  [Secuencial]\n")
	for r := 1; r <= *runs; r++ {
		t0 := time.Now()
		detector.DetectSequential(muestra, opts)
		elapsed := time.Since(t0)
		seqBTimes = append(seqBTimes, float64(elapsed.Milliseconds()))
		fmt.Printf("    corrida %d: %s\n", r, elapsed.Round(time.Millisecond))
	}
	seqBMean := detector.TrimmedMean(seqBTimes, 0.10)
	fmt.Printf("    Media recortada: %.0f ms\n\n", seqBMean)

	fmt.Printf("  [Concurrente — %d workers]\n", *workers)
	for r := 1; r <= *runs; r++ {
		t0 := time.Now()
		detector.DetectConcurrent(muestra, opts, *workers)
		elapsed := time.Since(t0)
		concBTimes = append(concBTimes, float64(elapsed.Milliseconds()))
		fmt.Printf("    corrida %d: %s\n", r, elapsed.Round(time.Millisecond))
	}
	concBMean := detector.TrimmedMean(concBTimes, 0.10)
	fmt.Printf("    Media recortada: %.0f ms\n\n", concBMean)

	speedupB := seqBMean / concBMean
	fmt.Printf("  Speedup TEXT_REPEAT (muestra %d): %.3fx\n\n", *sample, speedupB)

	// ─── Parte C: Fases O(n) sobre el 1M completo ───────────────────────────
	section(fmt.Sprintf("C — TIMING_BURST + EXACT_DUPLICATE (O(n)) sobre %d registros", total))

	var seqCTimes, concCTimes []float64

	fmt.Printf("  [Secuencial]\n")
	for r := 1; r <= *runs; r++ {
		t0 := time.Now()
		runLinearOnly(records, opts)
		elapsed := time.Since(t0)
		seqCTimes = append(seqCTimes, float64(elapsed.Milliseconds()))
		fmt.Printf("    corrida %d: %s\n", r, elapsed.Round(time.Millisecond))
	}
	seqCMean := detector.TrimmedMean(seqCTimes, 0.10)
	fmt.Printf("    Media recortada: %.0f ms\n\n", seqCMean)

	fmt.Printf("  [Concurrente — %d workers]\n", *workers)
	for r := 1; r <= *runs; r++ {
		t0 := time.Now()
		runLinearOnly(records, opts)
		elapsed := time.Since(t0)
		concCTimes = append(concCTimes, float64(elapsed.Milliseconds()))
		fmt.Printf("    corrida %d: %s\n", r, elapsed.Round(time.Millisecond))
	}
	concCMean := detector.TrimmedMean(concCTimes, 0.10)
	fmt.Printf("    Media recortada: %.0f ms\n\n", concCMean)

	// Las fases O(n) son secuenciales en ambas versiones
	fmt.Printf("  Nota: TIMING_BURST y EXACT_DUPLICATE son O(n) y no se\n")
	fmt.Printf("  paralelizan en el diseño actual (overhead > beneficio).\n\n")

	// ─── Resumen ─────────────────────────────────────────────────────────────
	section("RESUMEN")
	resMem := detector.CaptureResources()
	fmt.Printf("  %-35s  %8s\n", "Métrica", "Valor")
	fmt.Printf("  %s\n", dashes(50))
	fmt.Printf("  %-35s  %8d\n",   "Registros totales",                    total)
	fmt.Printf("  %-35s  %8.0f ms\n", "Carga CSV (media recortada)",       loadMean)
	fmt.Printf("  %-35s  %8.0f ms\n", "TEXT_REPEAT secuencial (muestra)",  seqBMean)
	fmt.Printf("  %-35s  %8.0f ms\n", fmt.Sprintf("TEXT_REPEAT concurrente x%d (muestra)", *workers), concBMean)
	fmt.Printf("  %-35s  %8.3f\n",    "Speedup TEXT_REPEAT",               speedupB)
	fmt.Printf("  %-35s  %8.0f ms\n", "O(n) fases sobre 1M (media)",       seqCMean)
	fmt.Printf("  %-35s  %8.1f MB\n", "HeapAlloc actual",                  resMem.HeapAllocMB)
	fmt.Printf("  %-35s  %8.1f MB\n", "TotalAlloc acumulado",              resMem.TotalAllocMB)
	fmt.Printf("  %-35s  %8d\n",      "Ciclos GC",                         resMem.GCCycles)
	fmt.Println()
}

// runLinearOnly ejecuta solo las fases O(n): TIMING_BURST y EXACT_DUPLICATE.
func runLinearOnly(records []detector.DetectorRecord, opts detector.Options) []detector.RedFlag {
	var flags []detector.RedFlag

	dateBuckets := make(map[string][]int, len(records)/10)
	for i, r := range records {
		if r.FechaPresentacion != "" {
			dateBuckets[r.FechaPresentacion] = append(dateBuckets[r.FechaPresentacion], i)
		}
	}
	for fecha, indices := range dateBuckets {
		if len(indices) > opts.BurstLimit {
			for _, idx := range indices {
				flags = append(flags, detector.RedFlag{
					RecordIndex: idx,
					FlagType:    detector.FlagTimingBurst,
					Score:       float64(len(indices)),
					Details:     fecha,
				})
			}
		}
	}

	seen := make(map[string]int, len(records))
	for i, r := range records {
		key := r.Expediente + "|" + r.Denunciado + "|" + r.FechaPresentacion
		if key == "||" {
			continue
		}
		if prev, ok := seen[key]; ok {
			flags = append(flags, detector.RedFlag{RecordIndex: prev, FlagType: detector.FlagExactDuplicate, Score: 1})
			flags = append(flags, detector.RedFlag{RecordIndex: i, FlagType: detector.FlagExactDuplicate, Score: 1})
		} else {
			seen[key] = i
		}
	}
	return flags
}

func banner(s string) {
	fmt.Printf("\n╔%s╗\n", dashes(len(s)+2))
	fmt.Printf("║ %s ║\n", s)
	fmt.Printf("╚%s╝\n\n", dashes(len(s)+2))
}

func section(s string) {
	fmt.Printf("━━━ %s %s\n", s, dashes(max(0, 55-len(s))))
}

func dashes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = '─'
	}
	return string(b)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
