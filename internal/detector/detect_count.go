package detector

import (
	"strings"
	"sync/atomic"
	"time"
)

// CountResult resume el trabajo de detección sin materializar []RedFlag.
// Útil para benchmarks grandes donde almacenar cada flag es prohibitivo.
type CountResult struct {
	TotalFlags uint64

	TextRepeatFlags                     uint64
	TextRepeatComparisons               uint64
	TextRepeatTruncated                 bool
	TextRepeatCandidatePairs            uint64
	TextRepeatCandidatesSkipped         uint64
	TextRepeatCandidatesSkippedByLength uint64
	TextRepeatCandidatesSkippedByUpper  uint64
	TextRepeatCandidatesTouched         uint64
	TextRepeatCandidatesSelected        uint64
	TextRepeatCandidatesSkippedByRare   uint64
	TextRepeatVocabSize                 int
	TextRepeatKeywordCount              int
	TextRepeatInvertedPostings          int
	TextRepeatPrefixPostings            int
	TextRepeatDurationMS                int64
	TextRepeatPostingsTruncatedTokens   uint64
	TextRepeatPostingsSkippedEntries    uint64
	TextRepeatJaccardCalls              uint64
	TextRepeatJaccardEarlyExits         uint64
	TextRepeatAcceptedPairs             uint64

	TimingBurstFlags    uint64
	ExactDuplicateFlags uint64
	KeywordSpamFlags    uint64
}

// CountProgress permite observar progreso desde el caller.
// Si los punteros son nil, no se reporta nada.
type CountProgress struct {
	Comparisons *uint64 // pares TEXT_REPEAT verificados con Jaccard exacto
	Flags       *uint64 // flags acumuladas (todas las fases)
}

func addUint64(ptr *uint64, delta uint64) {
	if ptr == nil {
		return
	}
	atomic.AddUint64(ptr, delta)
}

// DetectSequentialCount ejecuta la misma lógica que DetectSequential pero solo cuenta flags.
func DetectSequentialCount(records []DetectorRecord, opts Options, progress *CountProgress) (CountResult, time.Duration) {
	return DetectSequentialCountWithVocab(records, nil, opts, progress)
}

func DetectSequentialCountWithVocab(records []DetectorRecord, vocab *Vocab, opts Options, progress *CountProgress) (CountResult, time.Duration) {
	start := time.Now()
	if progress == nil {
		progress = &CountProgress{}
	}
	opts = normalizeOptions(opts)
	n := len(records)
	var res CountResult

	// Fase 1: TEXT_REPEAT (keyword-index + Jaccard final)
	if !opts.DisableTextRepeat {
		idx, stats := BuildKeywordIndex(records, vocab, opts)
		trStats, trDur := detectTextRepeatKeywordIndexCount(records, idx, opts, progress)
		fillTextRepeatCountResult(&res, stats, trStats, trDur)
	}

	// Fase 2: TIMING_BURST
	dateBuckets := make(map[string][]int, n/10)
	for i, r := range records {
		if r.FechaPresentacion != "" {
			dateBuckets[r.FechaPresentacion] = append(dateBuckets[r.FechaPresentacion], i)
		}
	}
	for _, indices := range dateBuckets {
		if len(indices) > opts.BurstLimit {
			inc := uint64(len(indices))
			res.TimingBurstFlags += inc
			res.TotalFlags += inc
			addUint64(progress.Flags, inc)
		}
	}

	// Fase 3: EXACT_DUPLICATE
	seen := make(map[string]int, n)
	for i, r := range records {
		key := r.Expediente + "|" + r.Denunciado + "|" + r.FechaPresentacion
		if key == "||" {
			continue
		}
		if _, ok := seen[key]; ok {
			// Misma semántica que DetectSequential: un duplicado produce 2 flags (prev + i)
			res.ExactDuplicateFlags += 2
			res.TotalFlags += 2
			addUint64(progress.Flags, 2)
		} else {
			seen[key] = i
		}
	}

	// Fase 4: KEYWORD_SPAM
	kwFlags := detectKeywordSpamCount(records, opts)
	res.KeywordSpamFlags = kwFlags
	res.TotalFlags += kwFlags
	addUint64(progress.Flags, kwFlags)

	return res, time.Since(start)
}

// DetectConcurrentCount ejecuta la misma lógica que DetectConcurrent pero solo cuenta flags.
func DetectConcurrentCount(records []DetectorRecord, opts Options, numWorkers int, progress *CountProgress) (CountResult, time.Duration) {
	return DetectConcurrentCountWithVocab(records, nil, opts, numWorkers, progress)
}

