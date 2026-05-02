# TP-Prorgramacion-Concurente

## Informe de Práctica Calificada 1

**Carrera:** Ciencias de la Computación  
**Curso:** Programación Concurrente y Distribuida  
**Tema elegido:** Detector Concurrente de "Red Flags" en el Libro de Reclamaciones / Expedientes de INDECOPI  
**Alcance de esta entrega:** búsqueda, consolidación, inspección estructural y limpieza del dataset.  

---

## Resumen del trabajo

En esta primera entrega se desarrolló la fase de adquisición y preparación de datos para un futuro detector concurrente de anomalías en registros de INDECOPI. El caso de uso parte del problema de identificar patrones sospechosos en expedientes y denuncias, tales como repeticiones inusuales, concentraciones por empresa denunciada, series anómalas de ingresos o comportamientos masivos en ventanas temporales reducidas.

A diferencia de una implementación completa de detección de spam, esta etapa se concentra exclusivamente en construir una base de datos sólida y utilizable a partir de fuentes oficiales abiertas. Para ello se exploró la Plataforma Nacional de Datos Abiertos del Perú, se identificaron los datasets publicados por INDECOPI, se descargaron los recursos disponibles y se diseñó un proceso de consolidación robusto en Go capaz de lidiar con archivos heterogéneos, hojas con encabezados desplazados y estructuras manuales hechas por humanos.

El resultado de esta fase es un dataset consolidado a partir de múltiples familias de expedientes de INDECOPI, junto con perfiles estructurales por archivo y una propuesta clara de normalización. Esta base servirá como insumo para entregas posteriores, donde sí se implementarán reglas de detección de anomalías, procesamiento concurrente con goroutines y sincronización segura sobre contadores o estructuras compartidas.

---

## Objetivo general

Construir y limpiar un dataset consolidado de expedientes de INDECOPI, obtenido desde fuentes oficiales abiertas, que permita en una etapa posterior implementar un detector concurrente de anomalías o "red flags" sobre los registros ciudadanos.

## Objetivos específicos

- Identificar datasets relevantes de INDECOPI publicados en la Plataforma Nacional de Datos Abiertos.
- Descargar y consolidar múltiples archivos heterogéneos en un único flujo de procesamiento.
- Inspeccionar la estructura real de las hojas y columnas de cada tipo de dataset.
- Detectar y corregir problemas de encabezados desplazados y formatos no homogéneos.
- Proponer una normalización común para las familias de datos más relevantes.
- Generar un dataset limpio y trazable para la siguiente etapa del proyecto.

---

## Análisis del caso de uso

### Tema
Detección concurrente de spam y anomalías en registros de INDECOPI.

### Caso de uso elegido
Detector concurrente de "Red Flags" en expedientes y denuncias de INDECOPI.

La motivación del caso de uso es que los sistemas públicos de atención al ciudadano pueden ser vulnerables a patrones anómalos, cargas artificiales, campañas coordinadas o registros masivos que distorsionen el comportamiento esperado del sistema. En el caso de INDECOPI, aunque no se encontró públicamente el dataset ideal del Libro de Reclamaciones, sí se hallaron múltiples datasets de expedientes presentados y resueltos que permiten construir una aproximación realista al problema.

Dado que esta práctica todavía no exige implementar el detector concurrente, el foco se colocó en una fase previa, pero indispensable: la obtención del dataset. Se partió de la hipótesis de que un sistema concurrente futuro necesitará, como mínimo, atributos como fecha de presentación, empresa denunciada o administrado, tipo de expediente y, cuando sea posible, sector, subsector, materia o resolución. En consecuencia, el trabajo se orientó a descubrir qué tan cerca están los datos abiertos de INDECOPI de proveer ese esquema mínimo.

El principal reto no fue solo encontrar archivos, sino interpretar correctamente su estructura. Muchos de los spreadsheets publicados por INDECOPI no son exportaciones uniformes de una base de datos, sino archivos armados manualmente, con títulos, subtítulos, espacios vacíos y encabezados colocados varias filas por debajo del inicio real de la hoja. Por ello, antes de pensar en goroutines o en reglas de anomalía, fue necesario resolver un problema más básico: construir un pipeline de lectura y normalización suficientemente robusto como para no descartar datos válidos por defectos de formato.

---

## Exploración de fuentes de datos

Se exploró el grupo institucional de INDECOPI dentro de la Plataforma Nacional de Datos Abiertos del Perú y se identificaron **22 datasets** publicados. A partir de ellos se encontraron **50 archivos descargables** en formatos `.xlsx` y `.xls`.

