from __future__ import annotations

import argparse
import json
from pathlib import Path

import joblib
import polars as pl
from sklearn.compose import ColumnTransformer
from sklearn.ensemble import HistGradientBoostingRegressor
from sklearn.metrics import mean_absolute_error, mean_squared_error, r2_score
from sklearn.model_selection import train_test_split
from sklearn.pipeline import Pipeline
from sklearn.preprocessing import OrdinalEncoder

PROJECT_ROOT = Path(__file__).resolve().parent.parent
PROCESSED_PATH = PROJECT_ROOT / "data" / "processed" / "yellow_tripdata_2026_clean.parquet"
MODEL_DIR = Path(__file__).resolve().parent / "models"

FEATURE_COLUMNS = ["PULocationID", "DOLocationID", "pickup_hour", "pickup_day_of_week"]
TARGET_COLUMN = "trip_duration_minutes"


def load_training_data(sample_fraction: float | None = None) -> pl.DataFrame:
    df = pl.read_parquet(PROCESSED_PATH)
    if sample_fraction is not None and 0 < sample_fraction < 1:
        df = df.sample(fraction=sample_fraction, seed=42)
    return df


def train_model(sample_fraction: float | None = 0.05, max_iter: int = 200) -> dict:
    df = load_training_data(sample_fraction)
    pdf = df.select(FEATURE_COLUMNS + [TARGET_COLUMN]).to_pandas()

    x = pdf[FEATURE_COLUMNS]
    y = pdf[TARGET_COLUMN]

    x_train, x_test, y_train, y_test = train_test_split(
        x, y, test_size=0.2, random_state=42
    )

    pipeline = Pipeline(
        steps=[
            (
                "preprocessor",
                ColumnTransformer(
                    transformers=[
                        (
                            "zones",
                            OrdinalEncoder(handle_unknown="use_encoded_value", unknown_value=-1),
                            ["PULocationID", "DOLocationID"],
                        ),
                        ("time", "passthrough", ["pickup_hour", "pickup_day_of_week"]),
                    ]
                ),
            ),
            (
                "model",
                HistGradientBoostingRegressor(
                    max_iter=max_iter,
                    random_state=42,
                    learning_rate=0.1,
                ),
            ),
        ]
    )

    pipeline.fit(x_train, y_train)
    predictions = pipeline.predict(x_test)

    metrics = {
        "mae_minutes": round(mean_absolute_error(y_test, predictions), 4),
        "rmse_minutes": round(mean_squared_error(y_test, predictions) ** 0.5, 4),
        "r2": round(r2_score(y_test, predictions), 4),
        "train_rows": len(x_train),
        "test_rows": len(x_test),
        "features": FEATURE_COLUMNS,
        "target": TARGET_COLUMN,
        "description": "Predice duración (min) de viaje zona X → zona Y a cierta hora del día",
    }

    MODEL_DIR.mkdir(parents=True, exist_ok=True)
    model_path = MODEL_DIR / "trip_duration_model.joblib"
    joblib.dump(pipeline, model_path)

    metadata_path = MODEL_DIR / "metrics.json"
    metadata_path.write_text(json.dumps(metrics, indent=2))

    print(f"Modelo guardado en: {model_path}")
    print(f"Métricas: MAE={metrics['mae_minutes']} min, RMSE={metrics['rmse_minutes']} min, R²={metrics['r2']}")
    print(f"Train: {metrics['train_rows']:,} filas | Test: {metrics['test_rows']:,} filas")

    example = predict_trip(
        pipeline,
        pu_location_id=237,
        do_location_id=138,
        pickup_hour=18,
        pickup_day_of_week=2,
    )
    print(f"\nEjemplo: zona 237 → 138, 18:00, martes → {example:.1f} min estimados")

    return metrics


def predict_trip(
    pipeline,
    pu_location_id: int,
    do_location_id: int,
    pickup_hour: int,
    pickup_day_of_week: int,
) -> float:
    import pandas as pd

    row = pd.DataFrame(
        [
            {
                "PULocationID": pu_location_id,
                "DOLocationID": do_location_id,
                "pickup_hour": pickup_hour,
                "pickup_day_of_week": pickup_day_of_week,
            }
        ]
    )
    return float(pipeline.predict(row)[0])


def main() -> None:
    parser = argparse.ArgumentParser(description="Entrenar modelo de duración de viaje")
    parser.add_argument(
        "--sample",
        type=float,
        default=0.05,
        help="Fracción de datos a usar (0-1). Usar 1.0 para dataset completo.",
    )
    parser.add_argument("--iter", type=int, default=200, help="Iteraciones del gradient boosting")
    args = parser.parse_args()
    train_model(sample_fraction=args.sample, max_iter=args.iter)


if __name__ == "__main__":
    main()
