# nyc-taxi

```
loader/ cargador concurrente
ml-node/ cluster
api/ endpoints
frontend/ spa
data/ parquets
```

## corroborar funcionamiento del cluster

```
docker compose up
```
el resultado debe ser:

```
Attaching to api-1, ml-node-1-1, ml-node-2-1, ml-node-3-1, ml-node-4-1
ml-node-3-1  | 2026/06/23 05:19:22 [node-3] escuchando en :9051
ml-node-1-1  | 2026/06/23 05:19:22 [node-1] escuchando en :9051
ml-node-4-1  | 2026/06/23 05:19:22 [node-4] escuchando en :9051
ml-node-2-1  | 2026/06/23 05:19:22 [node-2] escuchando en :9051
api-1        | 2026/06/23 05:19:22 api escuchando en :8080
```