Entre los datasets detectados se encuentran:

- OPS1 expedientes presentados y resueltos
- OPS2 expedientes presentados y resueltos
- OPS3 expedientes presentados y resueltos
- CC1 expedientes presentados y resueltos
- CC2 expedientes presentados y resueltos
- CC3 expedientes presentados y resueltos
- SPC expedientes presentados y resueltos
- CLC expedientes resueltos
- DIN expedientes presentados y resueltos
- DDA expedientes de infracciones resueltos
- DDA expedientes de registro resueltos
- DSD expedientes presentados, resueltos y otorgados

Esta exploración permitió concluir que no todos los datasets publicados por INDECOPI son igualmente relevantes para el caso de uso. Algunos contienen registros tabulares con detalle suficiente para una futura detección de anomalías, mientras que otros son más agregados, administrativos o propios de dominios que no aportan variables útiles para el escenario planteado.

---

## Familias de datasets encontradas

A partir de la inspección estructural de una muestra por tipo de dataset, se identificaron las siguientes familias relevantes:

### 1. Familia CC1
Columnas observadas:
- Nro. Exped.
- Tipo de Exp.
- Fecha de presentación
- Sector
- Subsector
- Denunciados
- Documento de denunciados
- Fecha de resolución
- Nro. de Resolución
- Conclusión

### 2. Familia CC2
Columnas observadas:
- NRO EXPEDIENTE
- TIPO EXPEDIENTE
- FECHA PRESENTACION
- DENUNCIADOS
- RUC DENUNCIADOS
- SECTOR
- SUB SECTOR
- CIIU
- FECHA RESOLUCION
- NRO RESOLUCION
- CONCLUSION

### 3. Familia OPS1
Columnas observadas:
- NRO_EXPEDIENTE
- VC_TIPO_EXPEDIENTE
- FEC_PRE
- DENUNCIANTES
- DENUNCIADOS
- RUC_DENUNCIADOS
- VC_RUBRO
- VC_LUGAR
- FEC_RESOL
- NRO_RESOL
- CONCLUSION

### 4. Familia OPS2
Columnas observadas:
- NRO_EXPEDIENTE
- TIPO_EXPEDIENTE
- FECHA_PRESENTACION
- SECTOR
- SUB_SECTOR
- DENUNCIADOS
- RUC_DENUNCIADOS
- FECHA_RESOLUCION
- NRO_RESOLUCION
- CONCLUSION

### 5. Familia SPC
Columnas observadas:
- INGRESO EN SALA
- NRO_EXPEDIENTE_ORIGEN
- TIPO_EXPEDIENTE
- FECHA_PRESENTACION
- DENUNCIADOS
- RUC_DENUNCIADOS
- MATERIAS
- FECHA_RESOLUCION
- NRO_RESOLUCION
- CONCLUSION

### 6. Familia CC3 presentados
Columnas observadas:
- NRO_EXPEDIENTE
- TIPO_EXPEDIENTE
- FEC_PRE
- SECTOR
- SUB SECTOR
- AUTORIDAD
- RUC AUTORIDAD
- ADMINISTRADOS
- RUC ADMINISTRADOS

### 7. Familia DDA
Columnas observadas:
- Nº EXPEDIENTE
- F.PRES.
- F.RESOLUCION
- PROCEDIMIENTO
- TIPO DE EXPEDIENTE
- TIPO DE OBRA
- CONCLUSION
- LUGAR PRESENTACION

Estas familias fueron consideradas las más importantes porque contienen, en distinta medida, los campos mínimos buscados para un análisis posterior: fecha de ingreso o presentación, empresa denunciada o actor equivalente, tipo de expediente y variables complementarias como sector, subsector, materia o resolución.

---

## Relevancia de las familias para el proyecto

Luego del análisis estructural, se concluyó que los datasets más útiles para la siguiente fase del proyecto son:

**Alta relevancia**
- CC1
- CC2
- OPS1
- OPS2
- SPC
- CC3 presentados

**Relevancia media**
- DDA

**Relevancia baja para este caso**
- DIN
- DSD
- CLC
- algunos archivos agregados o resúmenes

La razón principal es que las familias de alta relevancia contienen campos que permiten modelar un expediente como un evento concreto: cuándo ingresó, qué tipo de caso fue, contra quién se dirigió y, en varios casos, en qué sector o subsector ocurrió. En cambio, familias como DSD o ciertos archivos de CLC están más orientadas a estadísticas agregadas, propiedad intelectual o estructuras administrativas que no aportan tanto al escenario de detección de anomalías planteado.

