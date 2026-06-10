from __future__ import annotations

from datetime import datetime
from pathlib import Path

import polars as pl

PROJECT_ROOT = Path(__file__).resolve().parent.parent
RAW_DATA_DIR = PROJECT_ROOT / "data" / "raw dataset"
LOOKUP_PATH = PROJECT_ROOT / "data" / "taxi_zone_lookup.csv"
PROCESSED_DIR = PROJECT_ROOT / "data" / "processed"

DATE_START = datetime(2026, 1, 1)
DATE_END = datetime(2026, 3, 31, 23, 59, 59)
MAX_TRIP_DISTANCE_MILES = 100.0
MAX_TRIP_DURATION_MINUTES = 24 * 60
MIN_TRIP_DURATION_MINUTES = 1.0

PARQUET_FILES = [
    "yellow_tripdata_2026-01.parquet",
    "yellow_tripdata_2026-02.parquet",
    "yellow_tripdata_2026-03.parquet",
]


def load_zone_lookup() -> tuple[pl.DataFrame, pl.DataFrame]:
    lookup = pl.read_csv(LOOKUP_PATH)
    pu_lookup = lookup.rename(
        {
            "LocationID": "PULocationID",
            "Borough": "pu_borough",
            "Zone": "pu_zone",
            "service_zone": "pu_service_zone",
        }
    )
    do_lookup = lookup.rename(
        {
            "LocationID": "DOLocationID",
            "Borough": "do_borough",
            "Zone": "do_zone",
            "service_zone": "do_service_zone",
        }
    )
    return pu_lookup, do_lookup


def clean_dataframe(df: pl.DataFrame, pu_lookup: pl.DataFrame, do_lookup: pl.DataFrame) -> pl.DataFrame:
    return (
        df.filter(
            (pl.col("tpep_pickup_datetime") >= pl.lit(DATE_START))
            & (pl.col("tpep_pickup_datetime") <= pl.lit(DATE_END))
            & (pl.col("tpep_dropoff_datetime") > pl.col("tpep_pickup_datetime"))
            & (pl.col("trip_distance") > 0)
            & (pl.col("trip_distance") <= MAX_TRIP_DISTANCE_MILES)
            & (pl.col("fare_amount") >= 0)
            & pl.col("PULocationID").is_not_null()
            & pl.col("DOLocationID").is_not_null()
        )
        .with_columns(
            (
                (pl.col("tpep_dropoff_datetime") - pl.col("tpep_pickup_datetime")).dt.total_minutes()
            ).alias("trip_duration_minutes")
        )
        .filter(
            (pl.col("trip_duration_minutes") >= MIN_TRIP_DURATION_MINUTES)
            & (pl.col("trip_duration_minutes") <= MAX_TRIP_DURATION_MINUTES)
        )
        .join(pu_lookup, on="PULocationID", how="left")
        .join(do_lookup, on="DOLocationID", how="left")
        .filter(pl.col("pu_zone").is_not_null() & pl.col("do_zone").is_not_null())
        .with_columns(
            pl.col("tpep_pickup_datetime").dt.hour().alias("pickup_hour"),
            pl.col("tpep_pickup_datetime").dt.weekday().alias("pickup_day_of_week"),
            pl.col("tpep_pickup_datetime").dt.month().alias("pickup_month"),
        )
        .select(
            "VendorID",
            "tpep_pickup_datetime",
            "tpep_dropoff_datetime",
            "PULocationID",
            "DOLocationID",
            "pu_borough",
            "pu_zone",
            "pu_service_zone",
            "do_borough",
            "do_zone",
            "do_service_zone",
            "trip_distance",
            "fare_amount",
            "trip_duration_minutes",
            "pickup_hour",
            "pickup_day_of_week",
            "pickup_month",
        )
    )


def clean_parquet_file(
    file_path: Path,
    pu_lookup: pl.DataFrame,
    do_lookup: pl.DataFrame,
    output_path: Path | None = None,
) -> dict:
    raw_rows = pl.scan_parquet(file_path).select(pl.len()).collect().item()
    cleaned = clean_dataframe(pl.read_parquet(file_path), pu_lookup, do_lookup)

    if output_path is not None:
        output_path.parent.mkdir(parents=True, exist_ok=True)
        cleaned.write_parquet(output_path)

    return {
        "file": file_path.name,
        "raw_rows": raw_rows,
        "clean_rows": cleaned.height,
        "output_path": str(output_path) if output_path else None,
        "dataframe": cleaned if output_path is None else None,
    }


def clean_parquet_file_worker(args: tuple[str, str]) -> dict:
    file_name, temp_dir = args
    pu_lookup, do_lookup = load_zone_lookup()
    file_path = RAW_DATA_DIR / file_name
    output_path = Path(temp_dir) / f"clean_{file_name}"
    result = clean_parquet_file(file_path, pu_lookup, do_lookup, output_path=output_path)
    result.pop("dataframe", None)
    return result
