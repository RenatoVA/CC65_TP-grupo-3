package detector

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
)

// DetectorRecord es la representación plana de una fila del CSV enriquecido.
type DetectorRecord struct {
	Index             int
	DatasetID         string
	Expediente        string
	TipoExpediente    string
	FechaPresentacion string
	Denunciante       string
	Denunciado        string
	RUC               string
	Materia           string
	Queja             string
	Tokens            []string // tokens como strings (compatibilidad)
	TokenIDs          []uint32 // tokens como IDs numéricos para JaccardUint32
}

// LoadCSVConcurrent carga el CSV usando un pipeline de 3 etapas concurrentes.
//
// Etapa 1 — Reader (1 goroutine, I/O bound):
//
//	Lee filas del CSV y las envía a rawCh sin tokenizar.
//
// Etapa 2 — Worker Pool (numWorkers goroutines, CPU bound):
//
//	Cada worker recibe una fila indexada, llama TokenizeToIDs con el vocab
//	compartido (protegido por RWMutex) y envía el registro completo a outCh.
//
// Etapa 3 — Collector (goroutine principal):
//
//	Coloca cada registro en su posición exacta del slice pre-allocado usando
//	el índice original — sin mutex porque cada goroutine escribe en su propio slot.
//
// Retorna el slice ordenado y el Vocab construido para el indice keyword-index.
func LoadCSVConcurrent(path string, numWorkers int) ([]DetectorRecord, *Vocab, error) {
	// Contar filas primero para pre-allocar el slice
	total, err := countCSVRows(path)
	if err != nil {
		return nil, nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("abrir %s: %w", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.LazyQuotes = true
	r.FieldsPerRecord = -1
	if _, err := r.Read(); err != nil { // descartar header
		return nil, nil, fmt.Errorf("leer header: %w", err)
	}

	type rawItem struct {
		idx int
		row []string
	}
	type doneItem struct {
		idx int
		rec DetectorRecord
	}

	rawCh := make(chan rawItem, numWorkers*8)
	outCh := make(chan doneItem, numWorkers*8)
	vocab := NewVocab()
	records := make([]DetectorRecord, total)

	// Etapa 2: Worker Pool de tokenizadores
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range rawCh {
				get := func(i int) string {
					if i < len(item.row) {
						return strings.TrimSpace(item.row[i])
					}
					return ""
				}
				rec := DetectorRecord{
					Index:             item.idx,
					DatasetID:         get(0),
					Expediente:        get(3),
					TipoExpediente:    get(4),
					FechaPresentacion: get(5),
					Denunciante:       get(6),
					Denunciado:        get(7),
					RUC:               get(8),
					Materia:           get(9),
					Queja:             get(12),
				}
				rec.TokenIDs = TokenizeToIDs(rec.Queja, vocab)
				outCh <- doneItem{item.idx, rec}
			}
		}()
	}

	// Cierra outCh cuando todos los workers terminan
	go func() { wg.Wait(); close(outCh) }()

	// Etapa 1: Reader — envía filas indexadas a rawCh
	go func() {
		idx := 0
		for {
			row, err := r.Read()
			if err != nil {
				break
			}
			rawCh <- rawItem{idx, row}
			idx++
		}
		close(rawCh)
	}()

	// Etapa 3: Collector — coloca cada registro en su slot sin mutex
	for item := range outCh {
		records[item.idx] = item.rec
	}

	return records, vocab, nil
}

