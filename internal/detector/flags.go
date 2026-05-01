package detector

const (
	FlagTextRepeat     = "TEXT_REPEAT"
	FlagTimingBurst    = "TIMING_BURST"
	FlagExactDuplicate = "EXACT_DUPLICATE"
)

// RedFlag representa una anomalía detectada en un expediente.
type RedFlag struct {
	RecordIndex  int
	Expediente   string
	DatasetID    string
	FlagType     string
	Score        float64 // Jaccard para TEXT_REPEAT; conteo para TIMING_BURST; 1.0 para DUPLICATE
	Details      string
	QuejaPreview string // primeros 80 chars del campo queja
}

func quejaPreview(queja string) string {
	runes := []rune(queja)
	if len(runes) > 80 {
		return string(runes[:80]) + "..."
	}
	return queja
}