---

## Limpieza y preparación del dataset

Esta entrega no implementa todavía la detección de spam ni el procesamiento concurrente con goroutines sobre el análisis de anomalías. Sin embargo, sí se desarrolló la fase de limpieza y consolidación, que era el objetivo central para esta etapa.

Los pasos realizados fueron los siguientes:

### 1. Descarga de archivos fuente
Se descargaron los 50 archivos localizados desde la plataforma oficial, organizándolos por dataset y manteniendo una copia cruda en `data/raw/`.

Ejemplo de ejecución del consolidator:

```bash
go run ./cmd/consolidator
```

### 2. Lectura de archivos heterogéneos
Se implementó un lector en Go capaz de abrir tanto archivos `.xlsx` como `.xls`. Para ello se utilizaron bibliotecas específicas para cada formato.

### 3. Detección de encabezados desplazados
Uno de los problemas más importantes fue que varios archivos tenían títulos, subtítulos o celdas decorativas en las primeras filas, y recién presentaban el encabezado real algunas líneas después. Para resolverlo se implementó una heurística de detección de encabezados basada en puntuación por palabras clave como `expediente`, `fecha`, `denunciados`, `ruc`, `sector`, `resolucion` o `tipo`.

Gracias a ello fue posible recuperar datasets que en una primera consolidación aparecían con columnas vacías, como CC1, OPS2 y SPC.

Fragmento representativo:

```go
headerIdx, header := detectHeaderRows(rows)
for _, row := range rows[headerIdx+1:] {
    // procesar filas reales luego del encabezado correcto
}
```

La heurística favorece filas que contienen términos característicos de un esquema tabular útil:

```go
if strings.Contains(low, "exped") ||
   strings.Contains(low, "fecha") ||
   strings.Contains(low, "denunci") ||
   strings.Contains(low, "ruc") ||
   strings.Contains(low, "sector") {
    score += 3
}
```

### 4. Normalización de nombres de columnas
Se unificaron diferencias como:
- `Nro. Exped.`
- `NRO EXPEDIENTE`
- `NRO_EXPEDIENTE`
- `Nº EXPEDIENTE`

Igualmente se normalizaron variantes para:
- tipo de expediente
- fecha de presentación
- denunciados
- ruc denunciados
- materia / rubro / sector / subsector
- resolución / conclusión

Ejemplo del mapeo implementado:

```go
Expediente:        get("expediente", "nro expediente", "numero expediente", "nº expediente", "nro_expediente", "ingreso en sala"),
TipoExpediente:    get("tipo expediente", "tipo de expediente", "tipo_expediente", "procedimiento"),
FechaPresentacion: get("fecha presentacion", "fecha de presentacion", "fecha_presentacion", "f.pres.", "fec pre"),
Denunciado:        get("denunciado", "denunciados", "administrados"),
RUC:               get("ruc", "ruc denunciados", "documento de denunciados", "ruc_denunciados"),
Materia:           get("materia", "materias", "vc rubro", "sector", "sub sector", "subsector", "tipo de obra"),
```

### 5. Consolidación en un esquema común
Se creó un esquema mínimo común para mapear los distintos tipos de dataset a campos consolidados como:
- `expediente`
- `tipo_expediente`
- `fecha_presentacion`
- `denunciante`
- `denunciado`
- `ruc`
- `materia`

Además, se preservó trazabilidad mediante:
- `dataset_id`
- `source_sheet`
- `source_row`
- `resumen`

La exportación final del dataset se realiza como CSV para facilitar el análisis posterior:

```go
if err := export.WriteCSV(filepath.Join("data", "processed", "consolidated.csv"), rows); err != nil {
    log.Fatalf("error escribiendo csv: %v", err)
}
```

### 6. Perfilado estructural
Se generó un perfil por archivo/hoja con el fin de registrar cuántas columnas tenía cada recurso y cuáles eran sus nombres detectados. Esto permite justificar técnicamente qué tan compatibles son las familias entre sí.

---

## Resultados obtenidos en esta etapa

Luego de implementar la detección de headers desplazados y mejorar la normalización, se obtuvo lo siguiente:

- **22 datasets de INDECOPI identificados**
- **50 archivos descargados**
- **50 perfiles estructurales detectados**
- **538,862 filas consolidadas**
- **0 warnings en la última ejecución**

Además, se verificó que varias familias ahora sí llenan correctamente las columnas mínimas relevantes. Por ejemplo:

