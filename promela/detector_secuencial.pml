/*
 * detector_secuencial.pml
 *
 * Modelo Promela del detector SECUENCIAL actual.
 *
 * Flujo modelado:
 *   1. TEXT_REPEAT:
 *      - construir DF global
 *      - seleccionar keywords
 *      - construir indice invertido de prefijos
 *      - generar candidatos por keywords compartidas
 *      - aplicar filtros de longitud/cota superior
 *      - verificar Jaccard final
 *   2. TIMING_BURST
 *   3. EXACT_DUPLICATE
 *   4. KEYWORD_SPAM
 *
 * El modelo abstrae el contenido real de tokens con elecciones no deterministas
 * acotadas. La propiedad importante es el orden de fases, terminacion y que los
 * contadores de flags no entren en estados invalidos.
 *
 * Verificacion:
 *   spin -search promela/detector_secuencial.pml
 *   spin -search -ltl no_deadlock promela/detector_secuencial.pml
 *   spin -search -ltl orden_fases promela/detector_secuencial.pml
 */

#define N_REGISTROS     4
#define MAX_CANDIDATOS  2
#define MIN_SHARED      2
#define BURST_LIMITE    2

mtype = {
    INIT,
    BUILD_DF,
    BUILD_INDEX,
    TEXT_REPEAT,
    TIMING_BURST,
    EXACT_DUPLICATE,
    KEYWORD_SPAM,
    FIN
};

mtype fase = INIT;

int flags_repeat  = 0;
int flags_burst   = 0;
int flags_dup     = 0;
int flags_keyword = 0;
int flags_total   = 0;

bool terminado = false;

ltl no_deadlock { <> terminado }
ltl flags_ok    { [] (flags_total >= 0) }
ltl orden_fases { [] ((fase == EXACT_DUPLICATE) -> (flags_total >= flags_repeat + flags_burst)) }

inline add_flags(delta) {
    flags_total = flags_total + delta;
    assert(flags_total >= 0)
}

inline verificar_jaccard(i, j) {
    bool supera_umbral;

    if
    :: supera_umbral = true
    :: supera_umbral = false
    fi;

    if
    :: supera_umbral ->
        flags_repeat = flags_repeat + 2;
        add_flags(2)
    :: else -> skip
    fi
}

inline evaluar_candidato(i, j, selected) {
    byte shared;
    bool pasa_min_shared;
    bool pasa_length_filter;
    bool pasa_upper_bound;

    if
    :: shared = 0
    :: shared = 1
    :: shared = MIN_SHARED
    fi;

    pasa_min_shared = (shared >= MIN_SHARED);

    if
    :: pasa_length_filter = true
    :: pasa_length_filter = false
    fi;

    if
    :: pasa_upper_bound = true
    :: pasa_upper_bound = false
    fi;

    if
    :: pasa_min_shared && pasa_length_filter && pasa_upper_bound && selected < MAX_CANDIDATOS ->
        selected = selected + 1;
        verificar_jaccard(i, j)
    :: else -> skip
    fi
}

active proctype DetectorSecuencial() {
    byte i;
    byte j;
    byte selected;
    byte conteo_fecha;
    bool duplicado;
    bool keyword_spam;

    fase = BUILD_DF;
    i = 0;
    do
    :: i < N_REGISTROS ->
        /* DF global: cada registro aporta tokens normalizados. */
        i = i + 1
    :: else -> break
    od;

    fase = BUILD_INDEX;
    i = 0;
    do
    :: i < N_REGISTROS ->
        /* Indice invertido de prefijos keyword-index. */
        i = i + 1
    :: else -> break
    od;

    fase = TEXT_REPEAT;
    i = 0;
    do
    :: i < N_REGISTROS ->
        selected = 0;
        j = i + 1;
        do
        :: j < N_REGISTROS ->
            evaluar_candidato(i, j, selected);
            j = j + 1
        :: else -> break
        od;
        assert(selected <= MAX_CANDIDATOS);
        i = i + 1
    :: else -> break
    od;

    fase = TIMING_BURST;
    if
    :: conteo_fecha = BURST_LIMITE + 1 ->
        flags_burst = flags_burst + conteo_fecha;
        add_flags(conteo_fecha)
    :: conteo_fecha = 1 -> skip
    fi;

    fase = EXACT_DUPLICATE;
    if
    :: duplicado = true ->
        flags_dup = flags_dup + 2;
        add_flags(2)
    :: duplicado = false -> skip
    fi;

    fase = KEYWORD_SPAM;
    i = 0;
    do
    :: i < N_REGISTROS ->
        if
        :: keyword_spam = true ->
            flags_keyword = flags_keyword + 1;
            add_flags(1)
        :: keyword_spam = false -> skip
        fi;
        i = i + 1
    :: else -> break
    od;

    fase = FIN;
    assert(flags_total == flags_repeat + flags_burst + flags_dup + flags_keyword);
    terminado = true
}
