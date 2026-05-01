package detector

import (
	"runtime"
	"sort"
)

// TrimmedMean calcula la media recortada eliminando trimPct% de cada extremo.
// trimPct debe estar entre 0 y 0.5 (ej: 0.10 para recortar 10% de cada lado).
func TrimmedMean(values []float64, trimPct float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	cut := int(float64(len(sorted)) * trimPct)
	trimmed := sorted[cut : len(sorted)-cut]
	if len(trimmed) == 0 {
		return sorted[len(sorted)/2]
	}

	var sum float64
	for _, v := range trimmed {
		sum += v
	}
	return sum / float64(len(trimmed))
}

// ResourceSnapshot captura el estado de memoria y goroutines en un momento dado.
type ResourceSnapshot struct {
	HeapAllocMB  float64
	TotalAllocMB float64
	SysMB        float64
	GCCycles     uint32
	Goroutines   int
}

// CaptureResources toma un snapshot del uso de recursos actuales.
func CaptureResources() ResourceSnapshot {
	runtime.GC() // forzar GC para medición limpia
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return ResourceSnapshot{
		HeapAllocMB:  float64(ms.HeapAlloc) / 1024 / 1024,
		TotalAllocMB: float64(ms.TotalAlloc) / 1024 / 1024,
		SysMB:        float64(ms.Sys) / 1024 / 1024,
		GCCycles:     ms.NumGC,
		Goroutines:   runtime.NumGoroutine(),
	}
}