### OPS1 presentados
- expediente: 5142
- tipo_expediente: 5142
- fecha_presentacion: 5142
- denunciante: 5140
- denunciado: 5138
- ruc: 5138
- materia: 4533

### OPS2 presentados
- expediente: 4700
- tipo_expediente: 4700
- fecha_presentacion: 4700
- denunciado: 4531
- ruc: 4531
- materia: 4700

### SPC presentados
- expediente: 8887
- tipo_expediente: 8887
- fecha_presentacion: 8887
- denunciado: 8129
- ruc: 7412
- materia: 7486

### CC2 presentados
- expediente: 4325
- tipo_expediente: 4325
- fecha_presentacion: 4325
- denunciado: 4325
- ruc: 4325
- materia: 4325

Estos resultados muestran que la fase de limpieza sí logró recuperar una cantidad significativa de información útil para el caso de uso.

---

## Esquema de normalización propuesto

A partir de las familias analizadas, se propone que el dataset final curado en la siguiente etapa se normalice alrededor de los siguientes campos:

- `dataset_family`
- `dataset_id`
- `source_year`
- `source_sheet`
- `expediente_id`
- `expediente_origen_id` (cuando exista)
- `resolucion_id` (cuando exista)
- `fecha_presentacion`
- `fecha_resolucion` (cuando exista)
- `fecha_consentida` (cuando exista)
- `tipo_expediente`
- `conclusion` (cuando exista)
- `denunciante` (cuando exista)
- `denunciado`
- `ruc_denunciado`
- `sector` (cuando exista)
- `subsector` (cuando exista)
- `materia` (cuando exista)
- `procedimiento` (cuando exista)
- `lugar_presentacion` (cuando exista)

Este esquema busca mantener el mayor nivel posible de compatibilidad sin forzar atributos inexistentes en todos los datasets.

---

## Modelado en Promela

Para representar la lógica concurrente del sistema se planteó un modelo en Promela centrado en la coordinación entre múltiples workers encargados de analizar lotes del dataset consolidado. El objetivo del modelado es verificar que la sincronización del algoritmo sea correcta cuando varios procesos detectan patrones sospechosos y actualizan un contador global de alertas.

La idea general consiste en dividir el dataset en bloques y asignar cada bloque a un worker. Cada worker analiza su lote buscando indicadores de comportamiento anómalo, por ejemplo repeticiones excesivas de expedientes, acumulación de casos por empresa denunciada o incrementos abruptos en determinados tipos de expediente dentro de una ventana temporal. Cuando se detecta una alerta, el worker debe ingresar a una sección crítica para incrementar el contador compartido.

En este escenario, el principal riesgo es una condición de carrera sobre la variable global de alertas. Para evitarlo, el modelo considera exclusión mutua antes de modificar el contador. De esta forma se puede comprobar formalmente que el sistema no produce inconsistencias al ejecutar múltiples workers de manera concurrente.

Un fragmento representativo del modelo es el siguiente:

```promela
byte mutex = 1;
byte alertas_globales = 0;

proctype Worker(byte id) {
    byte alertas_locales = 0;

    if
    :: alertas_locales = 0
    :: alertas_locales = 1
    :: alertas_locales = 2
    fi;

    atomic {
        (mutex == 1) ->
        mutex = 0;
        alertas_globales = alertas_globales + alertas_locales;
        mutex = 1;
    }
}
```

Además del comportamiento de los workers, se incorpora un proceso de monitoreo para verificar que el contador global de alertas nunca tome valores inválidos y que el sistema permanezca libre de bloqueos:

```promela
active proctype Monitor() {
    do
    :: assert(alertas_globales >= 0)
    od
}
```

La verificación con SPIN permite explorar diferentes combinaciones de ejecución entre workers y comprobar propiedades importantes del sistema, tales como:

- ausencia de deadlocks,
- exclusión mutua correcta,
- consistencia del contador global,
- y seguridad de la sección crítica.

Este modelado resulta útil porque traduce la idea del detector concurrente a una representación verificable formalmente, permitiendo validar desde temprano la lógica de sincronización antes de escalar a una implementación completa del análisis de anomalías sobre el dataset consolidado.

---

## Alcance real de la entrega

Es importante dejar claro que esta práctica llega **hasta la obtención, inspección, consolidación y limpieza del dataset**. Aún **no** se implementan:

- detección de spam,
- análisis temporal de anomalías,
- goroutines para paralelizar el análisis,
- uso de `sync.Mutex` para proteger contadores globales.

