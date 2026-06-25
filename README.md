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

## stack backend

- **Go** — API, coordinador del cluster, nodos ML
- **MongoDB** — almacena usuarios y historial de entrenamientos
- **Redis** — blacklist de tokens JWT revocados (logout)
- **WebSockets** — transmite métricas de entrenamiento en tiempo real
- **JWT** — autenticación con Bearer tokens (24h TTL)

## funcionamiento del cluster

en una terminal:

```
docker compose up --build
```

luego, en otra terminal:

- registrar un usuario

```
curl -X POST http://localhost:8080/register \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"1234"}'
```

- login (devuelve un token JWT)

```
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"1234"}'
```

- para entrenar el modelo (tarda unos minutos, requiere token)

```
curl -X POST http://localhost:8080/train \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"total_trees": 100, "max_depth": 10, "max_features": 2}'
```

- hacer una predicción (hora 8, martes, 3.2 millas, Manhattan)

```
curl -X POST http://localhost:8080/predict \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"trip_distance": 3.2, "pickup_hour": 8, "pickup_day_of_week": 1, "pu_borough": 3}'
```

> pu_borough: 0 = Bronx, 1 = Brooklyn, 2 = EWR, 3 = Manhattan, 4 = Queens, 5 = Staten Island

- conectar al WebSocket para recibir métricas en tiempo real

```
wscat -c ws://localhost:8080/ws
```

### endpoints

| Método | Ruta | Auth | Descripción |
|--------|------|------|-------------|
| POST | /register | No | Registrar usuario |
| POST | /login | No | Obtener token JWT |
| GET | /health | No | Health check |
| WS | /ws | No | WebSocket métricas en tiempo real |
| POST | /train | JWT | Entrenar modelo distribuido |
| POST | /predict | JWT | Predicción de duración de viaje |
| GET | /metrics | JWT | Últimas métricas de entrenamiento |
| GET | /history | JWT | Historial de entrenamientos (MongoDB) |
| POST | /logout | JWT | Revocar token (Redis blacklist) |

### eventos WebSocket

- `train_start` — al iniciar entrenamiento (total_trees, num_nodes)
- `node_done` — cuando un nodo termina (node_id, trees, dur_ms)
- `train_done` — resultado final (MAE, RMSE, R²)
