package detector

import (
	"fmt"
	"sync"
	"time"
)

// DetectConcurrent ejecuta la detección usando un Worker Pool para la fase O(n²).
//
// Patrón: Worker Pool sobre la Fase 1 (TEXT_REPEAT).
//   - Un productor envía valores de i (índice de fila fuente) al canal workCh.
//   - Cada worker recibe un i y computa todos los pares (i, j>i) localmente,
//     acumulando flags en un slice local SIN mutex.
//   - Al agotar workCh, el worker envía su slice local de una sola vez a resultCh.
//   - Un WaitGroup señala al goroutine supervisor cuándo cerrar resultCh.
//   - El goroutine principal recolecta todos los slices locales de resultCh.
//
// Ventajas frente a mutex global:
//   - Cero contención durante el cómputo: cada worker trabaja en memoria propia.
//   - La única escritura concurrente ocurre UNA vez por worker (envío final a resultCh).
//   - No hay deadlocks posibles: canales son la única primitiva de sincronización.
//   - El canal buffered workCh absorbe la diferencia de velocidad entre productor y workers.
func DetectConcurrent(records []DetectorRecord, opts Options, numWorkers int) ([]RedFlag, time.Duration) {
	start := time.Now()
	n := len(records)

	// workCh distribuye los índices i a los workers.
	// Buffer = numWorkers*4 para que el productor no bloquee constantemente.
	workCh := make(chan int, numWorkers*4)

	// resultCh recibe el slice de flags local de cada worker al terminar.
	// Buffer = numWorkers para que los workers no bloqueen al enviar su resultado final.
	resultCh := make(chan []RedFlag, numWorkers)

	var wg sync.WaitGroup

	// --- Lanzar workers ---
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var local []RedFlag // acumulación local: sin mutex, sin contención

			for i := range workCh {
				for j := i + 1; j < n; j++ {
					score := Jaccard(records[i].Tokens, records[j].Tokens)
					if score >= opts.JaccardThreshold {
						detail := fmt.Sprintf("similar a expediente %s (score=%.2f)", records[j].Expediente, score)
						local = append(local, RedFlag{
							RecordIndex:  i,
							Expediente:   records[i].Expediente,
							DatasetID:    records[i].DatasetID,
							FlagType:     FlagTextRepeat,
							Score:        score,
							Details:      detail,
							QuejaPreview: quejaPreview(records[i].Queja),
						})
						detail2 := fmt.Sprintf("similar a expediente %s (score=%.2f)", records[i].Expediente, score)
						local = append(local, RedFlag{
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

			resultCh <- local // una sola escritura al canal por worker
		}()
	}

	// --- Productor: envía valores de i al canal ---
	// Goroutine separada para no bloquear el goroutine principal.
	go func() {
		for i := 0; i < n; i++ {
			workCh <- i
		}
		close(workCh) // señal a los workers de que no hay más trabajo
	}()

	// --- Supervisor: cierra resultCh cuando todos los workers terminaron ---
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// --- Recolector: goroutine principal recibe todos los slices locales ---
	var flags []RedFlag
	for local := range resultCh {
		flags = append(flags, local...)
	}

	// --- Fases 2 y 3 son O(n): se ejecutan secuencialmente ---
	// El overhead de paralelizarlas sería mayor que el beneficio para n=10k.

	// Fase 2: TIMING_BURST
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

	// Fase 3: EXACT_DUPLICATE
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

	return flags, time.Since(start)
}
