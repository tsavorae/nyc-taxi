import pandas as pd

df = pd.read_parquet("data/processed/yellow_tripdata_2026_clean.parquet")

df[['trip_distance','pickup_hour','pickup_day_of_week','pu_borough','trip_duration_minutes']].to_csv("data/processed/clean_data.csv", index=False)

print(f"CSV generado: {len(df)} filas")
