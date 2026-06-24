package main

import (
	"log"
	"os"

	"github.com/parquet-go/parquet-go"
)

type ParquetRow struct {
	TripDistance    float64 `parquet:"trip_distance"`
	PickupHour      int32   `parquet:"pickup_hour"`
	PickupDayOfWeek int32   `parquet:"pickup_day_of_week"`
	PUBorough       string  `parquet:"pu_borough"`
	TripDuration    int64   `parquet:"trip_duration_minutes"`
}

// boroughToFloat encodes the borough string as a numeric feature.
var boroughIndex = map[string]float64{
	"Bronx":         0,
	"Brooklyn":      1,
	"EWR":           2,
	"Manhattan":     3,
	"Queens":        4,
	"Staten Island": 5,
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
			dur := float64(row.TripDuration)
			if dur <= 0 || dur > 180 {
				continue
			}
			if row.TripDistance <= 0 {
				continue
			}
			borough, ok := boroughIndex[row.PUBorough]
			if !ok {
				continue // skip unknown boroughs
			}
			samples = append(samples, Sample{
				Features: [4]float64{
					row.TripDistance,
					float64(row.PickupHour),
					float64(row.PickupDayOfWeek),
					borough,
				},
				Target: dur,
			})
		}
		if err != nil {
			break
		}
	}

	log.Printf("parquet loaded: %d valid samples\n", len(samples))
	return samples, nil
}
