package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
)

type zoneInfo struct {
	Borough     string
	Zone        string
	ServiceZone string
}

func loadZoneLookup(path string) (map[int32]zoneInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, err
	}

	lookup := make(map[int32]zoneInfo, len(records))
	for i, row := range records {
		if i == 0 {
			continue
		}
		if len(row) < 4 {
			continue
		}
		id, err := strconv.Atoi(row[0])
		if err != nil {
			return nil, fmt.Errorf("invalid LocationID %q: %w", row[0], err)
		}
		lookup[int32(id)] = zoneInfo{
			Borough:     row[1],
			Zone:        row[2],
			ServiceZone: row[3],
		}
	}
	return lookup, nil
}
