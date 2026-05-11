// benchfull: benchmark oficial del detector de red flags.
//
// Mide en una sola ejecucion:
//   - Carga secuencial vs carga concurrente (pipeline)
//   - Deteccion secuencial vs concurrente para cada configuracion de workers
//   - Tabla de speedup con media recortada
//   - Uso de CPU y memoria por version
//
// La deteccion usa siempre la norma keyword-index del paquete detector.
package main

import (
	"flag"
	"fmt"
	"log"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"tp-programacion-concurente/internal/detector"
)

func main() {
	input := flag.String("input", "data/processed/enriched_filled.csv", "CSV de entrada")
	workersList := flag.String("workers-list", "2,4,8", "Workers a probar, separados por coma")
	runs := flag.Int("runs", 5, "Corridas por configuracion (para media recortada)")
	detectSample := flag.Int("detect-sample", 0, "Max registros para deteccion (0=todos)")
	limit := flag.Int("limit", 0, "Procesar solo las primeras N filas (0=todas)")
	countOnly := flag.Bool("count-only", false, "Solo contar flags (no almacenar objetos en memoria)")
	progressEvery := flag.Duration("progress-every", 2*time.Second, "Intervalo de logs de progreso (0=desactivar)")
	flag.Parse()

	workerCounts := parseWorkers(*workersList)
	opts := detector.DefaultOptions()

	banner("BENCHMARK COMPLETO - Detector de Red Flags INDECOPI")
	fmt.Printf("  Dataset:       %s\n", *input)
	fmt.Printf("  Corridas:      %d por configuracion\n", *runs)
	fmt.Printf("  Workers:       %v\n", workerCounts)
	fmt.Printf("  TEXT_REPEAT:   %s\n", detector.StandardTextRepeatSummary(opts))
	fmt.Printf("  Umbral:        %.2f Jaccard | Burst: %d | KeywordSpam: %d\n",
		opts.JaccardThreshold, opts.BurstLimit, opts.KeywordThreshold)
	if *limit > 0 {
		fmt.Printf("  Limite carga:  %d filas\n", *limit)
	}
	if *countOnly {
		fmt.Printf("  Modo:          COUNT (sin materializar flags)\n")
	}
	fmt.Printf("  CPUs:          %d\n\n", runtime.NumCPU())

	section("A - Carga del dataset")

	var seqLoadTimes []float64
	var records []detector.DetectorRecord
	var vocab *detector.Vocab

	fmt.Printf("  [Secuencial - LoadCSVWithVocab]\n")
	for r := 1; r <= *runs; r++ {
		t0 := time.Now()
		var (
			recs []detector.DetectorRecord
			vb   *detector.Vocab
			err  error
		)
		if *limit > 0 {
			recs, vb, err = detector.LoadCSVLimitWithVocab(*input, *limit)
		} else {
			recs, vb, err = detector.LoadCSVWithVocab(*input)
		}
		if err != nil {
			log.Fatalf("error cargando: %v", err)
		}
		elapsed := time.Since(t0)
		if r == *runs {
			records = recs
			vocab = vb
		}
		seqLoadTimes = append(seqLoadTimes, ms(elapsed))
		fmt.Printf("    corrida %d/%d: %s  (%d registros)\n", r, *runs, elapsed.Round(time.Millisecond), len(recs))
	}
	seqLoadMean := detector.TrimmedMean(seqLoadTimes, 0.10)
	fmt.Printf("    -> Media recortada: %.0f ms\n\n", seqLoadMean)

	runtime.GC()

	fmt.Printf("  [Concurrente - LoadCSVConcurrentWithVocab pipeline (%d workers)]\n", workerCounts[0])
	var concLoadTimes []float64
	for r := 1; r <= *runs; r++ {
		t0 := time.Now()
		var (
			recs []detector.DetectorRecord
			err  error
		)
		if *limit > 0 {
			recs, _, err = detector.LoadCSVConcurrentLimitWithVocab(*input, workerCounts[0], *limit)
		} else {
			recs, _, err = detector.LoadCSVConcurrentWithVocab(*input, workerCounts[0])
		}
		if err != nil {
			log.Fatalf("error cargando: %v", err)
		}
		elapsed := time.Since(t0)
		n := len(recs)
		recs = nil
		runtime.GC()
		concLoadTimes = append(concLoadTimes, ms(elapsed))
		fmt.Printf("    corrida %d/%d: %s  (%d registros)\n", r, *runs, elapsed.Round(time.Millisecond), n)
	}
	concLoadMean := detector.TrimmedMean(concLoadTimes, 0.10)
	fmt.Printf("    -> Media recortada: %.0f ms\n\n", concLoadMean)
	fmt.Printf("  [Memoria tras carga] %s\n\n", memLine())

	total := len(records)
	sample := records
	if *detectSample > 0 && *detectSample < total {
		sample = records[:*detectSample]
	}
	sampleN := len(sample)

	section(fmt.Sprintf("B - Deteccion sobre %d registros", sampleN))
	if sampleN < total {
		fmt.Printf("  (muestra de %d/%d; usar -detect-sample=0 para todos)\n", sampleN, total)
	}
	fmt.Println()

	fmt.Printf("  [Secuencial]\n")
	resBefSeq := detector.CaptureResources()
	var seqDetTimes []float64
	var seqFlags uint64
	for r := 1; r <= *runs; r++ {
		t0 := time.Now()
		flagsThis := runDetectionSequential(sample, vocab, opts, *countOnly, *progressEvery)
		elapsed := time.Since(t0)
		seqDetTimes = append(seqDetTimes, ms(elapsed))
		seqFlags = flagsThis
		fmt.Printf("    corrida %d/%d: %s  flags=%d\n", r, *runs, elapsed.Round(time.Millisecond), flagsThis)
	}
	resAftSeq := detector.CaptureResources()
	seqDetMean := detector.TrimmedMean(seqDetTimes, 0.10)
	fmt.Printf("    -> Media recortada: %.0f ms  |  flags detectadas: %d\n\n", seqDetMean, seqFlags)
	fmt.Printf("  [Memoria tras deteccion secuencial] %s\n\n", memLine())

	type concResult struct {
		workers int
		mean    float64
		flags   uint64
		resBef  detector.ResourceSnapshot
		resAft  detector.ResourceSnapshot
	}
	var concResults []concResult

	for _, w := range workerCounts {
		fmt.Printf("  [Concurrente - %d workers]\n", w)
		resBef := detector.CaptureResources()
		var times []float64
		var lastFlags uint64
		for r := 1; r <= *runs; r++ {
			t0 := time.Now()
			flagsThis := runDetectionConcurrent(sample, vocab, opts, w, *countOnly, *progressEvery)
			elapsed := time.Since(t0)
			times = append(times, ms(elapsed))
			lastFlags = flagsThis
			fmt.Printf("    corrida %d/%d: %s  flags=%d\n", r, *runs, elapsed.Round(time.Millisecond), flagsThis)
		}
		resAft := detector.CaptureResources()
		mean := detector.TrimmedMean(times, 0.10)
		fmt.Printf("    -> Media recortada: %.0f ms  |  flags detectadas: %d\n\n", mean, lastFlags)
		concResults = append(concResults, concResult{w, mean, lastFlags, resBef, resAft})
	}
	fmt.Printf("  [Memoria tras deteccion concurrente] %s\n\n", memLine())

	section("TABLA DE SPEEDUP")
	fmt.Println()
	fmt.Printf("  Registros cargados:     %d\n", total)
	fmt.Printf("  Registros en deteccion: %d\n\n", sampleN)

	fmt.Printf("  Carga secuencial:       %.0f ms\n", seqLoadMean)
	fmt.Printf("  Carga concurrente x%d:   %.0f ms  speedup=%.3fx\n\n",
		workerCounts[0], concLoadMean, seqLoadMean/concLoadMean)

	fmt.Printf("  Deteccion secuencial:   %.0f ms\n", seqDetMean)
	for _, cr := range concResults {
		speedup := seqDetMean / cr.mean
		efficiency := speedup / float64(cr.workers)
		fmt.Printf("  Deteccion concurrente x%d: %.0f ms  speedup=%.3fx  eficiencia=%.3f\n",
			cr.workers, cr.mean, speedup, efficiency)
	}
	fmt.Println()

	section("USO DE RECURSOS")
	fmt.Println()
	fmt.Printf("  Deteccion secuencial: HeapAlloc=%.1fM TotalAlloc=%.1fM GC=%d\n",
		resAftSeq.HeapAllocMB-resBefSeq.HeapAllocMB,
		resAftSeq.TotalAllocMB-resBefSeq.TotalAllocMB,
		resAftSeq.GCCycles-resBefSeq.GCCycles)
	for _, cr := range concResults {
		fmt.Printf("  Deteccion concurrente x%d: HeapAlloc=%.1fM TotalAlloc=%.1fM GC=%d\n",
			cr.workers,
			cr.resAft.HeapAllocMB-cr.resBef.HeapAllocMB,
			cr.resAft.TotalAllocMB-cr.resBef.TotalAllocMB,
			cr.resAft.GCCycles-cr.resBef.GCCycles)
	}
}

