package detector

import (
	"fmt"
	"time"
)

// Options configura los umbrales del detector.
type Options struct {
	JaccardThreshold float64 // similitud mínima para marcar TEXT_REPEAT (default 0.60)
	BurstLimit       int     // expedientes por fecha para marcar TIMING_BURST (default 5)
}

// DefaultOptions retorna la configuración por defecto.
func DefaultOptions() Options {
	return Options{JaccardThreshold: 0.60, BurstLimit: 5}
}

// DetectSequential ejecuta las tres fases de detección en un único goroutine.
// Retorna las red flags detectadas y el tiempo total de procesamiento.
func DetectSequential(records []DetectorRecord, opts Options) ([]RedFlag, time.Duration) {
	start := time.Now()
	n := len(records)
	var flags []RedFlag

	// --- Fase 1: TEXT_REPEAT — O(n²) comparación de todos los pares ---
	// Para cada par (i,j) calcula Jaccard sobre tokens pre-computados.
	// Si la similitud supera el umbral, ambos registros se marcan como sospechosos.
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			score := Jaccard(records[i].Tokens, records[j].Tokens)
			if score >= opts.JaccardThreshold {
				detail := fmt.Sprintf("similar a expediente %s (score=%.2f)", records[j].Expediente, score)
				flags = append(flags, RedFlag{
					RecordIndex:  i,
					Expediente:   records[i].Expediente,
					DatasetID:    records[i].DatasetID,
					FlagType:     FlagTextRepeat,
					Score:        score,
					Details:      detail,
					QuejaPreview: quejaPreview(records[i].Queja),
				})
				detail2 := fmt.Sprintf("similar a expediente %s (score=%.2f)", records[i].Expediente, score)
				flags = append(flags, RedFlag{
					RecordIndex:  j,
					Expediente:   records[j].Expediente,
					DatasetID:    records[j].DatasetID,
					FlagType:     FlagTextRepeat,
					Score:        score,
					Details:      detail2,
					QuejaPreview: quejaPreview(records[j].Queja),
				})
			}
		}
	}

	// --- Fase 2: TIMING_BURST — O(n) agrupación por fecha ---
	// Agrupa los expedientes por fecha de presentación.
	// Si una fecha supera el límite configurado, todos sus registros son flaggeados.
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

	// --- Fase 3: EXACT_DUPLICATE — O(n) con tabla hash ---
	// Construye una clave compuesta por expediente+denunciado+fecha.
	// Si la clave ya existe, ambos registros son marcados como duplicados exactos.
	seen := make(map[string]int, n)
	for i, r := range records {
		key := r.Expediente + "|" + r.Denunciado + "|" + r.FechaPresentacion
		if key == "||" {
			continue // clave vacía, no es duplicado significativo
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

	return flags, time.Since(start)
}
