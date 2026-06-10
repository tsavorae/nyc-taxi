from __future__ import annotations

import argparse
import gc
import multiprocessing as mp
import tempfile
import time
from concurrent.futures import ProcessPoolExecutor, as_completed
from pathlib import Path

import polars as pl

from clean import (
    PARQUET_FILES,
    PROCESSED_DIR,
    RAW_DATA_DIR,
    clean_parquet_file,
    clean_parquet_file_worker,
    load_zone_lookup,
)


def process_sequential() -> tuple[float, list[dict]]:
    pu_lookup, do_lookup = load_zone_lookup()
    stats: list[dict] = []

    start = time.perf_counter()
    for file_name in PARQUET_FILES:
        file_path = RAW_DATA_DIR / file_name
        with tempfile.TemporaryDirectory(prefix="nyc_taxi_seq_") as tmp:
            out = Path(tmp) / f"clean_{file_name}"
            result = clean_parquet_file(file_path, pu_lookup, do_lookup, output_path=out)
            stats.append(
                {"file": result["file"], "raw_rows": result["raw_rows"], "clean_rows": result["clean_rows"]}
            )
    elapsed = time.perf_counter() - start
    return elapsed, stats


def process_concurrent(max_workers: int | None = None) -> tuple[pl.DataFrame, float, list[dict]]:
    workers = max_workers or len(PARQUET_FILES)
    stats: list[dict] = []

    with tempfile.TemporaryDirectory(prefix="nyc_taxi_conc_") as temp_dir:
        start = time.perf_counter()
        worker_args = [(f, temp_dir) for f in PARQUET_FILES]
        with ProcessPoolExecutor(max_workers=workers, mp_context=mp.get_context("spawn")) as executor:
            futures = [executor.submit(clean_parquet_file_worker, args) for args in worker_args]
            for future in as_completed(futures):
                result = future.result()
                stats.append(
                    {
                        "file": result["file"],
                        "raw_rows": result["raw_rows"],
                        "clean_rows": result["clean_rows"],
                        "output_path": result["output_path"],
                    }
                )
        elapsed = time.perf_counter() - start

        paths = sorted(
            [Path(r["output_path"]) for r in stats],
            key=lambda p: p.name,
        )
        return pl.concat([pl.read_parquet(p) for p in paths]), elapsed, stats


def print_stats(stats: list[dict], label: str, elapsed: float) -> None:
    print(f"\n{'=' * 60}")
    print(f"  {label}")
    print(f"{'=' * 60}")
    total_raw = 0
    total_clean = 0
    for row in sorted(stats, key=lambda x: x["file"]):
        removed = row["raw_rows"] - row["clean_rows"]
        pct = (removed / row["raw_rows"]) * 100 if row["raw_rows"] else 0
        print(
            f"  {row['file']}: {row['raw_rows']:,} → {row['clean_rows']:,} "
            f"({removed:,} removidas, {pct:.2f}%)"
        )
        total_raw += row["raw_rows"]
        total_clean += row["clean_rows"]
    total_removed = total_raw - total_clean
    total_pct = (total_removed / total_raw) * 100 if total_raw else 0
    print(f"\n  Total: {total_raw:,} → {total_clean:,} ({total_removed:,} removidas, {total_pct:.2f}%)")
    print(f"  Tiempo: {elapsed:.2f}s")


def run_benchmark(max_workers: int | None = None) -> pl.DataFrame:
    print("Iniciando limpieza secuencial...")
    time_seq, stats_seq = process_sequential()
    print_stats(stats_seq, "LIMPIEZA SECUENCIAL", time_seq)
    gc.collect()

    print("\nIniciando limpieza concurrente...")
    df_conc, time_conc, stats_conc = process_concurrent(max_workers=max_workers)
    print_stats(stats_conc, "LIMPIEZA CONCURRENTE", time_conc)

    speedup = time_seq / time_conc if time_conc > 0 else float("inf")
    print(f"\n{'=' * 60}")
    print("  BENCHMARK DE SPEEDUP")
    print(f"{'=' * 60}")
    print(f"  Tiempo secuencial:  {time_seq:.2f}s")
    print(f"  Tiempo concurrente: {time_conc:.2f}s")
    print(f"  Speedup:            {speedup:.2f}x")
    print(f"{'=' * 60}")

    PROCESSED_DIR.mkdir(parents=True, exist_ok=True)
    output_path = PROCESSED_DIR / "yellow_tripdata_2026_clean.parquet"
    df_conc.write_parquet(output_path)
    print(f"\nDatos limpios guardados en: {output_path}")
    print(f"Filas finales: {df_conc.height:,}")
    print(f"Columnas: {df_conc.columns}")

    sample = df_conc.select("pu_zone", "do_zone", "pickup_hour", "trip_duration_minutes").head(3)
    print(f"\nMuestra (zonas + hora + duración):\n{sample}")

    return df_conc


def main() -> None:
    parser = argparse.ArgumentParser(description="Cargador concurrente NYC Taxi")
    parser.add_argument("--workers", type=int, default=None, help="Número de workers concurrentes")
    args = parser.parse_args()
    run_benchmark(max_workers=args.workers)


if __name__ == "__main__":
    main()