// LoadCSVConcurrentLimit es igual a LoadCSVConcurrent pero permite limitar la cantidad
// de filas procesadas (primeras N filas del CSV, sin contar el header).
// Si limit <= 0, procesa todo el archivo.
func LoadCSVConcurrentLimit(path string, numWorkers int, limit int) ([]DetectorRecord, *Vocab, error) {
	total := 0
	if limit > 0 {
		total = limit
	} else {
		// Contar filas primero para pre-allocar el slice
		n, err := countCSVRows(path)
		if err != nil {
			return nil, nil, err
		}
		total = n
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("abrir %s: %w", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.LazyQuotes = true
	r.FieldsPerRecord = -1
	if _, err := r.Read(); err != nil { // descartar header
		return nil, nil, fmt.Errorf("leer header: %w", err)
	}

	type rawItem struct {
		idx int
		row []string
	}
	type doneItem struct {
		idx int
		rec DetectorRecord
	}

	rawCh := make(chan rawItem, numWorkers*8)
	outCh := make(chan doneItem, numWorkers*8)
	vocab := NewVocab()
	records := make([]DetectorRecord, total)

	// Etapa 2: Worker Pool de tokenizadores
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range rawCh {
				get := func(i int) string {
					if i < len(item.row) {
						return strings.TrimSpace(item.row[i])
					}
					return ""
				}
				rec := DetectorRecord{
					Index:             item.idx,
					DatasetID:         get(0),
					Expediente:        get(3),
					TipoExpediente:    get(4),
					FechaPresentacion: get(5),
					Denunciante:       get(6),
					Denunciado:        get(7),
					RUC:               get(8),
					Materia:           get(9),
					Queja:             get(12),
				}
				rec.TokenIDs = TokenizeToIDs(rec.Queja, vocab)
				outCh <- doneItem{item.idx, rec}
			}
		}()
	}

	// Cierra outCh cuando todos los workers terminan
	go func() { wg.Wait(); close(outCh) }()

	// Etapa 1: Reader — envía filas indexadas a rawCh
	go func() {
		idx := 0
		for {
			if limit > 0 && idx >= limit {
				break
			}
			row, err := r.Read()
			if err != nil {
				break
			}
			rawCh <- rawItem{idx, row}
			idx++
		}
		close(rawCh)
	}()

	// Etapa 3: Collector — coloca cada registro en su slot sin mutex
	received := 0
	for item := range outCh {
		if item.idx >= 0 && item.idx < total {
			records[item.idx] = item.rec
			received++
		}
	}

	if limit > 0 {
		// Si el archivo tiene menos filas que el límite, recortar al recibido real.
		// Ojo: si se corta por EOF antes del límite, received puede ser < total.
		// Como los índices se generan secuencialmente desde 0, recortar es seguro.
		if received < total {
			records = records[:received]
		}
	}

	return records, vocab, nil
}

// LoadCSVConcurrentWithVocab es un alias explícito de LoadCSVConcurrent.
func LoadCSVConcurrentWithVocab(path string, numWorkers int) ([]DetectorRecord, *Vocab, error) {
	return LoadCSVConcurrent(path, numWorkers)
}

// LoadCSVConcurrentLimitWithVocab es un alias explícito de LoadCSVConcurrentLimit.
func LoadCSVConcurrentLimitWithVocab(path string, numWorkers int, limit int) ([]DetectorRecord, *Vocab, error) {
	return LoadCSVConcurrentLimit(path, numWorkers, limit)
}

func countCSVRows(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.LazyQuotes = true
	r.FieldsPerRecord = -1
	r.Read() // header
	count := 0
	for {
		if _, err := r.Read(); err != nil {
			break
		}
		count++
	}
	return count, nil
}

// LoadCSV carga el CSV enriquecido y pre-tokeniza el campo queja de cada fila.
func LoadCSV(path string) ([]DetectorRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("abrir %s: %w", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.LazyQuotes = true
	r.FieldsPerRecord = -1

	// descartar header
	if _, err := r.Read(); err != nil {
		return nil, fmt.Errorf("leer header: %w", err)
	}

	vocab := NewVocab()
	var records []DetectorRecord
	idx := 0
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue // fila malformada, saltar
		}

		get := func(i int) string {
			if i < len(row) {
				return strings.TrimSpace(row[i])
			}
			return ""
		}

		sourceRow, _ := strconv.Atoi(get(2))
		_ = sourceRow

		rec := DetectorRecord{
			Index:             idx,
			DatasetID:         get(0),
			Expediente:        get(3),
			TipoExpediente:    get(4),
			FechaPresentacion: get(5),
			Denunciante:       get(6),
			Denunciado:        get(7),
			RUC:               get(8),
			Materia:           get(9),
			Queja:             get(12),
		}
		rec.TokenIDs = TokenizeToIDs(rec.Queja, vocab)
		records = append(records, rec)
		idx++
	}
	return records, nil
}

