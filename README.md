# Task Manager API

Simple task CRUD API in Go (Gin + Postgres + Redis).

```mermaid
flowchart LR
    client([client]) -->|":8080"| api["gin API (todomgr)"]
    api <-->|"task:{id}, 60s TTL"| redis[("Redis :6379")]
    api --> pg[("Postgres :5432")]
    migrate["migrate (one-shot, golang-migrate)"] --> pg
    prom["Prometheus :9090"] -->|scrapes /metrics| api
```

`GET /tasks/:id` is cached in Redis (60s TTL, fail-open); writes evict
`task:{id}`. `GET /tasks` always hits Postgres.

## Run

```bash
docker compose up --build
```

App: `http://localhost:8080` (health: `/healthz`, docs: `/swagger/index.html`).
Migrations run automatically via the `migrate` service before the app starts.

Env vars (`app`): `DATABASE_URL` (default `postgres://todo:todo@localhost:5432/todo?sslmode=disable`),
`REDIS_URL` (default `redis://localhost:6379/0`), `PORT` (default `8080`).

## API

| Method | Path         | Codes            |
| ------ | ------------ | ---------------- |
| POST   | /tasks       | 201, 400, 500    |
| GET    | /tasks       | 200, 400, 500    |
| GET    | /tasks/:id   | 200, 400, 404    |
| PUT    | /tasks/:id   | 200, 400, 404    |
| DELETE | /tasks/:id   | 204, 400, 404    |

List params: `limit` (1–100, default 20), `offset` (default 0),
`status` (`true`/`false`), `assignee` (exact match).

### curl

```bash
curl -X POST localhost:8080/tasks -H 'Content-Type: application/json' \
  -d '{"title":"Buy milk","assignee":"Sara"}'

curl 'localhost:8080/tasks?status=false&assignee=Sara&limit=10&offset=0'

curl localhost:8080/tasks/1

curl -X PUT localhost:8080/tasks/1 -H 'Content-Type: application/json' \
  -d '{"title":"Buy milk","status":true,"assignee":"Sara"}'

curl -X DELETE localhost:8080/tasks/1
```

### Formats

Task:

```json
{"id":1,"title":"Buy milk","status":false,"assignee":"Sara",
 "created_at":"2026-09-13T22:20:52Z","updated_at":"2026-09-13T22:20:52Z"}
```

Create input: `title` (required, 1–500 chars), `assignee` (optional).
Update input: `title` (required), `status`, `assignee`.
Error: `{"error":"not found"}`.

## Tests

```bash
go test ./...
go vet ./...
```

## Migrations

Versioned SQL in `db/migrations/` (`000002_name.up.sql` / `.down.sql`),
applied by golang-migrate. Add a new pair with the next sequence number;
never edit an applied migration.
