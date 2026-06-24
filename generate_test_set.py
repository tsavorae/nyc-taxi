import pandas as pd

df = pd.read_parquet("data/processed/yellow_tripdata_2026_clean.parquet")
test = df.sample(frac=0.2, random_state=42)
test[['trip_distance','pickup_hour','pickup_day_of_week','pu_borough','trip_duration_minutes']] \
    .to_csv("data/processed/test_data.csv", index=False)
print(f"Test set: {len(test)} filas")