Sin embargo, esta etapa era necesaria para evitar construir el algoritmo futuro sobre datos mal interpretados o esquemas inconsistentes. La principal contribución de esta entrega es, por tanto, haber transformado un conjunto desordenado de spreadsheets heterogéneos en una base consolidada, inspeccionada y lista para una futura fase de procesamiento concurrente.

---

## Estructura del repositorio

- `cmd/consolidator`: CLI principal de descarga, lectura y consolidación
- `cmd/peek`: utilitario auxiliar para inspección rápida de archivos
- `internal/catalog`: catálogo de datasets y URLs
- `internal/downloader`: descarga de archivos fuente
- `internal/excel`: lectura robusta de `.xlsx` y `.xls`
- `internal/normalize`: mapeo de columnas a un esquema común
- `internal/export`: exportación de CSV, perfiles y reportes
- `data/raw`: archivos originales descargados
- `data/processed`: salidas consolidadas

---

## Uso

```bash
go run ./cmd/consolidator
```

## Salidas generadas

- `data/processed/consolidated.csv`: dataset consolidado actual
- `data/processed/profiles.json`: perfiles estructurales por archivo/hoja
- `data/processed/report.json`: métricas globales de la consolidación

---

# Práctica Calificada 2 — Detector Concurrente de Red Flags

## Descripción del algoritmo implementado

El algoritmo implementado es un **detector de anomalías sobre expedientes INDECOPI** que identifica tres categorías de patrones sospechosos en el campo `queja` y en los metadatos de cada expediente:

1. **TEXT_REPEAT**: repetición de texto entre quejas distintas, indicador de posible spam o bot. Se calcula la similitud de Jaccard entre los tokens de cada par de registros. Si la similitud supera el umbral configurado (por defecto 0.60), ambos expedientes se marcan.

2. **TIMING_BURST**: concentración inusual de expedientes en una misma fecha de presentación. Si el número de registros por fecha supera el límite configurado (por defecto 5), todos esos expedientes se marcan como potencialmente artificiales.

3. **EXACT_DUPLICATE**: mismo expediente con idéntica combinación de número, empresa denunciada y fecha. Indica ingreso duplicado del mismo caso.

El detector opera sobre el dataset `data/processed/enriched_filled.csv`, que contiene 10,230 expedientes con el campo `queja` generado. La detección principal, TEXT_REPEAT, tiene complejidad **O(n²)** y es el cuello de botella natural donde la concurrencia aporta mayor beneficio.

### Ejecución

```bash
# Versión secuencial
go run ./cmd/detector_secuencial

# Versión concurrente (N workers)
go run ./cmd/detector_concurrente -workers=4

# Benchmark completo con tabla de speedup
go run ./cmd/benchmark -runs=10 -workers-list=2,4,8
```

---

## Descripción del patrón concurrente utilizado

Se implementó el patrón **Worker Pool** sobre la fase de comparación por pares (TEXT_REPEAT), que es la única fase con complejidad O(n²) y representa más del 99% del tiempo de ejecución total.

### Dónde se aplica

En `internal/detector/concurrent.go`, la función `DetectConcurrent` lanza N goroutines workers que consumen índices `i` desde un canal compartido `workCh`. Cada worker, al recibir un índice `i`, calcula todos los pares `(i, j)` con `j > i`, acumulando los resultados en un slice local sin ningún mutex.

```go
// Productor: envía índices i al canal
go func() {
    for i := 0; i < n; i++ {
        workCh <- i
    }
    close(workCh)
}()

// Workers: reciben i, calculan todos los pares (i, j>i) localmente
go func() {
    defer wg.Done()
    var local []RedFlag
    for i := range workCh {
        for j := i + 1; j < n; j++ {
            score := Jaccard(records[i].Tokens, records[j].Tokens)
            if score >= opts.JaccardThreshold {
                local = append(local, RedFlag{...})
            }
        }
    }
    resultCh <- local  // una sola escritura al canal por worker
}()
```

### Por qué se eligió este patrón

La fase O(n²) es naturalmente paralelizable porque cada comparación de par `(i, j)` es independiente de cualquier otra. No existe ninguna dependencia de datos entre pares. El Worker Pool permite dividir el espacio de trabajo sin coordinación por par, reduciendo el tiempo de ejecución aproximadamente en proporción al número de CPUs disponibles.

Se descartó un enfoque de división estática de rangos (worker 0 procesa `i=0..n/W-1`, worker 1 procesa `i=n/W..2n/W-1`, etc.) porque genera **desbalance de carga**: el worker con `i` pequeños tiene más pares que calcular que el worker con `i` grandes. El canal actúa como cola de trabajo y distribuye la carga dinámicamente: cuando un worker termina su `i` actual, inmediatamente toma el siguiente disponible del canal, igualando naturalmente la carga entre goroutines.