func runDetectionSequential(records []detector.DetectorRecord, vocab *detector.Vocab, opts detector.Options, countOnly bool, progressEvery time.Duration) uint64 {
	if countOnly {
		var comps uint64
		var flags uint64
		progress := detector.CountProgress{Comparisons: &comps, Flags: &flags}
		stop := startProgressLogger(progressEvery, "secuencial", len(records), &comps, &flags)
		res, _ := detector.DetectSequentialCountWithVocab(records, vocab, opts, &progress)
		stop()
		printTextRepeatStats(res)
		return res.TotalFlags
	}
	flags, _ := detector.DetectSequentialWithVocab(records, vocab, opts)
	return uint64(len(flags))
}

func runDetectionConcurrent(records []detector.DetectorRecord, vocab *detector.Vocab, opts detector.Options, workers int, countOnly bool, progressEvery time.Duration) uint64 {
	if countOnly {
		var comps uint64
		var flags uint64
		progress := detector.CountProgress{Comparisons: &comps, Flags: &flags}
		stop := startProgressLogger(progressEvery, fmt.Sprintf("concurrente x%d", workers), len(records), &comps, &flags)
		res, _ := detector.DetectConcurrentCountWithVocab(records, vocab, opts, workers, &progress)
		stop()
		printTextRepeatStats(res)
		return res.TotalFlags
	}
	flags, _ := detector.DetectConcurrentWithVocab(records, vocab, opts, workers)
	return uint64(len(flags))
}

