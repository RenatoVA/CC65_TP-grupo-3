/*
 * detector_concurrente.pml
 *
 * Modelo Promela del detector CONCURRENTE actual usado por benchfull en
 * count-only: BuildKeywordIndex se construye una vez y luego TEXT_REPEAT se
 * reparte por rangos de registros entre workers.
 *
 * Lo que se verifica:
 *   - no deadlock: todos los workers terminan y el recolector termina
 *   - exclusion mutua: solo un worker puede entrar al merge global de stats
 *   - bound del WaitGroup: done_count nunca supera N_WORKERS
 *   - consistencia: flags_global nunca es negativo
 *
 * Verificacion:
 *   spin -search promela/detector_concurrente.pml
 *   spin -search -ltl no_deadlock promela/detector_concurrente.pml
 *   spin -search -ltl mutex_ok promela/detector_concurrente.pml
 */

#define N_REGISTROS     4
#define N_WORKERS       2
#define MAX_CANDIDATOS  2
#define MIN_SHARED      2
#define BURST_LIMITE    2

mtype = {
    INIT,
    BUILD_DF,
    BUILD_INDEX,
    TEXT_REPEAT,
    MERGE_STATS,
    LINEAR_PHASES,
    FIN
};

mtype fase = INIT;

int flags_global = 0;
int comparisons_global = 0;
int accepted_pairs_global = 0;

byte done_count = 0;
byte merge_owner = 255;
byte in_merge = 0;

bool index_ready = false;
bool workers_done = false;
bool recolectado = false;
bool terminado = false;

ltl no_deadlock { <> terminado }
ltl mutex_ok    { [] (in_merge <= 1) }
ltl wg_bound    { [] (done_count <= N_WORKERS) }
ltl flags_ok    { [] (flags_global >= 0) }

inline add_global_flags(delta) {
    flags_global = flags_global + delta;
    assert(flags_global >= 0)
}

inline verificar_jaccard(local_flags, local_comps, local_accepted) {
    bool supera_umbral;

    local_comps = local_comps + 1;

    if
    :: supera_umbral = true
    :: supera_umbral = false
    fi;

    if
    :: supera_umbral ->
        local_accepted = local_accepted + 1;
        local_flags = local_flags + 2
    :: else -> skip
    fi
}

inline evaluar_candidato(local_flags, local_comps, local_accepted, selected) {
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
        verificar_jaccard(local_flags, local_comps, local_accepted)
    :: else -> skip
    fi
}

proctype Worker(byte id) {
    byte i;
    byte j;
    byte lo;
    byte hi;
    byte selected;
    int local_flags = 0;
    int local_comps = 0;
    int local_accepted = 0;

    (index_ready == true);

    if
    :: id == 0 ->
        lo = 0;
        hi = 2
    :: id == 1 ->
        lo = 2;
        hi = N_REGISTROS
    fi;

    i = lo;
    do
    :: i < hi ->
        selected = 0;
        j = i + 1;
        do
        :: j < N_REGISTROS ->
            evaluar_candidato(local_flags, local_comps, local_accepted, selected);
            j = j + 1
        :: else -> break
        od;
        assert(selected <= MAX_CANDIDATOS);
        i = i + 1
    :: else -> break
    od;

    /*
     * Modela la zona critica de acumulacion global:
     * atomic.AddUint64(...) en Go para flags, comparisons y accepted pairs.
     */
    atomic {
        in_merge = in_merge + 1;
        merge_owner = id;
        assert(in_merge == 1);

        flags_global = flags_global + local_flags;
        comparisons_global = comparisons_global + local_comps;
        accepted_pairs_global = accepted_pairs_global + local_accepted;
        assert(flags_global >= 0);
        assert(accepted_pairs_global <= comparisons_global);

        merge_owner = 255;
        in_merge = in_merge - 1;

        done_count = done_count + 1;
        if
        :: done_count == N_WORKERS -> workers_done = true
        :: else -> skip
        fi
    }
}

active proctype RecolectorLineal() {
    byte conteo_fecha;
    bool duplicado;
    bool keyword_spam;
    byte i;

    (workers_done == true);
    fase = LINEAR_PHASES;

    if
    :: conteo_fecha = BURST_LIMITE + 1 -> add_global_flags(conteo_fecha)
    :: conteo_fecha = 1 -> skip
    fi;

    if
    :: duplicado = true -> add_global_flags(2)
    :: duplicado = false -> skip
    fi;

    i = 0;
    do
    :: i < N_REGISTROS ->
        if
        :: keyword_spam = true -> add_global_flags(1)
        :: keyword_spam = false -> skip
        fi;
        i = i + 1
    :: else -> break
    od;

    recolectado = true;
    fase = FIN;
    terminado = true
}

init {
    byte i;

    fase = BUILD_DF;
    i = 0;
    do
    :: i < N_REGISTROS ->
        /* DF global: lectura de registros y conteo de tokens. */
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

    index_ready = true;
    fase = TEXT_REPEAT;

    atomic {
        run Worker(0);
        run Worker(1)
    }
}