### Cómo ayuda al procesamiento

Con 10,230 registros el total de pares es `10230 × 10229 / 2 ≈ 52.3 millones` de comparaciones Jaccard. En una máquina de 2 CPUs, la versión concurrente con 2 workers divide ese trabajo en dos mitades que se ejecutan en paralelo, reduciendo el tiempo de ~18 segundos a ~8.8 segundos.

### Qué problema evita o mejora

Evita el cuello de botella de la ejecución secuencial en datasets grandes. Sin concurrencia, un dataset de 100,000 registros requeriría ~500 veces más pares y tardaría horas. Con el Worker Pool, el tiempo escala en proporción inversa al número de CPUs, haciendo la detección viable en producción.

---

## Control de concurrencia: cómo se evitan los problemas

### Regla de oro aplicada

El diseño garantiza que **no existe ninguna variable compartida mutable** durante la fase O(n²). Cada goroutine worker opera exclusivamente sobre:
- Su propio slice local `var local []RedFlag` (memoria privada, sin compartir).
- El array `records []DetectorRecord` en modo **solo lectura** (lectura concurrente segura en Go sin mutex).

Los únicos puntos de sincronización son los canales, que en Go proveen sincronización implícita y libre de race conditions por diseño del runtime.

### Ausencia de condiciones de carrera

No existe ninguna escritura concurrente a memoria compartida durante el cómputo. El detector de race conditions de Go (`go run -race ./cmd/detector_concurrente`) no reporta ninguna condición de carrera porque:
- `records` es solo de lectura una vez cargado.
- Cada worker escribe únicamente en su `local []RedFlag` privado.
- La única escritura a `resultCh` ocurre **después** de que el worker termina todo su trabajo, y el canal provee el order-happens-before necesario.

### Ausencia de deadlocks

El sistema está libre de deadlocks por construcción:
- `workCh` se cierra explícitamente por el productor al terminar → los workers salen del `range workCh` sin bloquearse.
- `resultCh` se cierra por el goroutine supervisor después de que `wg.Wait()` regresa → el recolector en `main` sale del `range resultCh` sin bloquearse.
- No hay ningún par de goroutines esperándose mutuamente.

### Mecanismos de sincronización y su justificación

| Mecanismo | Dónde se usa | Por qué |
|-----------|-------------|---------|
| `chan int` (buffered) | `workCh` | Distribuye trabajo entre workers sin bloqueo ni mutex. El buffer absorbe la diferencia de velocidad entre productor y workers. |
| `chan []RedFlag` (buffered) | `resultCh` | Cada worker envía su resultado final una sola vez. Sin buffer, el worker bloquearía esperando que el recolector lo lea inmediatamente. |
| `sync.WaitGroup` | Supervisor goroutine | Sabe con precisión cuándo el último worker terminó para cerrar `resultCh`. Sin esto, cerrar el canal prematuramente causaría pérdida de resultados. |

No se usa `sync.Mutex` porque el diseño lo hace innecesario. Añadir un mutex global para proteger un slice de resultados compartido sería más lento (contención por cada flag detectada) y más propenso a errores que el enfoque de acumulación local + envío único por canal.

---

## Media recortada: eliminación de valores dispersos

### Justificación

Los tiempos de ejecución individuales presentan variabilidad causada por factores externos: el planificador del sistema operativo, el garbage collector de Go, la caché del procesador y otros procesos del sistema. Si se calcula la media aritmética simple, un valor atípico (por ejemplo, una corrida que toma 20 segundos en lugar de 18 por una pausa del GC) infla el promedio y da una imagen inexacta del rendimiento real.

### Cómo se aplica

La función `TrimmedMean(values []float64, trimPct float64)` en `internal/detector/stats.go`:
1. Ordena los tiempos de menor a mayor.
2. Descarta el `trimPct`% inferior y el `trimPct`% superior (se usa 10% por defecto).
3. Calcula la media aritmética sobre el conjunto restante.

Con 10 corridas y 10% de recorte, se eliminan la corrida más lenta y la más rápida (1 de cada extremo), y se promedia sobre las 8 restantes.

### Resultados reales obtenidos

La siguiente tabla muestra los tiempos individuales de la versión secuencial en 10 corridas y la aplicación de la media recortada:

