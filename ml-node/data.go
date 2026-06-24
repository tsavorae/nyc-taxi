package main

import (
	"encoding/csv"
	"log"
	"os"
	"strconv"
)

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

	r := csv.NewReader(f)
	r.Read() // skip header

	var samples []Sample
	for {
		rec, err := r.Read()
		if err != nil {
			break
		}
		dist, _ := strconv.ParseFloat(rec[0], 64)
		hour, _ := strconv.ParseFloat(rec[1], 64)
		dow, _ := strconv.ParseFloat(rec[2], 64)
		borough := boroughIndex[rec[3]]
		dur, _ := strconv.ParseFloat(rec[4], 64)

		if dur <= 0 || dur > 180 || dist <= 0 {
			continue
		}
		samples = append(samples, Sample{
			Features: [4]float64{dist, hour, dow, borough},
			Target:   dur,
		})
	}
	log.Printf("data loaded: %d valid samples\n", len(samples))
	return samples, nil
}
