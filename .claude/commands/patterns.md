# /patterns — CrisisEcho Coding Patterns

## Module Name

```
crisisecho
```

---

## Folder Structure

Follow this structure for every domain:

```
internal/apps/<domain>/
  model/        → <domain>.go
  repository/   → <domain>_repository.go
  service/      → <domain>_service.go
  controller/   → <domain>_controller.go
```

Infrastructure folders:

```
internal/database/                                              → database.go
internal/database/abstractrepository/mongodb/mongodb.go        → generic base CRUD repo
internal/database/abstractrepository/location/location.go      → location DB base repo
internal/database/abstractrepository/vectordb/vectordb.go      → MongoDB vector search base
internal/server/                                               → server.go, routes.go, ws_routes.go
internal/middleware/                                           → auth_middleware.go
internal/ai/                                                   → Python scripts (snake_case)
cmd/api/main.go                                                → entry point
scripts/                                                       → one-time setup scripts
.claude/commands/                                              → slash command docs
```

---

## File Naming

Pattern: `<domain>_<layer>.go`

Examples:
- `post_controller.go`
- `post_service.go`
- `post_repository.go`
- `post.go` (model)

Python scripts in `internal/ai/`: `snake_case.py`

---

## Abstract Base Repo Pattern

```go
// Package mongoRepo
type MongoRepository[T any] struct {
    Collection *mongo.Collection
}

func NewMongoRepository[T any](db *mongo.Database, collectionName string) *MongoRepository[T] {
    return &MongoRepository[T]{
        Collection: db.Collection(collectionName),
    }
}
```

Provides: `GetByID`, `GetAll`, `Create` (reflect sets `_id` from `InsertedID`),
`Update` (full document replace), `FindOneAndUpdate`, `Delete`, `FindOne`, `FindMany`.

---

## Concrete Repo Pattern

```go
type PostRepository struct {
    *mongoRepo.MongoRepository[model.Post]
}

func NewPostRepository(db *mongo.Database) *PostRepository {
    return &PostRepository{
        MongoRepository: mongoRepo.NewMongoRepository[model.Post](db, "posts"),
    }
}
```

Add domain-specific methods on the concrete struct only (never on the generic base).

---

## Layer Rules

| Layer | Rule |
|-------|------|
| **Model** | Pure struct — bson/json/validate tags only, minimal logic |
| **Repository** | Data access only — embeds `MongoRepository[T]`, adds domain queries |
| **Service** | Public interface + unexported concrete struct — injects repos + deps |
| **Controller** | Holds service interface — `RegisterRoutes(router fiber.Router)`, returns JSON |

- Controllers **never** talk to repos directly.
- Services **never** import controller packages.

---

## Dependency Injection

Manual explicit wiring in `routes.go` **only**. No framework or container.

```go
// routes.go
func RegisterRoutes(srv *FiberServer) {
    postRepo    := postRepo.NewPostRepository(srv.MainClient.Database(os.Getenv("MONGO_DB_DATABASE")))
    postService := postService.NewPostService(postRepo)
    postCtrl    := postController.NewPostController(postService)

    api := srv.App.Group("/api")
    postCtrl.RegisterRoutes(api)
}
```

---

## Error Handling

```go
// Return (result, error) always
func (r *PostRepository) GetByID(ctx context.Context, id string) (*model.Post, error) {
    // ...
    if errors.Is(err, mongo.ErrNoDocuments) {
        return nil, errors.New("post not found")
    }
    return nil, fmt.Errorf("PostRepository.GetByID: %w", err)
}
```

Rules:
- Wrap: `fmt.Errorf("context: %w", err)`
- Sentinel errors: `errors.New("post not found")`
- Check `mongo.ErrNoDocuments` and `mongo.IsDuplicateKeyError(err)`
- Non-critical failures (e.g. embedding generation): `fmt.Printf("warn: ...")`, do **not** fail main operation
- User-facing: return graceful fallback response on all error paths

---

## Server Struct

```go
type FiberServer struct {
    App            *fiber.App
    MainClient     *mongo.Client   // main DB
    LocationClient *mongo.Client   // location DB
    VectorClient   *mongo.Client   // vector DB
    RedisClient    *redis.Client
}
```

- `server.New()` sets `ServerHeader: "crisisecho"`
- `RegisterRoutes()` is the **single** wiring function, called once from `main.go`
- Route groups: plural nouns — `/posts` `/clusters` `/alerts` `/query`
- WebSocket routes in `ws_routes.go` using Redis Pub/Sub
- CORS globally first, JWT middleware per route group
- Panic recovery via `defer/recover` in `main.go`

---

## Route Registration

```go
// In each controller:
func (c *PostController) RegisterRoutes(router fiber.Router) {
    posts := router.Group("/posts")
    posts.Get("/", c.List)
    posts.Get("/:id", c.GetByID)
    posts.Post("/", c.Create)
}
```

---

## Docker (multi-stage)

```dockerfile
# Stage 1: build
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /app/server ./cmd/api

# Stage 2: runtime
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y python3 pip python3-venv \
    && rm -rf /var/lib/apt/lists/*
# Pre-download ML models at build time (not runtime)
RUN python3 -m venv /venv && /venv/bin/pip install -r /app/requirements.txt
COPY --from=builder /app/server /app/server
RUN useradd -m appuser
USER appuser
CMD ["/app/server"]
```

Runtime: Python3 + pip + venv + ML models pre-downloaded at build time.
Run as non-root `appuser`.
