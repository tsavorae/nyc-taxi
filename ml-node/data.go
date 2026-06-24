package main

import (
	"log"
	"os"

	"github.com/parquet-go/parquet-go"
)

// ParquetRow mirrors the columns in yellow_tripdata_2026_clean.parquet.
// Adjust field names if your loader saved them differently.
type ParquetRow struct {
	TripDistance    float64 `parquet:"trip_distance"`
	PickupHour      int32   `parquet:"pickup_hour"`
	PickupDayOfWeek int32   `parquet:"pickup_day_of_week"`
	PUBorough       int32   `parquet:"pu_borough"` // encoded as int by loader
	TripDuration    float64 `parquet:"trip_duration"`
}

func loadData(path string) ([]Sample, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}

	pf, err := parquet.OpenFile(f, stat.Size())
	if err != nil {
		return nil, err
	}

	reader := parquet.NewGenericReader[ParquetRow](pf)
	defer reader.Close()

	batch := make([]ParquetRow, 1000)
	var samples []Sample

	for {
		n, err := reader.Read(batch)
		for _, row := range batch[:n] {
			if row.TripDuration <= 0 || row.TripDuration > 180 {
				continue
			}
			if row.TripDistance <= 0 {
				continue
			}
			samples = append(samples, Sample{
				Features: [4]float64{
					row.TripDistance,
					float64(row.PickupHour),
					float64(row.PickupDayOfWeek),
					float64(row.PUBorough),
				},
				Target: row.TripDuration,
			})
		}
		if err != nil {
			break // io.EOF is normal here
		}
	}

	log.Printf("parquet loaded: %d valid samples\n", len(samples))
	return samples, nil
}
