# Detector Concurrente de Red Flags - Expedientes INDECOPI

**Curso:** Programacion Concurrente y Distribuida  
**Caso de uso:** deteccion de spam y anomalias en denuncias ciudadanas ante INDECOPI.

## Resumen

El proyecto implementa un pipeline en Go para consolidar expedientes publicos de INDECOPI, generar/enriquecer el campo `queja` y detectar red flags sobre datasets grandes. El detector actual identifica:

- `TEXT_REPEAT`: textos sospechosamente similares.
- `TIMING_BURST`: muchas denuncias en una misma fecha.
- `EXACT_DUPLICATE`: duplicados por `expediente|denunciado|fecha_presentacion`.
- `KEYWORD_SPAM`: lenguaje de spam, phishing, markdown o texto sintetico.

La norma actual del algoritmo es fija: `TEXT_REPEAT` usa `keyword-index` con filtros de prefijo, longitud, cota superior y verificacion final con Jaccard sobre `[]uint32`.

## Algoritmo Actual

La version inicial comparaba todos los pares con Jaccard sobre strings, lo que era `O(n^2)` e inviable para 1M de filas. La version actual cambia el flujo:

1. Tokeniza cada `queja`, elimina stopwords y representa tokens como IDs `uint32`.
2. Calcula frecuencia documental global de tokens.
3. Selecciona solo keywords significativas:
   - `min-df=4`
   - `max-df-ratio=0.003`
   - `min-token-length=6`
4. Ordena keywords por frecuencia ascendente.
5. Construye un indice invertido usando prefijo PPJoin.
6. Genera candidatos solo si comparten al menos `7` keywords.
7. Aplica filtros antes de Jaccard:
   - prefix filter
   - length filter
   - upper-bound filter
8. Verifica como maximo `10` candidatos por registro.
9. Confirma `TEXT_REPEAT` con Jaccard final `>= 0.60`.

Configuracion estandar en codigo:

```text
JaccardThreshold=0.60
BurstLimit=5
KeywordThreshold=1
MinKeywordDocFreq=4
MaxKeywordDocFreqRatio=0.003
MinSharedKeywords=7
MinTokenLength=6
MaxCandidatesPerRecord=10
UsePrefixFilter=true
UseLengthFilter=true
UseUpperBoundFilter=true
```

## Concurrencia

- La carga concurrente usa un pipeline: reader -> worker pool tokenizador -> collector.
- El vocabulario global usa `sync.RWMutex`.
- La deteccion concurrente divide el rango de registros entre workers para la fase `TEXT_REPEAT`.
- Las fases `TIMING_BURST`, `EXACT_DUPLICATE` y `KEYWORD_SPAM` se mantienen lineales.

## Estructura Principal

```text
cmd/
  benchfull/             benchmark oficial
  consolidator/          descarga y consolida Excel de INDECOPI
  detector_secuencial/   exporta flags a CSV usando la norma actual
  detector_concurrente/  exporta flags a CSV usando la norma actual
  enrich/                genera campo queja via OpenRouter
  fillskipped/           rellena quejas faltantes con plantillas
  generar1m/             genera dataset grande para pruebas

internal/detector/
  sequential.go                  Options, defaults y detector secuencial
  concurrent.go                  detector concurrente
  detect_count.go                conteo sin materializar flags
  textrepeat_keyword_index.go    indice de keywords y filtros
  tokenizer.go                   tokenizacion, Jaccard y TokenIDs
  vocab.go                       vocabulario thread-safe
  keywords.go                    KEYWORD_SPAM
```

## Ejecucion

Preparar dataset:

```bash
go run ./cmd/consolidator

export OPENROUTER_API_KEY=sk-or-...
go run ./cmd/enrich -model=openai/gpt-4o-mini -limit=500
go run ./cmd/fillskipped
```

Detectores que exportan CSV:

```bash
go run ./cmd/detector_secuencial -input=data/processed/enriched_filled.csv -output=data/results/flagged_records.csv
go run ./cmd/detector_concurrente -input=data/processed/enriched_filled.csv -output=data/results/flagged_records_conc.csv -workers=4
```

Benchmark oficial:

```bash
go run ./cmd/benchfull -input=data/processed/enriched_filled.csv -runs=3 -workers-list=2,4,8 -count-only
```

Benchmark recomendado para 1M:

```bash
go run ./cmd/benchfull -input="data/processed/enriched_1m_real.csv" -runs=1 -workers-list="2,4,8" -count-only -progress-every=5s
```

`benchfull` ya aplica la norma del algoritmo internamente; no se deben pasar flags de estrategia o filtros.

## Verificacion

```bash
env GOCACHE=/tmp/go-build-cache go test ./...
```

Si se usa la cache global de Go en un entorno de solo lectura, `go test ./...` puede fallar por permisos de cache. En ese caso usar `GOCACHE=/tmp/go-build-cache`.

## Verificacion formal con Promela/SPIN

Los modelos actuales estan en:

- `promela/detector_secuencial.pml`
- `promela/detector_concurrente.pml`

Comandos recomendados:

```bash
spin -search promela/detector_secuencial.pml
spin -search -ltl no_deadlock promela/detector_secuencial.pml
spin -search -ltl orden_fases promela/detector_secuencial.pml

spin -search promela/detector_concurrente.pml
spin -search -ltl no_deadlock promela/detector_concurrente.pml
spin -search -ltl mutex_ok promela/detector_concurrente.pml
```

El modelo concurrente verifica que el merge global de estadisticas tenga exclusion mutua (`in_merge <= 1`) y que todos los workers terminen.