func DetectConcurrentCountWithVocab(records []DetectorRecord, vocab *Vocab, opts Options, numWorkers int, progress *CountProgress) (CountResult, time.Duration) {
	start := time.Now()
	if progress == nil {
		progress = &CountProgress{}
	}
	opts = normalizeOptions(opts)
	n := len(records)
	var res CountResult

	// Fase 1: TEXT_REPEAT
	if !opts.DisableTextRepeat {
		idx, stats := BuildKeywordIndex(records, vocab, opts)
		trStats, trDur := detectTextRepeatKeywordIndexCountConcurrent(records, idx, opts, numWorkers, progress)
		fillTextRepeatCountResult(&res, stats, trStats, trDur)
	}

	// Fase 2: TIMING_BURST
	dateBuckets := make(map[string][]int, n/10)
	for i, r := range records {
		if r.FechaPresentacion != "" {
			dateBuckets[r.FechaPresentacion] = append(dateBuckets[r.FechaPresentacion], i)
		}
	}
	for _, indices := range dateBuckets {
		if len(indices) > opts.BurstLimit {
			inc := uint64(len(indices))
			res.TimingBurstFlags += inc
			res.TotalFlags += inc
			addUint64(progress.Flags, inc)
		}
	}

	// Fase 3: EXACT_DUPLICATE
	seen := make(map[string]int, n)
	for i, r := range records {
		key := r.Expediente + "|" + r.Denunciado + "|" + r.FechaPresentacion
		if key == "||" {
			continue
		}
		if _, ok := seen[key]; ok {
			res.ExactDuplicateFlags += 2
			res.TotalFlags += 2
			addUint64(progress.Flags, 2)
		} else {
			seen[key] = i
		}
	}

	// Fase 4: KEYWORD_SPAM
	kwFlags := detectKeywordSpamCount(records, opts)
	res.KeywordSpamFlags = kwFlags
	res.TotalFlags += kwFlags
	addUint64(progress.Flags, kwFlags)

	return res, time.Since(start)
}

func fillTextRepeatCountResult(res *CountResult, idxStats TextRepeatStats, trStats TextRepeatStats, trDur time.Duration) {
	res.TextRepeatFlags = trStats.FlagsDetected
	res.TextRepeatComparisons = trStats.ComparisonsVerified
	res.TextRepeatTruncated = trStats.Truncated
	res.TextRepeatCandidatePairs = trStats.CandidatePairsGenerated
	res.TextRepeatCandidatesSkipped = trStats.CandidatesSkippedByCap
	res.TextRepeatCandidatesSkippedByLength = trStats.CandidatesSkippedByLength
	res.TextRepeatCandidatesSkippedByUpper = trStats.CandidatesSkippedByUpper
	res.TextRepeatCandidatesTouched = trStats.CandidatesTouched
	res.TextRepeatCandidatesSelected = trStats.CandidatesSelected
	res.TextRepeatCandidatesSkippedByRare = trStats.CandidatesSkippedByRare
	res.TextRepeatVocabSize = idxStats.TotalVocabSize
	res.TextRepeatKeywordCount = idxStats.SelectedKeywordCnt
	res.TextRepeatInvertedPostings = idxStats.InvertedPostings
	res.TextRepeatPrefixPostings = idxStats.PrefixPostings
	res.TextRepeatPostingsTruncatedTokens = idxStats.PostingsTruncatedTokens
	res.TextRepeatPostingsSkippedEntries = idxStats.PostingsSkippedEntries
	res.TextRepeatDurationMS = trDur.Milliseconds()
	res.TextRepeatJaccardCalls = trStats.JaccardCalls
	res.TextRepeatJaccardEarlyExits = trStats.JaccardEarlyExits
	res.TextRepeatAcceptedPairs = trStats.AcceptedPairs
	res.TotalFlags += trStats.FlagsDetected
}

func detectKeywordSpamCount(records []DetectorRecord, opts Options) uint64 {
	threshold := opts.KeywordThreshold
	if threshold <= 0 {
		threshold = 1
	}

	var flags uint64
	for _, r := range records {
		queja := strings.ToLower(r.Queja)
		matched := 0
		for _, kw := range spamKeywords {
			if strings.Contains(queja, kw) {
				matched++
			}
		}
		if matched >= threshold {
			flags++
		}
	}
	return flags
}
