package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/apache/arrow-go/v18/parquet"
	"github.com/apache/arrow-go/v18/parquet/file"
	"github.com/apache/arrow-go/v18/parquet/pqarrow"
)

type fileStats struct {
	File       string
	RawRows    int
	CleanRows  int
	OutputPath string
}

func polarsWeekday(t time.Time) int32 {
	wd := t.Weekday()
	if wd == time.Sunday {
		return 7
	}
	return int32(wd)
}

func cleanRecord(
	vendorID int32,
	pickupMicros, dropoffMicros int64,
	puID, doID int32,
	tripDistance, fareAmount float64,
	lookup map[int32]zoneInfo,
) (cleanRow, bool) {
	pickup := time.UnixMicro(pickupMicros)
	dropoff := time.UnixMicro(dropoffMicros)

	if pickup.Before(dateStart) || pickup.After(dateEnd) {
		return cleanRow{}, false
	}
	if !dropoff.After(pickup) {
		return cleanRow{}, false
	}
	if tripDistance <= 0 || tripDistance > maxTripDistanceMiles {
		return cleanRow{}, false
	}
	if fareAmount < 0 {
		return cleanRow{}, false
	}

	durationMin := dropoff.Sub(pickup).Minutes()
	if durationMin < minTripDurationMinutes || durationMin > maxTripDurationMinutes {
		return cleanRow{}, false
	}

	pu, okPU := lookup[puID]
	do, okDO := lookup[doID]
	if !okPU || !okDO {
		return cleanRow{}, false
	}

	return cleanRow{
		VendorID:            vendorID,
		PickupMicros:        pickupMicros,
		DropoffMicros:       dropoffMicros,
		PULocationID:        puID,
		DOLocationID:        doID,
		PuBorough:           pu.Borough,
		PuZone:              pu.Zone,
		PuServiceZone:       pu.ServiceZone,
		DoBorough:           do.Borough,
		DoZone:              do.Zone,
		DoServiceZone:       do.ServiceZone,
		TripDistance:        tripDistance,
		FareAmount:          fareAmount,
		TripDurationMinutes: int64(durationMin),
		PickupHour:          int32(pickup.Hour()),
		PickupDayOfWeek:     polarsWeekday(pickup),
		PickupMonth:         int32(pickup.Month()),
	}, true
}

type cleanRow struct {
	VendorID            int32
	PickupMicros        int64
	DropoffMicros       int64
	PULocationID        int32
	DOLocationID        int32
	PuBorough           string
	PuZone              string
	PuServiceZone       string
	DoBorough           string
	DoZone              string
	DoServiceZone       string
	TripDistance        float64
	FareAmount          float64
	TripDurationMinutes int64
	PickupHour          int32
	PickupDayOfWeek     int32
	PickupMonth         int32
}

func cleanSchema() *arrow.Schema {
	return arrow.NewSchema([]arrow.Field{
		{Name: "VendorID", Type: arrow.PrimitiveTypes.Int32, Nullable: true},
		{Name: "tpep_pickup_datetime", Type: &arrow.TimestampType{Unit: arrow.Microsecond, TimeZone: "UTC"}, Nullable: true},
		{Name: "tpep_dropoff_datetime", Type: &arrow.TimestampType{Unit: arrow.Microsecond, TimeZone: "UTC"}, Nullable: true},
		{Name: "PULocationID", Type: arrow.PrimitiveTypes.Int32, Nullable: true},
		{Name: "DOLocationID", Type: arrow.PrimitiveTypes.Int32, Nullable: true},
		{Name: "pu_borough", Type: arrow.BinaryTypes.String, Nullable: true},
		{Name: "pu_zone", Type: arrow.BinaryTypes.String, Nullable: true},
		{Name: "pu_service_zone", Type: arrow.BinaryTypes.String, Nullable: true},
		{Name: "do_borough", Type: arrow.BinaryTypes.String, Nullable: true},
		{Name: "do_zone", Type: arrow.BinaryTypes.String, Nullable: true},
		{Name: "do_service_zone", Type: arrow.BinaryTypes.String, Nullable: true},
		{Name: "trip_distance", Type: arrow.PrimitiveTypes.Float64, Nullable: true},
		{Name: "fare_amount", Type: arrow.PrimitiveTypes.Float64, Nullable: true},
		{Name: "trip_duration_minutes", Type: arrow.PrimitiveTypes.Int64, Nullable: true},
		{Name: "pickup_hour", Type: arrow.PrimitiveTypes.Int32, Nullable: true},
		{Name: "pickup_day_of_week", Type: arrow.PrimitiveTypes.Int32, Nullable: true},
		{Name: "pickup_month", Type: arrow.PrimitiveTypes.Int32, Nullable: true},
	}, nil)
}

