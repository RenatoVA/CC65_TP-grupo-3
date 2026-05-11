package detector

import (
	"fmt"
	"time"
)

// Options configura los umbrales del detector.
type Options struct {
	JaccardThreshold float64 // similitud mínima para marcar TEXT_REPEAT (default 0.60)
	BurstLimit       int     // expedientes por fecha para marcar TIMING_BURST (default 5)
	KeywordThreshold int     // keywords mínimas para marcar KEYWORD_SPAM (default 1)

	DisableTextRepeat bool // desactiva fase TEXT_REPEAT (bench seguro)

	MinKeywordDocFreq      int     // df mínima para keyword-index
	MaxKeywordDocFreqRatio float64 // df máxima como ratio de N (ej 0.05)
	MinSharedKeywords      int     // mínimo de keywords compartidas para comparar con Jaccard
	MinTokenLength         int     // longitud mínima de token (aplica al vocab)
	MaxCandidatesPerRecord int     // keyword-index: máximo de candidatos verificados por registro (0=sin tope)

	UsePrefixFilter     bool // keyword-index: indexar solo tokens del prefijo (PPJoin-style)
	UseLengthFilter     bool // keyword-index: filtro por ratio de longitudes antes de Jaccard
	UseUpperBoundFilter bool // keyword-index: filtro por cota superior (posición/prefijo) antes de Jaccard

	MinRareScore        uint32 // keyword-index: descartar candidatos con rare score < este valor
	MaxPrefixTokens     int    // keyword-index: cap de tokens de prefijo por registro (0=sin tope)
	MaxPostingsPerToken int    // keyword-index: cap de postings por token (0=sin tope)
}

// DefaultOptions retorna la configuración estándar del detector para 1M registros.
func DefaultOptions() Options {
	return Options{
		JaccardThreshold:       0.60,
		BurstLimit:             5,
		KeywordThreshold:       1,
		MinKeywordDocFreq:      4,
		MaxKeywordDocFreqRatio: 0.003,
		MinSharedKeywords:      7,
		MinTokenLength:         6,
		MaxCandidatesPerRecord: 10,
		UsePrefixFilter:        true,
		UseLengthFilter:        true,
		UseUpperBoundFilter:    true,
	}
}

func normalizeOptions(opts Options) Options {
	defaults := DefaultOptions()
	if opts.JaccardThreshold <= 0 {
		opts.JaccardThreshold = defaults.JaccardThreshold
	}
	if opts.BurstLimit <= 0 {
		opts.BurstLimit = defaults.BurstLimit
	}
	if opts.KeywordThreshold <= 0 {
		opts.KeywordThreshold = defaults.KeywordThreshold
	}
	if opts.MinKeywordDocFreq <= 0 {
		opts.MinKeywordDocFreq = defaults.MinKeywordDocFreq
	}
	if opts.MaxKeywordDocFreqRatio <= 0 {
		opts.MaxKeywordDocFreqRatio = defaults.MaxKeywordDocFreqRatio
	}
	if opts.MinSharedKeywords <= 0 {
		opts.MinSharedKeywords = defaults.MinSharedKeywords
	}
	if opts.MinTokenLength <= 0 {
		opts.MinTokenLength = defaults.MinTokenLength
	}
	if opts.MaxCandidatesPerRecord <= 0 {
		opts.MaxCandidatesPerRecord = defaults.MaxCandidatesPerRecord
	}
	opts.UsePrefixFilter = true
	opts.UseLengthFilter = true
	opts.UseUpperBoundFilter = true
	return opts
}

func StandardTextRepeatSummary(opts Options) string {
	opts = normalizeOptions(opts)
	return fmt.Sprintf(
		"keyword-index min-df=%d max-df-ratio=%.6f min-shared=%d min-len=%d max-candidates-per-record=%d prefix=%t length-filter=%t upper-bound=%t",
		opts.MinKeywordDocFreq,
		opts.MaxKeywordDocFreqRatio,
		opts.MinSharedKeywords,
		opts.MinTokenLength,
		opts.MaxCandidatesPerRecord,
		opts.UsePrefixFilter,
		opts.UseLengthFilter,
		opts.UseUpperBoundFilter,
	)
}

// DetectSequential ejecuta las tres fases de detección en un único goroutine.
// Retorna las red flags detectadas y el tiempo total de procesamiento.
func DetectSequential(records []DetectorRecord, opts Options) ([]RedFlag, time.Duration) {
	return DetectSequentialWithVocab(records, nil, opts)
}

func DetectSequentialWithVocab(records []DetectorRecord, vocab *Vocab, opts Options) ([]RedFlag, time.Duration) {
	start := time.Now()
	opts = normalizeOptions(opts)
	n := len(records)
	var flags []RedFlag

	// --- Fase 1: TEXT_REPEAT — keyword-index + filtros + Jaccard final ---
	if !opts.DisableTextRepeat {
		idx, stats := BuildKeywordIndex(records, vocab, opts)
		_ = stats
		trFlags, _, _ := detectTextRepeatKeywordIndexFlags(records, idx, opts)
		flags = append(flags, trFlags...)
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

	// --- Fase 4: KEYWORD_SPAM — O(n·k) escaneo de palabras clave sospechosas ---
	// Detecta quejas que contienen lenguaje de spam, phishing, markdown o texto
	// auto-generado por plantillas. Independiente del Jaccard.
	flags = append(flags, DetectKeywordSpam(records, opts)...)

	return flags, time.Since(start)
}
