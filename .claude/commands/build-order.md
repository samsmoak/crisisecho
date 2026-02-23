# /build-order — CrisisEcho Build Order

## Infrastructure First (Prompt 1)

Build infrastructure in this exact order before any domain code:

```
1. go.mod                                                     ← module root
2. internal/database/abstractrepository/mongodb/mongodb.go    ← generic CRUD base
3. internal/database/abstractrepository/location/location.go  ← location DB base
4. internal/database/abstractrepository/vectordb/vectordb.go  ← vector search base
5. internal/database/database.go                              ← three client connections + collection helpers
6. internal/server/server.go                                  ← FiberServer struct + New()
7. internal/server/routes.go                                  ← RegisterRoutes() stub
8. internal/server/ws_routes.go                               ← RegisterWSRoutes() stub
9. cmd/api/main.go                                            ← entry point
10. scripts/create_vector_indexes.js                          ← Atlas one-time setup
```

---

## Domain Build Order (Prompt 2+)

Build domains in this sequence — each domain may depend on the one before it:

```
1. post
2. cluster
3. crisis
4. alert
5. ingest
6. preprocess
7. rag
8. notify
9. query
```

---

## Layer Order Within Each Domain

For every domain, always build layers in this order:

```
1. model       → internal/apps/<domain>/model/<domain>.go
2. repository  → internal/apps/<domain>/repository/<domain>_repository.go
3. service     → internal/apps/<domain>/service/<domain>_service.go
4. controller  → internal/apps/<domain>/controller/<domain>_controller.go
```

Then wire the domain in `internal/server/routes.go`.

---

## File Naming Rules

| Layer | Pattern | Example |
|-------|---------|---------|
| Model | `<domain>.go` | `post.go` |
| Repository | `<domain>_repository.go` | `post_repository.go` |
| Service | `<domain>_service.go` | `post_service.go` |
| Controller | `<domain>_controller.go` | `post_controller.go` |
| Python script | `snake_case.py` | `embed_text.py` |

---

## Wiring Order (routes.go)

When adding a new domain to `RegisterRoutes()`:

```
1. Instantiate repository  →  repo := domainRepo.NewXxxRepository(db)
2. Instantiate service     →  svc  := domainService.NewXxxService(repo, …deps)
3. Instantiate controller  →  ctrl := domainCtrl.NewXxxController(svc)
4. Call RegisterRoutes     →  ctrl.RegisterRoutes(apiGroup)
```

---

## API Group Prefix

All REST routes mount under `/api`:

```go
api := srv.App.Group("/api")
```

Route groups use plural nouns:
- `/api/posts`
- `/api/clusters`
- `/api/crises`
- `/api/alerts`
- `/api/subscriptions`
- `/api/query`

WebSocket routes mount directly:
- `/ws/alerts`

---

## Middleware Application Order

```
1. CORS                    ← applied globally on App
2. JWT auth middleware      ← applied per route group (not on public routes)
3. Rate limiter            ← applied per route group (future)
```

---

## Database Collection Assignment

| Domain | Database Client | Collection |
|--------|----------------|------------|
| post | MainClient | `posts` |
| cluster | MainClient | `clusters` |
| crisis | MainClient | `crises` |
| alert | MainClient | `alerts` |
| ingest | MainClient | per-source raw |
| preprocess | MainClient + LocationClient | `posts` + location DB |
| rag | VectorClient | `text_embeddings`, `image_embeddings` |
| notify | MainClient | `subscriptions` |
| query | VectorClient + MainClient | embeddings + clusters |
