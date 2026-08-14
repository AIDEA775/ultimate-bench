# Ultimate Bench

Un simulador HPC escrito en Go para generar cargas de trabajo intensivas de CPU y memoria, ideal para realizar benchmarks y analizar la latencia de caché.

## Compilación

Para compilar el binario del simulador, simplemente ejecuta el siguiente comando en la raíz del proyecto:

```bash
go build -o simuleta main.go
```

## Uso y Flags Importantes

Puedes ejecutar el simulador directamente. Las flags o opciones más importantes (aquellas que realmente afectan el comportamiento de la simulación) son:

- `-duration`: Duración de la simulación en segundos (por defecto `10`).
- `-workers`: Cantidad de rutinas/workers concurrentes a ejecutar (por defecto `1`).
- `-p`: Tamaño del problema o arreglo por iteración. Afecta directamente los saltos de memoria y los posibles *cache misses* (por defecto `8192`).
- `-disable-checkpoints`: Flag booleano para deshabilitar por completo la escritura de archivos temporales al disco. Útil si solo te interesa testear CPU/RAM y no I/O.
- `-checkpoint-dir`: Directorio absoluto donde se guardarán los checkpoints (por defecto la carpeta actual).
- `-checkpoint-interval`: Cada cuántos segundos hacer un checkpoint a disco (por defecto `2`).
- `-v`: Modo verboso. Útil para debugear las barreras de sincronización.


Comando ~óptimo
```
strace -e openat ./simuleta --workers 12
GOMAXPROCS=12 ./simuleta --workers 12 --checkpoint-interval 5 --checkpoint-dir /dev/shm/ --duration 20 --p 2048
```


## Ejecutar el Benchmark

Si quieres correr pruebas automatizadas con distintos tamaños de memoria (`p`) para ver la caída de rendimiento por las cachés L1/L2/L3, puedes usar el script de bash incluido (requiere GNU parallel):

```bash
./run_benchmark.sh
```

Esto generará un archivo `results.csv` con los OPS (operaciones por segundo) para cada tamaño de problema testeado.