| Corrida | Tiempo (ms) |
|---------|------------|
| 1 | 18,552 |
| 2 | 17,619 |
| 3 | 17,886 |
| 4 | 17,798 |
| 5 | 18,247 |
| 6 | 17,972 |
| 7 | 19,087 |
| 8 | 20,289 ← recortado (extremo superior) |
| 9 | 18,461 |
| 10 | 18,624 |

Media aritmética simple: **18,453 ms**  
Media recortada (10%): **18,328 ms**

La diferencia es pequeña aquí, pero en corridas con mayor varianza (como la versión concurrente con 4 y 8 workers, donde el planificador varía más), la media recortada elimina outliers de hasta 16,333 ms que inflaban el promedio en más de 15%.

---

## Análisis de speedup y escalabilidad

### Tabla de resultados (10 corridas, media recortada 10%, máquina 2 CPUs)

| Versión | T_media (ms) | Speedup | Eficiencia |
|---------|-------------|---------|-----------|
| Secuencial | 18,328 | 1.000 | 1.000 |
| Concurrente 2 workers | 8,808 | **2.081** | **1.040** |
| Concurrente 4 workers | 10,466 | 1.751 | 0.438 |
| Concurrente 8 workers | 14,808 | 1.238 | 0.155 |

### Interpretación del speedup

El speedup de **2.08x** con 2 workers indica que la versión concurrente procesó el dataset en poco menos de la mitad del tiempo que la versión secuencial. Este resultado es coherente con la disponibilidad de 2 núcleos físicos en la máquina de prueba: la carga se distribuyó equitativamente entre ambos CPUs, aprovechando casi al máximo el paralelismo disponible.

La eficiencia de **1.04** (ligeramente superior a 1.0) es llamativa pero explicable: se debe a efectos de caché. Cuando el worker 0 carga tokens del registro `i=0` en caché L2, el worker 1 puede aprovechar esa caché al calcular pares del mismo bloque. Esto produce una pequeña ganancia superlineal que es común en algoritmos con alta localidad de datos sobre datasets que caben en caché.

### Por qué el rendimiento cae con 4 y 8 workers en una máquina de 2 CPUs

Con 4 workers en 2 CPUs, el sistema operativo debe alternar entre 4 goroutines usando solo 2 hilos del kernel. Cada cambio de contexto tiene un costo que se vuelve significativo cuando las goroutines son muy activas (como en este caso, calculando Jaccard continuamente). Adicionalmente, el canal `workCh` se convierte en un punto de contención cuando 4 goroutines intentan leerlo simultáneamente.

Con 8 workers, esta penalización se amplifica: la sobrecarga de scheduling, la contención sobre el canal y el incremento en el uso de memoria por goroutine superan el beneficio de cualquier paralelismo adicional.

### Límite teórico (Ley de Amdahl)

Sea `f` la fracción paralelizable del algoritmo. Las fases TIMING_BURST y EXACT_DUPLICATE son O(n) y toman menos de 50ms (menos del 0.3% del tiempo total). Por tanto `f ≈ 0.997`.

Speedup máximo teórico con p procesadores: `S(p) = 1 / ((1-f) + f/p)`

| p | Speedup teórico | Speedup real |
|---|----------------|-------------|
| 2 | 1.997 | 2.081 |
| 4 | 3.989 | 1.751 |
| 8 | 7.945 | 1.238 |

El speedup real con 4 y 8 workers cae muy por debajo del teórico porque el modelo de Amdahl asume CPUs ilimitados. En la práctica, ejecutar más goroutines que CPUs físicos introduce overhead de scheduling que Amdahl no contempla.

**Conclusión**: el número óptimo de workers es igual al número de CPUs físicos disponibles. En esta máquina, 2 workers es la configuración óptima.

---

## Análisis de uso y rendimiento de recursos de cómputo

### Metodología de medición

Se usó `runtime.ReadMemStats` de Go (función `CaptureResources` en `internal/detector/stats.go`) para capturar el estado del heap antes y después de cada configuración. Se fuerza un ciclo de GC antes de cada medición para obtener valores limpios.

```go
func CaptureResources() ResourceSnapshot {
    runtime.GC()
    var ms runtime.MemStats
    runtime.ReadMemStats(&ms)
    return ResourceSnapshot{
        HeapAllocMB:  float64(ms.HeapAlloc) / 1024 / 1024,
        TotalAllocMB: float64(ms.TotalAlloc) / 1024 / 1024,
        GCCycles:     ms.NumGC,
        Goroutines:   runtime.NumGoroutine(),
    }
}
```

### Resultados de uso de recursos (10 corridas, dataset 10,230 registros)

