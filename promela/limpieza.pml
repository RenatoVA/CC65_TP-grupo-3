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
active proctype Monitor() {
    do
    :: assert(alertas_globales >= 0)
    od
}