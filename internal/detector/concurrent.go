package detector

import (
	"fmt"
	"time"
)

// DetectConcurrent ejecuta la detección con TEXT_REPEAT keyword-index.
func DetectConcurrent(records []DetectorRecord, opts Options, numWorkers int) ([]RedFlag, time.Duration) {
	return DetectConcurrentWithVocab(records, nil, opts, numWorkers)
}

func DetectConcurrentWithVocab(records []DetectorRecord, vocab *Vocab, opts Options, numWorkers int) ([]RedFlag, time.Duration) {
	start := time.Now()
	opts = normalizeOptions(opts)
	n := len(records)

	// --- Fase 1: TEXT_REPEAT con keyword-index ---
	flags := make([]RedFlag, 0)
	if !opts.DisableTextRepeat {
		idx, stats := BuildKeywordIndex(records, vocab, opts)
		_ = stats
		trFlags, _, _ := detectTextRepeatKeywordIndexFlags(records, idx, opts)
		flags = append(flags, trFlags...)
	}

	// --- Fase 2: TIMING_BURST — O(n) secuencial ---
	dateBuckets := make(map[string][]int, n/10)
	for i, r := range records {
		if r.FechaPresentacion != "" {
			dateBuckets[r.FechaPresentacion] = append(dateBuckets[r.FechaPresentacion], i)
		}
	}
	for fecha, indices := range dateBuckets {
		if len(indices) > opts.BurstLimit {
			for _, idx := range indices {
				flags = append(flags, RedFlag{
					RecordIndex:  idx,
					Expediente:   records[idx].Expediente,
					DatasetID:    records[idx].DatasetID,
					FlagType:     FlagTimingBurst,
					Score:        float64(len(indices)),
					Details:      fmt.Sprintf("%d expedientes en fecha %s", len(indices), fecha),
					QuejaPreview: quejaPreview(records[idx].Queja),
				})
			}
		}
	}

	// --- Fase 3: EXACT_DUPLICATE — O(n) secuencial ---
	seen := make(map[string]int, n)
	for i, r := range records {
		key := r.Expediente + "|" + r.Denunciado + "|" + r.FechaPresentacion
		if key == "||" {
			continue
		}
		if prev, ok := seen[key]; ok {
			flags = append(flags, RedFlag{
				RecordIndex:  prev,
				Expediente:   records[prev].Expediente,
				DatasetID:    records[prev].DatasetID,
				FlagType:     FlagExactDuplicate,
				Score:        1.0,
				Details:      fmt.Sprintf("duplicado con índice %d", i),
				QuejaPreview: quejaPreview(records[prev].Queja),
			})
			flags = append(flags, RedFlag{
				RecordIndex:  i,
				Expediente:   r.Expediente,
				DatasetID:    r.DatasetID,
				FlagType:     FlagExactDuplicate,
				Score:        1.0,
				Details:      fmt.Sprintf("duplicado con índice %d", prev),
				QuejaPreview: quejaPreview(r.Queja),
			})
		} else {
			seen[key] = i
		}
	}

	// --- Fase 4: KEYWORD_SPAM --- O(n·k), misma lógica que secuencial
	flags = append(flags, DetectKeywordSpam(records, opts)...)

	return flags, time.Since(start)
}
