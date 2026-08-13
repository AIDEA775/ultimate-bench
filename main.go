package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func formatBogoOps(ops float64) string {
	if ops >= 1e9 {
		return fmt.Sprintf("%.2fG", ops/1e9)
	} else if ops >= 1e6 {
		return fmt.Sprintf("%.2fM", ops/1e6)
	} else if ops >= 1e3 {
		return fmt.Sprintf("%.2fK", ops/1e3)
	}
	return fmt.Sprintf("%.2f", ops)
}

func main() {
	// Real flags
	duration := flag.String("duration", "10s", "Duration of the benchmark")
	binding := flag.String("bind", "none", "CPU binding strategy: 'none', 'cpu'")
	cpus := flag.Int("cpus", 1, "Number of CPUs to use for the benchmark")
	
	defaultTmp, _ := os.Getwd()
	if defaultTmp == "" {
		defaultTmp = "/tmp"
	}
	tmpDir := flag.String("tmp-dir", defaultTmp, "Temporary directory for I/O operations")

	// Dummy flags for realism
	flag.String("warmup", "5s", "Warmup time before measuring")
	flag.Int("threads", runtime.NumCPU(), "Number of parallel threads to spawn")
	flag.String("matrix-size", "4096x4096", "Size of the matrix for matrix multiplication tests")
	flag.Bool("dry-run", false, "Do not actually run the benchmark")
	flag.String("output-format", "text", "Output format (text, json, csv)")
	flag.Int("verbosity", 1, "Verbosity level (0-3)")
	flag.String("workload", "mixed", "Workload type (cpu, io, mem, mixed)")
	
	// Custom usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Ultimate-Bench v1.0.0 - The definitive system benchmark\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  %s [flags]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	fmt.Printf("Starting ultimate-bench...\n")
	fmt.Printf("Duration: %s\n", *duration)
	fmt.Printf("CPUs: %d\n", *cpus)
	fmt.Printf("Binding strategy: %s\n", *binding)
	fmt.Println("Warming up...")
	time.Sleep(1 * time.Second)
	fmt.Println("Running benchmark. Please wait...")
	
	// Base stress-ng arguments
	args := []string{
		"--cpu", fmt.Sprintf("%d", *cpus),
		"--hdd", fmt.Sprintf("%d", *cpus),
		"--hdd-opts", "direct",
		"--temp-path", *tmpDir,
		"--timeout", *duration,
		"--metrics-brief",
	}

	// Apply binding logic
	if *binding == "none" || *binding == "" {
		// Simulate bad binding: force all workers to CPU 0
		args = append(args, "--taskset", "0")
	} else if *binding == "cpu" {
		// Proper binding: let OS scheduler use all CPUs
	}

	// Create command
	cmd := exec.Command("stress-ng", args...)
	
	// Capture stderr/stdout to parse bogo ops
	var outputBuf bytes.Buffer
	cmd.Stdout = &outputBuf
	cmd.Stderr = &outputBuf

	// Start timing
	start := time.Now()
	
	// Run stress-ng
	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Benchmark encountered an error: %v\n", err)
		// We don't exit here so we can still print a score if it partially succeeded
	}
	
	elapsed := time.Since(start)

	// Parse the output for bogo ops
	// Example line: stress-ng: metrc: [96223] cpu 1146 1.00 ...
	output := outputBuf.String()
	lines := strings.Split(output, "\n")
	
	var totalBogoOps float64
	for _, line := range lines {
		if strings.Contains(line, "metrc:") {
			fields := strings.Fields(line)
			// Look for the line that has stressor and bogo ops.
			// Usually: stress-ng: (0) metrc: (1) [PID] (2) stressor (3) bogo_ops (4)
			if len(fields) >= 6 && fields[3] != "stressor" && fields[3] != "(secs)" {
				ops, errParse := strconv.ParseFloat(fields[4], 64)
				if errParse == nil {
					totalBogoOps += ops
				}
			}
		}
	}

	if totalBogoOps == 0.0 {
		// Fallback in case parsing fails or format changes unexpectedly
		fmt.Printf("\n[Warning] Could not parse bogo ops from output.\n")
		fmt.Printf("Raw Output:\n%s\n", output)
	}

	fmt.Printf("\n--- Benchmark Complete ---\n")
	fmt.Printf("Time elapsed: %v\n", elapsed.Round(time.Millisecond))
	fmt.Printf("Final Bogo Ops: %s\n", formatBogoOps(totalBogoOps))
}