func printTextRepeatStats(res detector.CountResult) {
	fmt.Printf("      [text-repeat keyword-index] vocab=%d keywords=%d postings=%d prefix-postings=%d candidates=%d selected=%d touched=%d comparisons=%d flags=%d tr_ms=%d\n",
		res.TextRepeatVocabSize, res.TextRepeatKeywordCount, res.TextRepeatInvertedPostings, res.TextRepeatPrefixPostings,
		res.TextRepeatCandidatePairs, res.TextRepeatCandidatesSelected, res.TextRepeatCandidatesTouched,
		res.TextRepeatComparisons, res.TextRepeatFlags, res.TextRepeatDurationMS,
	)
	if res.TextRepeatPostingsTruncatedTokens > 0 || res.TextRepeatPostingsSkippedEntries > 0 {
		fmt.Printf("      [text-repeat keyword-index] postings-truncated-tokens=%d postings-skipped-entries=%d\n",
			res.TextRepeatPostingsTruncatedTokens, res.TextRepeatPostingsSkippedEntries)
	}
	if res.TextRepeatCandidatesSkipped > 0 {
		fmt.Printf("      [text-repeat keyword-index] candidates-skipped-by-cap=%d\n", res.TextRepeatCandidatesSkipped)
	}
	if res.TextRepeatCandidatesSkippedByRare > 0 {
		fmt.Printf("      [text-repeat keyword-index] candidates-skipped-by-rare-score=%d\n", res.TextRepeatCandidatesSkippedByRare)
	}
	if res.TextRepeatCandidatesSkippedByLength > 0 {
		fmt.Printf("      [text-repeat keyword-index] candidates-skipped-by-length=%d\n", res.TextRepeatCandidatesSkippedByLength)
	}
	if res.TextRepeatCandidatesSkippedByUpper > 0 {
		fmt.Printf("      [text-repeat keyword-index] candidates-skipped-by-upper-bound=%d\n", res.TextRepeatCandidatesSkippedByUpper)
	}
	fmt.Printf("      [text-repeat keyword-index] jaccard-calls=%d jaccard-early-exits=%d accepted-pairs=%d\n",
		res.TextRepeatJaccardCalls, res.TextRepeatJaccardEarlyExits, res.TextRepeatAcceptedPairs)
}

func parseWorkers(s string) []int {
	var result []int
	for _, p := range strings.Split(s, ",") {
		if n, err := strconv.Atoi(strings.TrimSpace(p)); err == nil && n > 0 {
			result = append(result, n)
		}
	}
	if len(result) == 0 {
		return []int{2, 4, 8}
	}
	return result
}

func ms(d time.Duration) float64 { return float64(d.Milliseconds()) }

func banner(s string) {
	fmt.Printf("\n%s\n%s\n%s\n\n", strings.Repeat("=", len(s)), s, strings.Repeat("=", len(s)))
}

func section(s string) {
	fmt.Printf("=== %s ===\n", s)
}

func startProgressLogger(every time.Duration, label string, records int, comparisons *uint64, flags *uint64) func() {
	if every <= 0 {
		return func() {}
	}
	stopCh := make(chan struct{})
	go func() {
		t := time.NewTicker(every)
		defer t.Stop()
		var ms runtime.MemStats
		for {
			select {
			case <-t.C:
				runtime.ReadMemStats(&ms)
				c := atomic.LoadUint64(comparisons)
				f := atomic.LoadUint64(flags)
				fmt.Printf("      [progreso %s] records=%d comparisons=%d flags=%d HeapAlloc=%.1fMB TotalAlloc=%.1fMB NumGC=%d\n",
					label, records, c, f,
					float64(ms.HeapAlloc)/1024/1024,
					float64(ms.TotalAlloc)/1024/1024,
					ms.NumGC,
				)
			case <-stopCh:
				return
			}
		}
	}()
	return func() { close(stopCh) }
}

func memLine() string {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return fmt.Sprintf("HeapAlloc=%.1fMB TotalAlloc=%.1fMB Sys=%.1fMB NumGC=%d Goroutines=%d",
		float64(ms.HeapAlloc)/1024/1024,
		float64(ms.TotalAlloc)/1024/1024,
		float64(ms.Sys)/1024/1024,
		ms.NumGC,
		runtime.NumGoroutine(),
	)
}
