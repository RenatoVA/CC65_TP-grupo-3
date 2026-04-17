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

## Conclusiones (PC1)

La exploración de datasets de INDECOPI mostró que el principal desafío de esta práctica no era la descarga de archivos, sino la heterogeneidad estructural de los datos publicados. Lejos de ser una colección uniforme exportada desde una sola base de datos, los recursos encontrados presentan encabezados desplazados, títulos embebidos, variaciones semánticas de columnas y diferencias fuertes entre familias institucionales.

A pesar de ello, fue posible identificar grupos de datasets con suficiente compatibilidad para una futura consolidación curada, especialmente CC1, CC2, OPS1, OPS2 y SPC. Asimismo, la implementación de un parser más robusto permitió recuperar una gran parte de la información que inicialmente parecía inutilizable.

En consecuencia, la práctica deja como resultado una base de datos consolidada y una propuesta concreta de normalización, lo que reduce sustancialmente la incertidumbre para la siguiente fase del proyecto. Gracias a este trabajo previo, la implementación posterior del detector concurrente podrá enfocarse realmente en la lógica de anomalías, en lugar de desperdiciar esfuerzo corrigiendo inconsistencias básicas de origen.

---