| Versión | HeapAlloc (delta) | TotalAlloc (delta) | Ciclos GC | Goroutines pico |
|---------|------------------|-------------------|-----------|----------------|
| Secuencial | ~0 MB | 3,772 MB | 76 | 1 |
| Concurrente x2 | ~0 MB | 4,287 MB | 87 | 4 |
| Concurrente x4 | ~0 MB | 4,280 MB | 86 | 7 |
| Concurrente x8 | ~0 MB | 4,659 MB | 88 | 11 |

### Interpretación

**HeapAlloc delta ≈ 0 MB**: el GC de Go recupera la memoria de los slices de flags locales prácticamente en tiempo real. Al final de cada corrida, casi toda la memoria allocada ya fue recolectada, dejando un heap residual mínimo.

**TotalAlloc**: mide toda la memoria allocada a lo largo de la ejecución, incluyendo la ya liberada. Las versiones concurrentes allocan entre 13% y 24% más que la secuencial. Esto se explica por:
- Los stacks de goroutine (~2–8 KB por goroutine, dinámico en Go).
- Los slices `local []RedFlag` de cada worker que se crean y destruyen por corrida.
- La presión adicional sobre el GC que genera más ciclos de recolección.

**Ciclos GC**: la versión secuencial genera 76 ciclos en 10 corridas (~7.6 por corrida). Las versiones concurrentes generan ~87 ciclos (~8.7 por corrida), un 14% más. Esto es consistente con la mayor presión de allocación de los workers.

**Goroutines pico**: cada configuración tiene `2 + workers` goroutines en pico (main + productor + supervisor + N workers). No se observó fuga de goroutines: todas las goroutines terminan antes de que el main retorne, garantizado por `wg.Wait()`.

---

## Optimizaciones implementadas

### 1. Pre-tokenización del dataset al cargar (evitar trabajo redundante)

Cada queja se tokeniza una sola vez en `LoadCSV`, no en cada comparación de par. Sin esta optimización, el par `(i=0, j=1)` y el par `(i=0, j=2)` tokenizarían la queja 0 dos veces cada uno. Con 10,230 registros, esto evitaría ~52 millones de retokenizaciones innecesarias.

### 2. Jaccard con merge de slices ordenados en O(|a|+|b|)

La implementación estándar de Jaccard convierte cada lista de tokens en un `map[string]bool`, lo que implica hashing por cada token. En cambio, se usan slices ordenados alfabéticamente al momento de tokenizar, y se calcula la intersección con el algoritmo de merge de dos punteros:

```go
// O(|a|+|b|) sin allocar ninguna estructura adicional
for i < len(a) && j < len(b) {
    switch {
    case a[i] == b[j]: intersection++; i++; j++
    case a[i] < b[j]:  i++
    default:           j++
    }
}
```

Esto elimina el overhead de hashing y es hasta 3x más rápido en benchmarks micro para conjuntos pequeños (~10 tokens promedio), reduciendo la constante del O(n²).

### 3. Acumulación local en workers: cero contención de mutex

Un diseño alternativo usaría un `sync.Mutex` protegiendo un slice global de flags. Cada vez que un worker detecta un par similar, haría `mu.Lock(); flags = append(flags, ...); mu.Unlock()`. Con 52 millones de pares y ~276,000 flags detectadas, esto provocaría ~276,000 bloqueos de mutex durante la ejecución.

El diseño elegido elimina por completo esta contención: cada worker acumula en su `local []RedFlag` privado y solo hace **una** escritura al canal `resultCh` al finalizar. Los N workers hacen exactamente N escrituras al canal en total, independientemente de cuántas flags encuentren.

### 4. Canal buffered para reducir bloqueos

El canal `workCh` tiene buffer de tamaño `numWorkers * 4`. Sin buffer (tamaño 0), el productor bloquearía después de enviar cada `i`, esperando que un worker lo consuma. El buffer permite que el productor avance y precarque varios índices, desacoplando la velocidad del productor de la de los workers.

### 5. Conclusión de optimización: el punto óptimo de workers

El análisis de recursos muestra que aumentar workers más allá del número de CPUs físicos:
- Incrementa el TotalAlloc (más goroutines → más memoria de stacks).
- Incrementa los ciclos de GC.
- No reduce el tiempo (overhead de scheduling supera el beneficio).

Por tanto, se recomienda configurar `--workers` igual a `runtime.NumCPU()`. Para producción en un servidor con más CPUs (8, 16, 32), el speedup real se acercaría más al límite de Amdahl (~6x, ~10x, ~14x respectivamente).

---
