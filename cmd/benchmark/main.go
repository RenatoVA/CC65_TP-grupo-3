package main

import (
	"flag"
	"fmt"
	"log"
	"runtime"
	"strconv"
	"strings"

	"tp-programacion-concurente/internal/detector"
)

func main() {
	input := flag.String("input", "data/processed/enriched_filled.csv", "CSV de entrada")
	runs := flag.Int("runs", 10, "Corridas por configuración")
	workersList := flag.String("workers-list", "2,4,8", "Workers a probar, separados por coma")
	threshold := flag.Float64("threshold", 0.60, "Umbral Jaccard")
	burstLimit := flag.Int("burst-limit", 5, "Max expedientes por fecha")
	flag.Parse()

	workerCounts := parseWorkersList(*workersList)

	log.Printf("cargando dataset: %s", *input)
	records, err := detector.LoadCSV(*input)
	if err != nil {
		log.Fatalf("error cargando CSV: %v", err)
	}

	opts := detector.Options{
		JaccardThreshold: *threshold,
		BurstLimit:       *burstLimit,
	}

	fmt.Printf("\n╔══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║       BENCHMARK: Detector de Red Flags INDECOPI              ║\n")
	fmt.Printf("╚══════════════════════════════════════════════════════════════╝\n")
	fmt.Printf("Dataset:    %s (%d registros)\n", *input, len(records))
	fmt.Printf("Umbral:     %.2f Jaccard | Burst: %d exp/fecha\n", *threshold, *burstLimit)
	fmt.Printf("Corridas:   %d por configuración | CPUs: %d\n\n", *runs, runtime.NumCPU())

	// ── Secuencial ──────────────────────────────────────────────────────────
	fmt.Printf("━━━ Secuencial (%d corridas) ━━━\n", *runs)
	fmt.Printf("%-8s %10s\n", "Corrida", "Tiempo(ms)")
	seqDurations := make([]float64, 0, *runs)
	resBeforeSeq := detector.CaptureResources()

	for run := 1; run <= *runs; run++ {
		_, d := detector.DetectSequential(records, opts)
		ms := float64(d.Milliseconds())
		seqDurations = append(seqDurations, ms)
		fmt.Printf("%-8d %10.0f\n", run, ms)
	}
	resAfterSeq := detector.CaptureResources()
	seqMean := detector.TrimmedMean(seqDurations, 0.10)
	fmt.Printf("Media recortada (10%%): %.0f ms\n\n", seqMean)

	// ── Concurrente por número de workers ───────────────────────────────────
	type concResult struct {
		workers int
		mean    float64
		resBefore detector.ResourceSnapshot
		resAfter  detector.ResourceSnapshot
	}
	var concResults []concResult

	for _, w := range workerCounts {
		fmt.Printf("━━━ Concurrente %d workers (%d corridas) ━━━\n", w, *runs)
		fmt.Printf("%-8s %10s\n", "Corrida", "Tiempo(ms)")
		durations := make([]float64, 0, *runs)
		rBefore := detector.CaptureResources()

		for run := 1; run <= *runs; run++ {
			_, d := detector.DetectConcurrent(records, opts, w)
			ms := float64(d.Milliseconds())
			durations = append(durations, ms)
			fmt.Printf("%-8d %10.0f\n", run, ms)
		}
		rAfter := detector.CaptureResources()
		mean := detector.TrimmedMean(durations, 0.10)
		fmt.Printf("Media recortada (10%%): %.0f ms\n\n", mean)

		concResults = append(concResults, concResult{w, mean, rBefore, rAfter})
	}

	// ── Tabla de Speedup ────────────────────────────────────────────────────
	fmt.Printf("┌─────────────────────────────────────────────────────┐\n")
	fmt.Printf("│                   TABLA DE SPEEDUP                   │\n")
	fmt.Printf("├──────────┬──────────────┬─────────┬─────────────────┤\n")
	fmt.Printf("│ %-8s │ %12s │ %7s │ %15s │\n", "Versión", "T_media(ms)", "Speedup", "Eficiencia")
	fmt.Printf("├──────────┼──────────────┼─────────┼─────────────────┤\n")
	fmt.Printf("│ %-8s │ %12.0f │ %7.3f │ %15.3f │\n", "Secuencial", seqMean, 1.0, 1.0)
	for _, cr := range concResults {
		speedup := seqMean / cr.mean
		efficiency := speedup / float64(cr.workers)
		fmt.Printf("│ %-8s │ %12.0f │ %7.3f │ %15.3f │\n",
			fmt.Sprintf("Conc x%d", cr.workers), cr.mean, speedup, efficiency)
	}
	fmt.Printf("└──────────┴──────────────┴─────────┴─────────────────┘\n\n")

	// ── Análisis de Recursos ─────────────────────────────────────────────────
	fmt.Printf("┌─────────────────────────────────────────────────────────────┐\n")
	fmt.Printf("│                    USO DE RECURSOS                          │\n")
	fmt.Printf("├────────────────┬───────────┬────────────┬───────┬──────────┤\n")
	fmt.Printf("│ %-14s │ %9s │ %10s │ %5s │ %8s │\n",
		"Versión", "HeapAlloc", "TotalAlloc", "GC", "Gorout.")
	fmt.Printf("├────────────────┼───────────┼────────────┼───────┼──────────┤\n")
	fmt.Printf("│ %-14s │ %8.1f M │ %9.1f M │ %5d │ %8d │\n",
		"Secuencial",
		resAfterSeq.HeapAllocMB-resBeforeSeq.HeapAllocMB,
		resAfterSeq.TotalAllocMB-resBeforeSeq.TotalAllocMB,
		resAfterSeq.GCCycles-resBeforeSeq.GCCycles,
		resAfterSeq.Goroutines)
	for _, cr := range concResults {
		fmt.Printf("│ %-14s │ %8.1f M │ %9.1f M │ %5d │ %8d │\n",
			fmt.Sprintf("Conc x%d", cr.workers),
			cr.resAfter.HeapAllocMB-cr.resBefore.HeapAllocMB,
			cr.resAfter.TotalAllocMB-cr.resBefore.TotalAllocMB,
			cr.resAfter.GCCycles-cr.resBefore.GCCycles,
			cr.resAfter.Goroutines)
	}
	fmt.Printf("└────────────────┴───────────┴────────────┴───────┴──────────┘\n")
}

func parseWorkersList(s string) []int {
	var result []int
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if n, err := strconv.Atoi(part); err == nil && n > 0 {
			result = append(result, n)
		}
	}
	if len(result) == 0 {
		result = []int{2, 4, 8}
	}
	return result
}