func buildCleanRecord(pool memory.Allocator, rows []cleanRow) arrow.Record {
	b := array.NewRecordBuilder(pool, cleanSchema())
	defer b.Release()

	for _, r := range rows {
		b.Field(0).(*array.Int32Builder).Append(r.VendorID)
		b.Field(1).(*array.TimestampBuilder).Append(arrow.Timestamp(r.PickupMicros))
		b.Field(2).(*array.TimestampBuilder).Append(arrow.Timestamp(r.DropoffMicros))
		b.Field(3).(*array.Int32Builder).Append(r.PULocationID)
		b.Field(4).(*array.Int32Builder).Append(r.DOLocationID)
		b.Field(5).(*array.StringBuilder).Append(r.PuBorough)
		b.Field(6).(*array.StringBuilder).Append(r.PuZone)
		b.Field(7).(*array.StringBuilder).Append(r.PuServiceZone)
		b.Field(8).(*array.StringBuilder).Append(r.DoBorough)
		b.Field(9).(*array.StringBuilder).Append(r.DoZone)
		b.Field(10).(*array.StringBuilder).Append(r.DoServiceZone)
		b.Field(11).(*array.Float64Builder).Append(r.TripDistance)
		b.Field(12).(*array.Float64Builder).Append(r.FareAmount)
		b.Field(13).(*array.Int64Builder).Append(r.TripDurationMinutes)
		b.Field(14).(*array.Int32Builder).Append(r.PickupHour)
		b.Field(15).(*array.Int32Builder).Append(r.PickupDayOfWeek)
		b.Field(16).(*array.Int32Builder).Append(r.PickupMonth)
	}

	return b.NewRecord()
}

func colIndex(cols []arrow.Array, name string, schema *arrow.Schema) int {
	for i, f := range schema.Fields() {
		if f.Name == name {
			return i
		}
	}
	panic("column not found: " + name)
}

func cleanParquetFile(inputPath, outputPath string, lookup map[int32]zoneInfo) (fileStats, error) {
	pr, err := file.OpenParquetFile(inputPath, false)
	if err != nil {
		return fileStats{}, err
	}
	defer pr.Close()

	pool := memory.NewGoAllocator()
	reader, err := pqarrow.NewFileReader(pr, pqarrow.ArrowReadProperties{}, pool)
	if err != nil {
		return fileStats{}, err
	}

	stats := fileStats{File: filepath.Base(inputPath)}

	var writer *pqarrow.FileWriter
	if outputPath != "" {
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
			return fileStats{}, err
		}
		out, err := os.Create(outputPath)
		if err != nil {
			return fileStats{}, err
		}
		defer out.Close()

		pwProps := parquet.NewWriterProperties()
		arrowProps := pqarrow.NewArrowWriterProperties()
		writer, err = pqarrow.NewFileWriter(cleanSchema(), out, pwProps, arrowProps)
		if err != nil {
			return fileStats{}, err
		}
		defer writer.Close()
		stats.OutputPath = outputPath
	}

	tbl, err := reader.ReadTable(context.Background())
	if err != nil {
		return fileStats{}, err
	}
	defer tbl.Release()

	tr := array.NewTableReader(tbl, 65536)
	defer tr.Release()

	for tr.Next() {
		rec := tr.Record()
		schema := rec.Schema()
		cols := rec.Columns()

		vendor := cols[colIndex(cols, "VendorID", schema)].(*array.Int32)
		pickup := cols[colIndex(cols, "tpep_pickup_datetime", schema)].(*array.Timestamp)
		dropoff := cols[colIndex(cols, "tpep_dropoff_datetime", schema)].(*array.Timestamp)
		puID := cols[colIndex(cols, "PULocationID", schema)].(*array.Int32)
		doID := cols[colIndex(cols, "DOLocationID", schema)].(*array.Int32)
		distance := cols[colIndex(cols, "trip_distance", schema)].(*array.Float64)
		fare := cols[colIndex(cols, "fare_amount", schema)].(*array.Float64)

		cleaned := make([]cleanRow, 0, int(rec.NumRows()))
		for i := 0; i < int(rec.NumRows()); i++ {
			stats.RawRows++
			if vendor.IsNull(i) || pickup.IsNull(i) || dropoff.IsNull(i) ||
				puID.IsNull(i) || doID.IsNull(i) || distance.IsNull(i) || fare.IsNull(i) {
				continue
			}
			row, ok := cleanRecord(
				vendor.Value(i),
				int64(pickup.Value(i)),
				int64(dropoff.Value(i)),
				puID.Value(i),
				doID.Value(i),
				distance.Value(i),
				fare.Value(i),
				lookup,
			)
			if ok {
				cleaned = append(cleaned, row)
				stats.CleanRows++
			}
		}

		if writer != nil && len(cleaned) > 0 {
			outRec := buildCleanRecord(pool, cleaned)
			if err := writer.Write(outRec); err != nil {
				outRec.Release()
				rec.Release()
				return fileStats{}, err
			}
			outRec.Release()
		}
		rec.Release()
	}
	if err := tr.Err(); err != nil {
		return fileStats{}, err
	}

	return stats, nil
}

