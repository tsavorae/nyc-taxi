# nyc-taxi

```
loader/ cargador concurrente
ml-node/ cluster
api/ endpoints
frontend/ spa
data/ parquets
```
## info del cluster

- topología estrella vía TCP
- 4 nodos que se organizan mediante api/coordinator.go
- cada nodo carga los 10M y entrena los árboles
- las métricas se evalúan con una muestra de 100k
- al final solo se usaron 4 features sin fare_amount para prevenir data leakage

## funcionamiento del cluster

en una terminal:

```
docker compose up
```

luego, en otra terminal:

- para entrenar el modelo (tarda unos minutos)

```
curl -X POST http://localhost:8080/train \
  -H "Content-Type: application/json" \
  -d '{"total_trees": 100, "max_depth": 10, "max_features": 2}'
```

- hacer una predicción (hora 8, martes, 3.2 millas, Manhattan)

```
curl -X POST http://localhost:8080/predict \
  -H "Content-Type: application/json" \
  -d '{"trip_distance": 3.2, "pickup_hour": 8, "pickup_day_of_week": 1, "pu_borough": 3}'
```

> pu_borough: 0 = Bronx, 1 = Brooklyn, 2 = EWR, 3 = Manhattan, 4 = Queens, 5 = Staten Island


### algunos requests

- POST /train → entrena y devuelve MAE/RMSE/R² reales
- POST /predict → devuelve predicción en minutos
- GET /metrics → devuelve últimas métricas
- GET /health → health check
