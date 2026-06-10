from __future__ import annotations

import argparse
import json
from pathlib import Path

import joblib
import polars as pl

MODEL_PATH = Path(__file__).resolve().parent / "models" / "trip_duration_model.joblib"
LOOKUP_PATH = Path(__file__).resolve().parent.parent / "data" / "taxi_zone_lookup.csv"


def load_model():
    return joblib.load(MODEL_PATH)


def zone_name_to_id(zone_name: str) -> int:
    lookup = pl.read_csv(LOOKUP_PATH)
    match = lookup.filter(pl.col("Zone") == zone_name)
    if match.is_empty():
        raise ValueError(f"Zona no encontrada: {zone_name}")
    return int(match["LocationID"][0])


def predict(
    pu_zone: str,
    do_zone: str,
    pickup_hour: int,
    pickup_day_of_week: int = 1,
) -> float:
    pipeline = load_model()
    pu_id = zone_name_to_id(pu_zone)
    do_id = zone_name_to_id(do_zone)
    import pandas as pd

    row = pd.DataFrame(
        [
            {
                "PULocationID": pu_id,
                "DOLocationID": do_id,
                "pickup_hour": pickup_hour,
                "pickup_day_of_week": pickup_day_of_week,
            }
        ]
    )
    return float(pipeline.predict(row)[0])


def main() -> None:
    parser = argparse.ArgumentParser(description="Predecir duración de viaje")
    parser.add_argument("--from-zone", required=True, help="Zona de origen (nombre exacto del CSV)")
    parser.add_argument("--to-zone", required=True, help="Zona de destino")
    parser.add_argument("--hour", type=int, required=True, help="Hora del día (0-23)")
    parser.add_argument("--weekday", type=int, default=1, help="Día de la semana (1=lunes, 7=domingo)")
    args = parser.parse_args()

    duration = predict(args.from_zone, args.to_zone, args.hour, args.weekday)
    print(json.dumps({"duration_minutes": round(duration, 2), "from": args.from_zone, "to": args.to_zone}))


if __name__ == "__main__":
    main()
