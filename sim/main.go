package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

const CheckpointSizeBytes = 50 * 1024 * 1024 // 50MB por defecto, ajustable a mano

func main() {
	// GOMAXPROCS default logic: si no está definida la variable de entorno, limitarlo a 1.
	if os.Getenv("GOMAXPROCS") == "" {
		runtime.GOMAXPROCS(1)
	}

	pwd, err := os.Getwd()
	if err != nil {
		pwd = "/tmp"
	}

	// Flags
	durationSec := flag.Int("duration", 10, "Duración de la simulación en segundos")
	workers := flag.Int("workers", 1, "Cantidad de workers (goroutines) concurrentes")
	checkpointDir := flag.String("checkpoint-dir", pwd, "Directorio absoluto para escribir checkpoints")
	checkpointInterval := flag.Int("checkpoint-interval", 2, "Cantidad de segundos entre cada checkpoint")
	disableCheckpoints := flag.Bool("disable-checkpoints", false, "Deshabilitar por completo la escritura de checkpoints")
	p := flag.Int("p", 1000, "Tamaño del problema (elementos del arreglo por iteración)")
	flag.Parse()

	fmt.Printf("Iniciando simulación HPC (Pid: %d)\n", os.Getpid())
	fmt.Printf("Tamaño del problema (p): %d\n", *p)

	// Ensure checkpoint directory exists
	if !*disableCheckpoints {
		if err := os.MkdirAll(*checkpointDir, 0755); err != nil {
			log.Fatalf("Error creando directorio de checkpoints: %v", err)
		}
		// Cleanup at start
		cleanupCheckpoints(*checkpointDir, *workers)
	}

	// Setup simulation variables
	var totalOps atomic.Uint64
	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*durationSec)*time.Second)
	defer cancel()

	startTime := time.Now()

	// Launch workers
	for w := 0; w < *workers; w++ {
		wg.Add(1)
		go worker(ctx, w, *p, *checkpointDir, *checkpointInterval, *disableCheckpoints, &totalOps, &wg)
	}

	// Wait for workers to finish
	wg.Wait()
	elapsed := time.Since(startTime)

	// Cleanup at end
	if !*disableCheckpoints {
		cleanupCheckpoints(*checkpointDir, *workers)
	}

	// Final Metrics
	ops := totalOps.Load()
	opsPerSec := float64(ops) / elapsed.Seconds()
	
	fmt.Printf("Simulación finalizada en %v\n", elapsed)
	fmt.Printf("Operaciones por Segundo (OPS): %s\n", formatOPS(opsPerSec))
}

func formatOPS(ops float64) string {
	if ops >= 1e9 {
		return fmt.Sprintf("%.2f G", ops/1e9)
	} else if ops >= 1e6 {
		return fmt.Sprintf("%.2f M", ops/1e6)
	} else if ops >= 1e3 {
		return fmt.Sprintf("%.2f K", ops/1e3)
	}
	return fmt.Sprintf("%.2f", ops)
}

func cleanupCheckpoints(dir string, workers int) {
	for i := 0; i < workers; i++ {
		filename := filepath.Join(dir, fmt.Sprintf("checkpoint_worker_%d.tmp", i))
		os.Remove(filename)
	}
}

func worker(ctx context.Context, id int, p int, checkpointDir string, checkpointIntervalSec int, disableCheckpoints bool, totalOps *atomic.Uint64, wg *sync.WaitGroup) {
	defer wg.Done()

	// Asignar un arreglo propio para forzar lecturas/escrituras en memoria y evidenciar cache misses
	data := make([]float64, p)
	for i := range data {
		data[i] = rand.Float64()
	}

	iterations := 0
	checkpointFilename := filepath.Join(checkpointDir, fmt.Sprintf("checkpoint_worker_%d.tmp", id))
	lastCheckpoint := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// CPU Intensive work: simulación con acceso pseudoaleatorio dependiente de los datos.
		// Al hacer que el siguiente índice dependa del valor cargado de la memoria en esta iteración,
		// obligamos al procesador a serializar las lecturas (derrotamos el Out-of-Order execution y el Memory Level Parallelism).
		// Esto revelará la latencia real de la caché L1, L2 y L3 de forma escalonada.
		sum := 0.0
		var state uint32 = uint32(id+1) + uint32(iterations) // seed
		idx := 0
		for i := 0; i < p; i++ {
			// Hacer que el PRNG dependa de la lectura ANTERIOR de memoria
			bits := math.Float64bits(data[idx])
			state ^= uint32(bits)
			
			// Fast Xorshift32
			state ^= state << 13
			state ^= state >> 17
			state ^= state << 5
			
			// Lemire's method
			idx = int((uint64(state) * uint64(p)) >> 32)
			
			data[idx] = data[idx]*1.0000001 + 0.0000001
			sum += data[idx]
		}
		
		// Prevenir optimización total del compilador
		_ = sum 

		// Track ops. Cada loop hace `p` operaciones
		totalOps.Add(uint64(p))
		iterations++

		// IO operation (checkpoint)
		if !disableCheckpoints && time.Since(lastCheckpoint) >= time.Duration(checkpointIntervalSec)*time.Second {
			writeCheckpoint(checkpointFilename)
			lastCheckpoint = time.Now()
		}
	}
}

func writeCheckpoint(filename string) {
	// Escribir a disco usando file IO, util para observar con strace
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return // ignorar errores para no frenar la simulacion si el disco falla
	}
	defer f.Close()
	
	// Escribir en bloques de 1MB para no saturar memoria RAM
	chunk := make([]byte, 1024*1024)
	for i := range chunk {
		chunk[i] = 1 // Datos no nulos para evitar optimizaciones de sparse files
	}
	
	bytesWritten := 0
	for bytesWritten < CheckpointSizeBytes {
		n, _ := f.Write(chunk)
		bytesWritten += n
	}
	f.Sync() // forzar fsync para generar IO wait
}
