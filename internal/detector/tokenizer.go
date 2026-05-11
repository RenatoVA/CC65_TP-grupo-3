package detector

import (
	"sort"
	"strings"
	"unicode"
)

var spanishStopwords = map[string]bool{
	"de": true, "la": true, "el": true, "en": true, "que": true, "y": true,
	"a": true, "un": true, "por": true, "con": true, "una": true, "es": true,
	"se": true, "no": true, "al": true, "lo": true, "le": true, "da": true,
	"su": true, "los": true, "las": true, "del": true, "me": true, "mi": true,
	"mas": true, "pero": true, "sus": true, "les": true, "como": true, "ha": true,
	"ya": true, "fue": true, "son": true, "ser": true, "hay": true, "para": true,
	"he": true, "ante": true, "sin": true, "sobre": true, "entre": true,
	"era": true, "muy": true, "si": true, "nos": true, "este": true, "esta": true,
	"esto": true, "ese": true, "esa": true, "eso": true, "cual": true, "donde": true,
	"cuando": true, "todo": true, "todos": true, "toda": true, "han": true,
	"porque": true, "aunque": true, "ni": true, "bien": true, "sido": true,
	"hacia": true, "hasta": true, "desde": true, "durante": true, "mientras": true,
}

// Tokenize convierte texto en un slice de tokens normalizados y únicos,
// eliminando stopwords y tokens de menos de 3 caracteres.
func Tokenize(text string) []string {
	text = strings.ToLower(text)

	// reemplazar puntuación por espacios
	var sb strings.Builder
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' {
			sb.WriteRune(r)
		} else {
			sb.WriteRune(' ')
		}
	}

	words := strings.Fields(sb.String())
	seen := make(map[string]struct{}, len(words))
	tokens := make([]string, 0, len(words))
	for _, w := range words {
		if len(w) < 3 {
			continue
		}
		if spanishStopwords[w] {
			continue
		}
		if _, ok := seen[w]; ok {
			continue
		}
		seen[w] = struct{}{}
		tokens = append(tokens, w)
	}
	sort.Strings(tokens) // ordenar para merge eficiente en Jaccard
	return tokens
}

// TokenizeToIDs convierte texto en un slice de IDs uint32 ordenados usando el vocab global.
// Mismo pipeline que Tokenize() pero el output son enteros → comparación 4x más rápida.
func TokenizeToIDs(text string, vocab *Vocab) []uint32 {
	text = strings.ToLower(text)
	var sb strings.Builder
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' {
			sb.WriteRune(r)
		} else {
			sb.WriteRune(' ')
		}
	}
	words := strings.Fields(sb.String())
	seen := make(map[uint32]struct{}, len(words))
	ids := make([]uint32, 0, len(words))
	for _, w := range words {
		if len(w) < 3 || spanishStopwords[w] {
			continue
		}
		id := vocab.GetOrAdd(w)
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// JaccardUint32 calcula Jaccard sobre slices de uint32 ordenados.
// Merge de dos punteros O(|a|+|b|) comparando enteros — sin string hashing.
func JaccardUint32(a, b []uint32) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}
	intersection := 0
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] == b[j]:
			intersection++
			i++
			j++
		case a[i] < b[j]:
			i++
		default:
			j++
		}
	}
	union := len(a) + len(b) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

// Jaccard calcula la similitud de Jaccard entre dos slices de tokens ordenados.
// Usa merge de dos punteros en O(|a|+|b|) sin allocar sets.
func Jaccard(a, b []string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}

	intersection := 0
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] == b[j]:
			intersection++
			i++
			j++
		case a[i] < b[j]:
			i++
		default:
			j++
		}
	}

	union := len(a) + len(b) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}