func mergeParquetFiles(paths []string, outputPath string) (int, error) {
	if len(paths) == 0 {
		return 0, fmt.Errorf("no input files")
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return 0, err
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return 0, err
	}
	defer out.Close()

	pool := memory.NewGoAllocator()

	first, err := file.OpenParquetFile(paths[0], false)
	if err != nil {
		return 0, err
	}
	firstReader, err := pqarrow.NewFileReader(first, pqarrow.ArrowReadProperties{}, pool)
	if err != nil {
		first.Close()
		return 0, err
	}
	schema, err := firstReader.Schema()
	first.Close()
	if err != nil {
		return 0, err
	}

	pwProps := parquet.NewWriterProperties()
	arrowProps := pqarrow.NewArrowWriterProperties()
	writer, err := pqarrow.NewFileWriter(schema, out, pwProps, arrowProps)
	if err != nil {
		return 0, err
	}
	defer writer.Close()

	total := 0
	for _, path := range paths {
		pr, err := file.OpenParquetFile(path, false)
		if err != nil {
			return 0, err
		}

		reader, err := pqarrow.NewFileReader(pr, pqarrow.ArrowReadProperties{}, pool)
		if err != nil {
			pr.Close()
			return 0, err
		}

		tbl, err := reader.ReadTable(context.Background())
		if err != nil {
			pr.Close()
			return 0, err
		}

		tr := array.NewTableReader(tbl, 65536)
		for tr.Next() {
			rec := tr.Record()
			total += int(rec.NumRows())
			if err := writer.Write(rec); err != nil {
				rec.Release()
				tr.Release()
				tbl.Release()
				pr.Close()
				return 0, err
			}
			rec.Release()
		}
		if err := tr.Err(); err != nil {
			tr.Release()
			tbl.Release()
			pr.Close()
			return 0, err
		}
		tr.Release()
		tbl.Release()
		pr.Close()
	}

	return total, nil
}

func printSample(outputPath string) error {
	pr, err := file.OpenParquetFile(outputPath, false)
	if err != nil {
		return err
	}
	defer pr.Close()

	pool := memory.NewGoAllocator()
	reader, err := pqarrow.NewFileReader(pr, pqarrow.ArrowReadProperties{}, pool)
	if err != nil {
		return err
	}

	tbl, err := reader.ReadTable(context.Background())
	if err != nil {
		return err
	}
	defer tbl.Release()

	tr := array.NewTableReader(tbl, 65536)
	defer tr.Release()

	if !tr.Next() {
		return io.EOF
	}
	rec := tr.Record()
	defer rec.Release()

	puZone := rec.Column(colIndex(rec.Columns(), "pu_zone", rec.Schema())).(*array.String)
	doZone := rec.Column(colIndex(rec.Columns(), "do_zone", rec.Schema())).(*array.String)
	hour := rec.Column(colIndex(rec.Columns(), "pickup_hour", rec.Schema())).(*array.Int32)
	duration := rec.Column(colIndex(rec.Columns(), "trip_duration_minutes", rec.Schema())).(*array.Int64)

	fmt.Println("\nMuestra (zonas + hora + duración):")
	limit := min(3, int(rec.NumRows()))
	for i := 0; i < limit; i++ {
		fmt.Printf("  %s → %s | hora %d | %d min\n",
			puZone.Value(i), doZone.Value(i), hour.Value(i), duration.Value(i))
	}
	return nil
}
