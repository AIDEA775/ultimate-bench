#!/bin/bash

# Arreglo de tamaños de problema (p) a evaluar. 
P_VALUES=(2 4 8 16 32 64 128 256 512 1024 2048 4096 8192 16384 32768 65536 131072 262144 524288 1048576)

# Función que ejecutará GNU parallel
run_sim() {
    local p=$1
    local slot=$2
    
    # Los cores suelen indexarse desde 0, así que restamos 1 al slot
    local core=$((slot - 1))
    
    # Ejecuta el binario forzando 1 core (GOMAXPROCS=1), pineado a un core específico con taskset
    output=$(taskset -c "$core" env GOMAXPROCS=1 ./simuleta --disable-checkpoints --duration 5 --p "$p" 2>&1)
    
    # Extraer el valor de la línea "Operaciones por Segundo (OPS): <valor>"
    ops=$(echo "$output" | grep "Operaciones por Segundo" | awk -F': ' '{print $2}')
    
    # Imprimir resultado en formato CSV
    echo "$p,$ops"
}

# Exportar la función para que 'parallel' pueda verla en las subshells
export -f run_sim

# Pasar los valores de P a parallel.
# -j 10 : ejecuta 10 trabajos en paralelo
# --keep-order (opcional): mantiene el orden de salida igual al de entrada
# {%} es el slot de parallel (de 1 a 10)
printf "%s\n" "${P_VALUES[@]}" | parallel -j 8 --keep-order run_sim {} {%}
