package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

func printStats(stats []fileStats, label string, elapsed time.Duration) {
	fmt.Printf("\n%s\n  %s\n%s\n", sep(), label, sep())

	totalRaw := 0
	totalClean := 0
	sort.Slice(stats, func(i, j int) bool { return stats[i].File < stats[j].File })

	for _, row := range stats {
		removed := row.RawRows - row.CleanRows
		pct := 0.0
		if row.RawRows > 0 {
			pct = float64(removed) / float64(row.RawRows) * 100
		}
		fmt.Printf("  %s: %s → %s (%s removidas, %.2f%%)\n",
			row.File,
			formatInt(row.RawRows),
			formatInt(row.CleanRows),
			formatInt(removed),
			pct,
		)
		totalRaw += row.RawRows
		totalClean += row.CleanRows
	}

	totalRemoved := totalRaw - totalClean
	totalPct := 0.0
	if totalRaw > 0 {
		totalPct = float64(totalRemoved) / float64(totalRaw) * 100
	}
	fmt.Printf("\n  Total: %s → %s (%s removidas, %.2f%%)\n",
		formatInt(totalRaw),
		formatInt(totalClean),
		formatInt(totalRemoved),
		totalPct,
	)
	fmt.Printf("  Tiempo: %.2fs\n", elapsed.Seconds())
}

func processSequential(p paths, lookup map[int32]zoneInfo) (time.Duration, []fileStats, error) {
	start := time.Now()
	stats := make([]fileStats, 0, len(parquetFiles))

	for _, fileName := range parquetFiles {
		inputPath := filepath.Join(p.rawDataDir, fileName)
		tmpDir, err := os.MkdirTemp("", "nyc_taxi_seq_")
		if err != nil {
			return 0, nil, err
		}
		outputPath := filepath.Join(tmpDir, "clean_"+fileName)
		result, err := cleanParquetFile(inputPath, outputPath, lookup)
		os.RemoveAll(tmpDir)
		if err != nil {
			return 0, nil, err
		}
		stats = append(stats, result)
	}

	return time.Since(start), stats, nil
}

func processConcurrent(p paths, lookup map[int32]zoneInfo, workers int, tmpDir string) (time.Duration, []fileStats, error) {
	start := time.Now()
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	stats := make([]fileStats, 0, len(parquetFiles))
	errCh := make(chan error, len(parquetFiles))

	for _, fileName := range parquetFiles {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			inputPath := filepath.Join(p.rawDataDir, name)
			outputPath := filepath.Join(tmpDir, "clean_"+name)
			result, err := cleanParquetFile(inputPath, outputPath, lookup)
			if err != nil {
				errCh <- err
				return
			}
			mu.Lock()
			stats = append(stats, result)
			mu.Unlock()
		}(fileName)
	}

	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			return 0, nil, err
		}
	}

	return time.Since(start), stats, nil
}

func runBenchmark(workers int) error {
	p := resolvePaths()
	lookup, err := loadZoneLookup(p.lookupPath)
	if err != nil {
		return fmt.Errorf("load zone lookup: %w", err)
	}

	fmt.Println("Iniciando limpieza secuencial...")
	timeSeq, statsSeq, err := processSequential(p, lookup)
	if err != nil {
		return err
	}
	printStats(statsSeq, "LIMPIEZA SECUENCIAL", timeSeq)

	fmt.Println("\nIniciando limpieza concurrente...")
	tmpDir, err := os.MkdirTemp("", "nyc_taxi_conc_")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	timeConc, statsConc, err := processConcurrent(p, lookup, workers, tmpDir)
	if err != nil {
		return err
	}
	printStats(statsConc, "LIMPIEZA CONCURRENTE", timeConc)

	speedup := timeSeq.Seconds() / timeConc.Seconds()
	if timeConc.Seconds() == 0 {
		speedup = 0
	}

	fmt.Printf("\n%s\n  BENCHMARK DE SPEEDUP\n%s\n", sep(), sep())
	fmt.Printf("  Tiempo secuencial:  %.2fs\n", timeSeq.Seconds())
	fmt.Printf("  Tiempo concurrente: %.2fs\n", timeConc.Seconds())
	fmt.Printf("  Speedup:            %.2fx\n", speedup)
	fmt.Println(sep())

	sort.Slice(statsConc, func(i, j int) bool { return statsConc[i].File < statsConc[j].File })
	paths := make([]string, len(statsConc))
	for i, s := range statsConc {
		paths[i] = s.OutputPath
	}

	outputPath := filepath.Join(p.processedDir, "yellow_tripdata_2026_clean.parquet")
	totalRows, err := mergeParquetFiles(paths, outputPath)
	if err != nil {
		return fmt.Errorf("merge parquet files: %w", err)
	}

	fmt.Printf("\nDatos limpios guardados en: %s\n", outputPath)
	fmt.Printf("Filas finales: %s\n", formatInt(totalRows))

	return printSample(outputPath)
}

func sep() string {
	return "============================================================"
}

func formatInt(n int) string {
	s := fmt.Sprintf("%d", n)
	if n < 1000 {
		return s
	}

	var out []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(c))
	}
	return string(out)
}

func main() {
	workers := flag.Int("workers", len(parquetFiles), "número de workers concurrentes")
	flag.Parse()

	if *workers < 1 {
		fmt.Fprintln(os.Stderr, "workers debe ser >= 1")
		os.Exit(1)
	}

	if err := runBenchmark(*workers); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