// LoadCSVWithVocab es igual a LoadCSV pero retorna también el vocab construido.
// Útil para estrategias de TEXT_REPEAT que necesitan metadatos globales del vocab.
func LoadCSVWithVocab(path string) ([]DetectorRecord, *Vocab, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("abrir %s: %w", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.LazyQuotes = true
	r.FieldsPerRecord = -1

	// descartar header
	if _, err := r.Read(); err != nil {
		return nil, nil, fmt.Errorf("leer header: %w", err)
	}

	vocab := NewVocab()
	var records []DetectorRecord
	idx := 0
	for {
		row, err := r.Read()
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

		rec := DetectorRecord{
			Index:             idx,
			DatasetID:         get(0),
			Expediente:        get(3),
			TipoExpediente:    get(4),
			FechaPresentacion: get(5),
			Denunciante:       get(6),
			Denunciado:        get(7),
			RUC:               get(8),
			Materia:           get(9),
			Queja:             get(12),
		}
		rec.TokenIDs = TokenizeToIDs(rec.Queja, vocab)
		records = append(records, rec)
		idx++
	}
	return records, vocab, nil
}

// LoadCSVLimitWithVocab es igual a LoadCSVLimit pero retorna también el vocab construido.
func LoadCSVLimitWithVocab(path string, limit int) ([]DetectorRecord, *Vocab, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("abrir %s: %w", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.LazyQuotes = true
	r.FieldsPerRecord = -1

	// descartar header
	if _, err := r.Read(); err != nil {
		return nil, nil, fmt.Errorf("leer header: %w", err)
	}

	vocab := NewVocab()
	capacity := 0
	if limit > 0 {
		capacity = limit
	}
	records := make([]DetectorRecord, 0, capacity)
	idx := 0
	for {
		if limit > 0 && idx >= limit {
			break
		}
		row, err := r.Read()
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

		rec := DetectorRecord{
			Index:             idx,
			DatasetID:         get(0),
			Expediente:        get(3),
			TipoExpediente:    get(4),
			FechaPresentacion: get(5),
			Denunciante:       get(6),
			Denunciado:        get(7),
			RUC:               get(8),
			Materia:           get(9),
			Queja:             get(12),
		}
		rec.TokenIDs = TokenizeToIDs(rec.Queja, vocab)
		records = append(records, rec)
		idx++
	}
	return records, vocab, nil
}

// LoadCSVLimit es igual a LoadCSV pero permite limitar la cantidad de filas
// procesadas (primeras N filas del CSV, sin contar el header).
// Si limit <= 0, procesa todo el archivo.
func LoadCSVLimit(path string, limit int) ([]DetectorRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("abrir %s: %w", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.LazyQuotes = true
	r.FieldsPerRecord = -1

	// descartar header
	if _, err := r.Read(); err != nil {
		return nil, fmt.Errorf("leer header: %w", err)
	}

	vocab := NewVocab()
	capacity := 0
	if limit > 0 {
		capacity = limit
	}
	records := make([]DetectorRecord, 0, capacity)
	idx := 0
	for {
		if limit > 0 && idx >= limit {
			break
		}
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue // fila malformada, saltar
		}

		get := func(i int) string {
			if i < len(row) {
				return strings.TrimSpace(row[i])
			}
			return ""
		}

		rec := DetectorRecord{
			Index:             idx,
			DatasetID:         get(0),
			Expediente:        get(3),
			TipoExpediente:    get(4),
			FechaPresentacion: get(5),
			Denunciante:       get(6),
			Denunciado:        get(7),
			RUC:               get(8),
			Materia:           get(9),
			Queja:             get(12),
		}
		rec.TokenIDs = TokenizeToIDs(rec.Queja, vocab)
		records = append(records, rec)
		idx++
	}
	return records, nil
}
