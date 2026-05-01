package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"tp-programacion-concurente/internal/detector"
)

func main() {
	input := flag.String("input", "data/processed/enriched_filled.csv", "CSV de entrada con campo queja")
	output := flag.String("output", "data/results/flagged_records_conc.csv", "CSV de salida con red flags")
	threshold := flag.Float64("threshold", 0.60, "Umbral Jaccard para TEXT_REPEAT (0-1)")
	burstLimit := flag.Int("burst-limit", 5, "Max expedientes por fecha para TIMING_BURST")
	workers := flag.Int("workers", runtime.NumCPU(), "Número de goroutines worker")
	runs := flag.Int("runs", 1, "Número de corridas (mide tiempo de cada una)")
	flag.Parse()

	log.Printf("cargando dataset: %s", *input)
	records, err := detector.LoadCSV(*input)
	if err != nil {
		log.Fatalf("error cargando CSV: %v", err)
	}
	log.Printf("%d registros cargados | workers=%d | CPUs=%d", len(records), *workers, runtime.NumCPU())

	opts := detector.Options{
		JaccardThreshold: *threshold,
		BurstLimit:       *burstLimit,
	}

	resBefore := detector.CaptureResources()

	var flags []detector.RedFlag
	var durations []float64

	for run := 1; run <= *runs; run++ {
		f, d := detector.DetectConcurrent(records, opts, *workers)
		durations = append(durations, float64(d.Milliseconds()))
		flags = f
		log.Printf("[corrida %d/%d] tiempo=%s flags=%d workers=%d",
			run, *runs, d.Round(time.Millisecond), len(f), *workers)
	}

	resAfter := detector.CaptureResources()

	if *runs > 1 {
		mean := detector.TrimmedMean(durations, 0.10)
		fmt.Printf("\nMedia recortada (10%%) sobre %d corridas: %.0f ms\n", *runs, mean)
	}

	fmt.Printf("\n=== Recursos ===\n")
	fmt.Printf("Antes  → HeapAlloc=%.1f MB  TotalAlloc=%.1f MB  GC=%d\n",
		resBefore.HeapAllocMB, resBefore.TotalAllocMB, resBefore.GCCycles)
	fmt.Printf("Después → HeapAlloc=%.1f MB  TotalAlloc=%.1f MB  GC=%d\n",
		resAfter.HeapAllocMB, resAfter.TotalAllocMB, resAfter.GCCycles)

	counts := map[string]int{}
	for _, f := range flags {
		counts[f.FlagType]++
	}
	fmt.Printf("\n=== Red Flags Detectadas ===\n")
	fmt.Printf("Total: %d\n", len(flags))
	fmt.Printf("  TEXT_REPEAT:      %d\n", counts[detector.FlagTextRepeat])
	fmt.Printf("  TIMING_BURST:     %d\n", counts[detector.FlagTimingBurst])
	fmt.Printf("  EXACT_DUPLICATE:  %d\n", counts[detector.FlagExactDuplicate])

	if err := writeFlags(*output, flags); err != nil {
		log.Fatalf("error escribiendo output: %v", err)
	}
	fmt.Printf("\nResultados escritos en %s\n", *output)
}

func writeFlags(path string, flags []detector.RedFlag) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	_ = w.Write([]string{"record_index", "expediente", "dataset_id", "flag_type", "score", "details", "queja_preview"})
	for _, fl := range flags {
		_ = w.Write([]string{
			strconv.Itoa(fl.RecordIndex),
			fl.Expediente,
			fl.DatasetID,
			fl.FlagType,
			strconv.FormatFloat(fl.Score, 'f', 4, 64),
			fl.Details,
			fl.QuejaPreview,
		})
	}
	return nil
}
