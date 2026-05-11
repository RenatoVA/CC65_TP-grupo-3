package detector

import (
	"fmt"
	"math"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type TextRepeatStats struct {
	TotalVocabSize          int
	SelectedKeywordCnt      int
	InvertedPostings        int // suma de longitudes de postings (todos los tokens indexados)
	PrefixPostings          int // suma de longitudes de postings del prefijo (si aplica)
	PostingsTruncatedTokens uint64
	PostingsSkippedEntries  uint64

	CandidatePairsGenerated   uint64 // candidatos únicos (shared>=minShared) antes de filtros/cap
	CandidatesTouched         uint64 // candidatos tocados (shared>=1) antes de minShared
	CandidatesSelected        uint64 // candidatos finalmente verificados (tras filtros y cap)
	CandidatesSkippedByCap    uint64
	CandidatesSkippedByRare   uint64
	CandidatesSkippedByLength uint64
	CandidatesSkippedByUpper  uint64

	ComparisonsVerified uint64
	JaccardCalls        uint64
	JaccardEarlyExits   uint64
	AcceptedPairs       uint64
	FlagsDetected       uint64
	Truncated           bool
}

type postEntry struct {
	idx int
	pos uint16 // posición (0-based) en el orden global por DF asc
}

type keywordIndex struct {
	df       map[uint32]uint32
	keywords map[uint32]struct{}    // keywordIDs seleccionadas
	postings map[uint32][]postEntry // token -> postings (solo prefijo si UsePrefixFilter)
	prefix   [][]uint32             // prefix tokens por registro, en orden DF asc
}

// BuildKeywordIndex construye DF global, filtra keywords y crea inverted index.
// Si opts.UsePrefixFilter está activo, indexa solo tokens del prefijo PPJoin:
// |pref(x)| = ceil((1-t)*|x|) + 1, con tokens ordenados por DF asc.
func BuildKeywordIndex(records []DetectorRecord, vocab *Vocab, opts Options) (keywordIndex, TextRepeatStats) {
	n := len(records)

	df := make(map[uint32]uint32, 1024)
	for _, r := range records {
		for _, id := range r.TokenIDs {
			df[id]++
		}
	}

	minLen := opts.MinTokenLength
	if minLen <= 0 {
		minLen = 4
	}
	minDF := opts.MinKeywordDocFreq
	if minDF <= 0 {
		minDF = 2
	}
	maxRatio := opts.MaxKeywordDocFreqRatio
	if maxRatio <= 0 {
		maxRatio = 0.05
	}
	maxDF := uint32(float64(n) * maxRatio)
	if maxDF < uint32(minDF) {
		maxDF = uint32(minDF)
	}

	keywords := make(map[uint32]struct{}, len(df)/4)
	for id, c := range df {
		if c < uint32(minDF) {
			continue
		}
		if maxDF > 0 && c > maxDF {
			continue
		}
		if vocab != nil {
			if tok, ok := vocab.Token(id); ok {
				if len(tok) < minLen {
					continue
				}
			}
		}
		keywords[id] = struct{}{}
	}

	stats := TextRepeatStats{
		SelectedKeywordCnt: len(keywords),
	}
	if vocab != nil {
		stats.TotalVocabSize = vocab.Size()
	}

	// Precomputar prefijos (o lista completa) por registro para evitar ordenar N veces en el hot path.
	prefix := make([][]uint32, n)
	postings := make(map[uint32][]postEntry, len(keywords))

	for i, r := range records {
		ordered := filterAndOrderByDF(r.TokenIDs, keywords, df)
		if len(ordered) == 0 {
			prefix[i] = nil
			continue
		}

		toIndex := ordered
		if opts.UsePrefixFilter {
			pLen := prefixLenPPJoin(len(ordered), opts.JaccardThreshold)
			if pLen > len(ordered) {
				pLen = len(ordered)
			}
			if opts.MaxPrefixTokens > 0 && pLen > opts.MaxPrefixTokens {
				pLen = opts.MaxPrefixTokens
			}
			toIndex = ordered[:pLen]
			prefix[i] = toIndex
		} else {
			// Sin prefijo: usamos todos los tokens filtrados como "prefix" para reutilizar lógica.
			prefix[i] = ordered
		}

		for pos, tok := range toIndex {
			if opts.MaxPostingsPerToken > 0 {
				cur := postings[tok]
				if len(cur) >= opts.MaxPostingsPerToken {
					// truncar determinísticamente: mantener los primeros MaxPostingsPerToken (idx más chicos)
					stats.PostingsSkippedEntries++
					if len(cur) == opts.MaxPostingsPerToken {
						// contar token truncado solo la primera vez que se excede el cap
						stats.PostingsTruncatedTokens++
					}
					continue
				}
			}
			postings[tok] = append(postings[tok], postEntry{idx: i, pos: uint16(pos)})
			stats.PrefixPostings++
		}
		stats.InvertedPostings += len(ordered)
	}

	return keywordIndex{df: df, keywords: keywords, postings: postings, prefix: prefix}, stats
}

func filterAndOrderByDF(tokenIDs []uint32, keywords map[uint32]struct{}, df map[uint32]uint32) []uint32 {
	out := make([]uint32, 0, len(tokenIDs))
	for _, id := range tokenIDs {
		if _, ok := keywords[id]; !ok {
			continue
		}
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool {
		di := df[out[i]]
		dj := df[out[j]]
		if di != dj {
			return di < dj
		}
		return out[i] < out[j]
	})
	return out
}

// prefixLenPPJoin aplica la Eq. 6: |pref(x)| = ceil((1-t)*|x|) + 1.
func prefixLenPPJoin(size int, t float64) int {
	if size <= 0 {
		return 0
	}
	if t <= 0 {
		return size
	}
	if t >= 1 {
		return 1
	}
	return int(math.Ceil((1.0-t)*float64(size))) + 1
}

func requiredOverlapJaccard(lenA, lenB int, t float64) int {
	if lenA <= 0 || lenB <= 0 {
		return 0
	}
	if t <= 0 {
		return 0
	}
	return int(math.Ceil((t / (1.0 + t)) * float64(lenA+lenB)))
}

func rareTokenScore(df uint32) uint32 {
	// score mayor para tokens raros; evita float en el hot path.
	if df == 0 {
		return 0
	}
	return uint32(1_000_000 / df)
}

// JaccardAtLeastUint32 verifica si Jaccard(a,b) >= threshold sin calcular el score exacto.
// Usa dos punteros sobre slices ordenados y únicos, y corta temprano cuando sea imposible
// alcanzar el overlap requerido.
func JaccardAtLeastUint32(a, b []uint32, threshold float64) (ok bool, earlyExit bool) {
	lenA, lenB := len(a), len(b)
	if lenA == 0 || lenB == 0 {
		return false, false
	}
	required := requiredOverlapJaccard(lenA, lenB, threshold)
	if required <= 0 {
		return true, false
	}

	i, j := 0, 0
	overlap := 0
	for i < lenA && j < lenB {
		// early accept
		if overlap >= required {
			return true, false
		}
		// early reject: incluso con match perfecto del resto no alcanza
		remain := lenA - i
		if r := lenB - j; r < remain {
			remain = r
		}
		if overlap+remain < required {
			return false, true
		}

		ai := a[i]
		bj := b[j]
		if ai == bj {
			overlap++
			i++
			j++
		} else if ai < bj {
			i++
		} else {
			j++
		}
	}
	if overlap >= required {
		return true, false
	}
	return false, true
}

type candAgg struct {
	j        int
	shared   uint16
	rare     uint32
	lastPosX uint16
	lastPosY uint16
}

func candBetter(a, b candAgg, lenA int, records []DetectorRecord) bool {
	if a.shared != b.shared {
		return a.shared > b.shared
	}
	if a.rare != b.rare {
		return a.rare > b.rare
	}
	da := absInt(lenA - len(records[a.j].TokenIDs))
	db := absInt(lenA - len(records[b.j].TokenIDs))
	if da != db {
		return da < db
	}
	return a.j < b.j
}

// topNSelector mantiene solo los mejores N candidatos sin ordenar todo el universo.
// El heap guarda al peor candidato en root (índice 0).
type topNSelector struct {
	n       int
	data    []candAgg
	lenA    int
	records []DetectorRecord
}

func (s *topNSelector) reset(n int, lenA int, records []DetectorRecord) {
	s.n = n
	s.lenA = lenA
	s.records = records
	s.data = s.data[:0]
}

func (s *topNSelector) worse(i, j int) bool {
	// true si data[i] es peor que data[j]
	return candBetter(s.data[j], s.data[i], s.lenA, s.records)
}

func (s *topNSelector) siftUp(i int) {
	for i > 0 {
		p := (i - 1) / 2
		if !s.worse(p, i) {
			break
		}
		s.data[p], s.data[i] = s.data[i], s.data[p]
		i = p
	}
}

func (s *topNSelector) siftDown(i int) {
	for {
		l := 2*i + 1
		if l >= len(s.data) {
			return
		}
		r := l + 1
		worst := l
		if r < len(s.data) && s.worse(l, r) {
			worst = r
		}
		if !s.worse(i, worst) {
			return
		}
		s.data[i], s.data[worst] = s.data[worst], s.data[i]
		i = worst
	}
}

func (s *topNSelector) consider(c candAgg) {
	if s.n <= 0 {
		s.data = append(s.data, c)
		return
	}
	if len(s.data) < s.n {
		s.data = append(s.data, c)
		s.siftUp(len(s.data) - 1)
		return
	}
	// reemplazar root si c es mejor que el peor actual
	if candBetter(c, s.data[0], s.lenA, s.records) {
		s.data[0] = c
		s.siftDown(0)
	}
}

func (s *topNSelector) finalize() []candAgg {
	// ordenar los seleccionados determinísticamente por ranking "mejor primero"
	sort.Slice(s.data, func(i, j int) bool {
		return candBetter(s.data[i], s.data[j], s.lenA, s.records)
	})
	return s.data
}

// detectTextRepeatKeywordIndexCount cuenta flags TEXT_REPEAT usando prefix filtering + index.
func detectTextRepeatKeywordIndexCount(records []DetectorRecord, idx keywordIndex, opts Options, progress *CountProgress) (TextRepeatStats, time.Duration) {
	start := time.Now()
	n := len(records)

	minShared := opts.MinSharedKeywords
	if minShared <= 0 {
		minShared = 2
	}

	var stats TextRepeatStats
	var comparisons uint64
	var flags uint64
	var jaccardCalls uint64
	var jaccardEarly uint64
	var accepted uint64

	// Reuso por registro
	counts := make(map[int]*candAgg, 4096)
	touched := make([]*candAgg, 0, 4096)
	var selector topNSelector

	for i := 0; i < n; i++ {
		// limpiar
		for _, c := range touched {
			delete(counts, c.j)
		}
		touched = touched[:0]

		pref := idx.prefix[i]
		for posX, tok := range pref {
			lst := idx.postings[tok]
			// lst está en orden idx asc; buscar first > i
			p := sort.Search(len(lst), func(k int) bool { return lst[k].idx > i })
			for ; p < len(lst); p++ {
				ent := lst[p]
				j := ent.idx
				c := counts[j]
				if c == nil {
					c = &candAgg{j: j, lastPosX: uint16(posX), lastPosY: ent.pos}
					counts[j] = c
					touched = append(touched, c)
					stats.CandidatesTouched++
				}
				if c.shared < 0xFFFF {
					c.shared++
				}
				c.rare += rareTokenScore(idx.df[tok])
				if uint16(posX) > c.lastPosX {
					c.lastPosX = uint16(posX)
				}
				if ent.pos > c.lastPosY {
					c.lastPosY = ent.pos
				}
			}
		}

		lenA := len(records[i].TokenIDs)
		selector.reset(opts.MaxCandidatesPerRecord, lenA, records)
		eligible := 0

		for _, cp := range touched {
			if int(cp.shared) < minShared {
				continue
			}
			c := *cp
			eligible++
			j := c.j
			lenB := len(records[j].TokenIDs)

			if opts.MinRareScore > 0 && c.rare < opts.MinRareScore {
				stats.CandidatesSkippedByRare++
				continue
			}

			if opts.UseLengthFilter {
				minL, maxL := lenA, lenB
				if minL > maxL {
					minL, maxL = maxL, minL
				}
				if maxL > 0 && float64(minL)/float64(maxL) < opts.JaccardThreshold {
					stats.CandidatesSkippedByLength++
					continue
				}
			}

			if opts.UseUpperBoundFilter {
				alpha := requiredOverlapJaccard(lenA, lenB, opts.JaccardThreshold)
				// MaxOverlap = sharedPrefix + min(|rp(x)|, |rp(y)|) con rp basado en última coincidencia del prefijo.
				rpX := lenA - int(c.lastPosX) - 1
				rpY := lenB - int(c.lastPosY) - 1
				if rpX < 0 {
					rpX = 0
				}
				if rpY < 0 {
					rpY = 0
				}
				maxOverlap := int(c.shared) + minInt(rpX, rpY)
				if maxOverlap < alpha {
					stats.CandidatesSkippedByUpper++
					continue
				}
			}

			selector.consider(c)
		}

		stats.CandidatePairsGenerated += uint64(eligible)
		selected := selector.finalize()
		stats.CandidatesSelected += uint64(len(selected))
		if opts.MaxCandidatesPerRecord > 0 && eligible > len(selected) {
			stats.CandidatesSkippedByCap += uint64(eligible - len(selected))
		}

		for _, c := range selected {
			comparisons++
			j := c.j
			jaccardCalls++
			ok, early := JaccardAtLeastUint32(records[i].TokenIDs, records[j].TokenIDs, opts.JaccardThreshold)
			if early {
				jaccardEarly++
			}
			addUint64(progress.Comparisons, 1)
			if ok {
				accepted++
				flags += 2
				addUint64(progress.Flags, 2)
			}
		}
	}

	stats.ComparisonsVerified = comparisons
	stats.FlagsDetected = flags
	stats.JaccardCalls = jaccardCalls
	stats.JaccardEarlyExits = jaccardEarly
	stats.AcceptedPairs = accepted
	addUint64(progress.Flags, 0)
	return stats, time.Since(start)
}

func detectTextRepeatKeywordIndexFlags(records []DetectorRecord, idx keywordIndex, opts Options) ([]RedFlag, TextRepeatStats, time.Duration) {
	start := time.Now()
	n := len(records)
	minShared := opts.MinSharedKeywords
	if minShared <= 0 {
		minShared = 2
	}

	var stats TextRepeatStats
	var comparisons uint64
	var flagsCnt uint64
	flags := make([]RedFlag, 0, 1024)

	counts := make(map[int]*candAgg, 4096)
	touched := make([]*candAgg, 0, 4096)

	for i := 0; i < n; i++ {
		for _, c := range touched {
			delete(counts, c.j)
		}
		touched = touched[:0]

		pref := idx.prefix[i]
		for posX, tok := range pref {
			lst := idx.postings[tok]
			p := sort.Search(len(lst), func(k int) bool { return lst[k].idx > i })
			for ; p < len(lst); p++ {
				ent := lst[p]
				j := ent.idx
				c := counts[j]
				if c == nil {
					c = &candAgg{j: j, lastPosX: uint16(posX), lastPosY: ent.pos}
					counts[j] = c
					touched = append(touched, c)
				}
				if c.shared < 0xFFFF {
					c.shared++
				}
				c.rare += rareTokenScore(idx.df[tok])
				if uint16(posX) > c.lastPosX {
					c.lastPosX = uint16(posX)
				}
				if ent.pos > c.lastPosY {
					c.lastPosY = ent.pos
				}
			}
		}

		cands := make([]candAgg, 0, len(touched))
		for _, c := range touched {
			if int(c.shared) < minShared {
				continue
			}
			cands = append(cands, *c)
		}
		stats.CandidatePairsGenerated += uint64(len(cands))

		lenA := len(records[i].TokenIDs)
		filtered := make([]candAgg, 0, len(cands))
		for _, c := range cands {
			j := c.j
			lenB := len(records[j].TokenIDs)
			if opts.UseLengthFilter {
				minL, maxL := lenA, lenB
				if minL > maxL {
					minL, maxL = maxL, minL
				}
				if maxL > 0 && float64(minL)/float64(maxL) < opts.JaccardThreshold {
					stats.CandidatesSkippedByLength++
					continue
				}
			}
			if opts.UseUpperBoundFilter {
				alpha := requiredOverlapJaccard(lenA, lenB, opts.JaccardThreshold)
				rpX := lenA - int(c.lastPosX) - 1
				rpY := lenB - int(c.lastPosY) - 1
				if rpX < 0 {
					rpX = 0
				}
				if rpY < 0 {
					rpY = 0
				}
				maxOverlap := int(c.shared) + minInt(rpX, rpY)
				if maxOverlap < alpha {
					stats.CandidatesSkippedByUpper++
					continue
				}
			}
			filtered = append(filtered, c)
		}

		sort.Slice(filtered, func(a, b int) bool {
			aa, bb := filtered[a], filtered[b]
			if aa.shared != bb.shared {
				return aa.shared > bb.shared
			}
			if aa.rare != bb.rare {
				return aa.rare > bb.rare
			}
			da := absInt(lenA - len(records[aa.j].TokenIDs))
			db := absInt(lenA - len(records[bb.j].TokenIDs))
			if da != db {
				return da < db
			}
			return aa.j < bb.j
		})
		if opts.MaxCandidatesPerRecord > 0 && len(filtered) > opts.MaxCandidatesPerRecord {
			stats.CandidatesSkippedByCap += uint64(len(filtered) - opts.MaxCandidatesPerRecord)
			filtered = filtered[:opts.MaxCandidatesPerRecord]
		}

		for _, c := range filtered {
			comparisons++
			j := c.j
			score := JaccardUint32(records[i].TokenIDs, records[j].TokenIDs)
			if score >= opts.JaccardThreshold {
				flags = append(flags, RedFlag{
					RecordIndex:  i,
					Expediente:   records[i].Expediente,
					DatasetID:    records[i].DatasetID,
					FlagType:     FlagTextRepeat,
					Score:        score,
					Details:      fmt.Sprintf("similar a expediente %s (score=%.2f, keyword-index)", records[j].Expediente, score),
					QuejaPreview: quejaPreview(records[i].Queja),
				})
				flags = append(flags, RedFlag{
					RecordIndex:  j,
					Expediente:   records[j].Expediente,
					DatasetID:    records[j].DatasetID,
					FlagType:     FlagTextRepeat,
					Score:        score,
					Details:      fmt.Sprintf("similar a expediente %s (score=%.2f, keyword-index)", records[i].Expediente, score),
					QuejaPreview: quejaPreview(records[j].Queja),
				})
				flagsCnt += 2
			}
		}
	}

	stats.ComparisonsVerified = comparisons
	stats.FlagsDetected = flagsCnt
	return flags, stats, time.Since(start)
}

func detectTextRepeatKeywordIndexCountConcurrent(records []DetectorRecord, idx keywordIndex, opts Options, numWorkers int, progress *CountProgress) (TextRepeatStats, time.Duration) {
	start := time.Now()
	n := len(records)
	if n == 0 {
		return TextRepeatStats{}, time.Since(start)
	}
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}

	minShared := opts.MinSharedKeywords
	if minShared <= 0 {
		minShared = 2
	}

	chunk := (n + numWorkers - 1) / numWorkers

	var totalStats TextRepeatStats
	var totalComparisons uint64
	var totalFlags uint64
	var totalJaccardCalls uint64
	var totalJaccardEarly uint64
	var totalAccepted uint64

	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		lo := w * chunk
		hi := lo + chunk
		if hi > n {
			hi = n
		}
		if lo >= hi {
			continue
		}
		wg.Add(1)
		go func(lo, hi int) {
			defer wg.Done()

			local := TextRepeatStats{}
			var comps uint64
			var flags uint64
			var jaccCalls uint64
			var jaccEarly uint64
			var accepted uint64

			counts := make(map[int]*candAgg, 4096)
			touched := make([]*candAgg, 0, 4096)
			var selector topNSelector

			for i := lo; i < hi; i++ {
				for _, c := range touched {
					delete(counts, c.j)
				}
				touched = touched[:0]

				pref := idx.prefix[i]
				for posX, tok := range pref {
					lst := idx.postings[tok]
					p := sort.Search(len(lst), func(k int) bool { return lst[k].idx > i })
					for ; p < len(lst); p++ {
						ent := lst[p]
						j := ent.idx
						c := counts[j]
						if c == nil {
							c = &candAgg{j: j, lastPosX: uint16(posX), lastPosY: ent.pos}
							counts[j] = c
							touched = append(touched, c)
							local.CandidatesTouched++
						}
						if c.shared < 0xFFFF {
							c.shared++
						}
						c.rare += rareTokenScore(idx.df[tok])
						if uint16(posX) > c.lastPosX {
							c.lastPosX = uint16(posX)
						}
						if ent.pos > c.lastPosY {
							c.lastPosY = ent.pos
						}
					}
				}
				lenA := len(records[i].TokenIDs)
				selector.reset(opts.MaxCandidatesPerRecord, lenA, records)
				eligible := 0
				for _, cp := range touched {
					if int(cp.shared) < minShared {
						continue
					}
					c := *cp
					eligible++
					j := c.j
					lenB := len(records[j].TokenIDs)
					if opts.MinRareScore > 0 && c.rare < opts.MinRareScore {
						local.CandidatesSkippedByRare++
						continue
					}
					if opts.UseLengthFilter {
						minL, maxL := lenA, lenB
						if minL > maxL {
							minL, maxL = maxL, minL
						}
						if maxL > 0 && float64(minL)/float64(maxL) < opts.JaccardThreshold {
							local.CandidatesSkippedByLength++
							continue
						}
					}
					if opts.UseUpperBoundFilter {
						alpha := requiredOverlapJaccard(lenA, lenB, opts.JaccardThreshold)
						rpX := lenA - int(c.lastPosX) - 1
						rpY := lenB - int(c.lastPosY) - 1
						if rpX < 0 {
							rpX = 0
						}
						if rpY < 0 {
							rpY = 0
						}
						maxOverlap := int(c.shared) + minInt(rpX, rpY)
						if maxOverlap < alpha {
							local.CandidatesSkippedByUpper++
							continue
						}
					}
					selector.consider(c)
				}

				local.CandidatePairsGenerated += uint64(eligible)
				selected := selector.finalize()
				local.CandidatesSelected += uint64(len(selected))
				if opts.MaxCandidatesPerRecord > 0 && eligible > len(selected) {
					local.CandidatesSkippedByCap += uint64(eligible - len(selected))
				}

				for _, c := range selected {
					comps++
					addUint64(progress.Comparisons, 1)
					jaccCalls++
					ok, early := JaccardAtLeastUint32(records[i].TokenIDs, records[c.j].TokenIDs, opts.JaccardThreshold)
					if early {
						jaccEarly++
					}
					if ok {
						accepted++
						flags += 2
						addUint64(progress.Flags, 2)
					}
				}
			}

			atomic.AddUint64(&totalComparisons, comps)
			atomic.AddUint64(&totalFlags, flags)
			atomic.AddUint64(&totalJaccardCalls, jaccCalls)
			atomic.AddUint64(&totalJaccardEarly, jaccEarly)
			atomic.AddUint64(&totalAccepted, accepted)

			// acumular stats locales
			atomic.AddUint64(&totalStats.CandidatePairsGenerated, local.CandidatePairsGenerated)
			atomic.AddUint64(&totalStats.CandidatesTouched, local.CandidatesTouched)
			atomic.AddUint64(&totalStats.CandidatesSelected, local.CandidatesSelected)
			atomic.AddUint64(&totalStats.CandidatesSkippedByCap, local.CandidatesSkippedByCap)
			atomic.AddUint64(&totalStats.CandidatesSkippedByRare, local.CandidatesSkippedByRare)
			atomic.AddUint64(&totalStats.CandidatesSkippedByLength, local.CandidatesSkippedByLength)
			atomic.AddUint64(&totalStats.CandidatesSkippedByUpper, local.CandidatesSkippedByUpper)
		}(lo, hi)
	}

	wg.Wait()

	totalStats.ComparisonsVerified = atomic.LoadUint64(&totalComparisons)
	totalStats.FlagsDetected = atomic.LoadUint64(&totalFlags)
	totalStats.JaccardCalls = atomic.LoadUint64(&totalJaccardCalls)
	totalStats.JaccardEarlyExits = atomic.LoadUint64(&totalJaccardEarly)
	totalStats.AcceptedPairs = atomic.LoadUint64(&totalAccepted)
	totalStats.Truncated = false
	return totalStats, time.Since(start)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
