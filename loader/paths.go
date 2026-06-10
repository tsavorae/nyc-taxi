package main

import (
	"os"
	"path/filepath"
	"time"
)

const (
	maxTripDistanceMiles    = 100.0
	maxTripDurationMinutes  = 24 * 60
	minTripDurationMinutes  = 1.0
)

var parquetFiles = []string{
	"yellow_tripdata_2026-01.parquet",
	"yellow_tripdata_2026-02.parquet",
	"yellow_tripdata_2026-03.parquet",
}

var (
	dateStart = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	dateEnd   = time.Date(2026, 3, 31, 23, 59, 59, 0, time.UTC)
)

type paths struct {
	root        string
	rawDataDir  string
	lookupPath  string
	processedDir string
}

func resolvePaths() paths {
	root := projectRoot()
	return paths{
		root:         root,
		rawDataDir:   filepath.Join(root, "data", "raw dataset"),
		lookupPath:   filepath.Join(root, "data", "taxi_zone_lookup.csv"),
		processedDir: filepath.Join(root, "data", "processed"),
	}
}

func projectRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	if filepath.Base(wd) == "loader" {
		return filepath.Dir(wd)
	}
	return wd
}
