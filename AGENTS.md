# Repository Guidelines

## Project Structure

- `cmd/app/main.go` — Go entrypoint: loads config, opens SQLite, migrates, seeds, serves `net/http` on `127.0.0.1:8080`. Flags: `-port`, `-migrate-only`, `-seed-only`.
- `internal/config` — XDG-based paths under `~/.config/learning-roadmap/` and `~/.local/share/learning-roadmap/`; honors `XDG_CONFIG_HOME`/`XDG_DATA_HOME`.
- `internal/database` — SQLite open/migrate/seed; `internal/app` is the composition root that wires routes plus health/version endpoints (no domain logic).
- `internal/modules/{settings,roadmap,courses}` — domain modules, each with its own handler registering routes on the shared mux; `internal/shared/httpx` — common HTTP transport, error shape, middleware.
- `migrations/` — embedded SQL migrations (`NNNN_*.sql`); schema changes only via new files, never by editing existing ones. `seeds/backend_go_roadmap.json` — idempotent first-run seed.
- Frontend embed: `web/web_prod.go` (build tag `production`, `go:embed all:dist`) vs `web/web_dev.go` (points to Vite). Plain `go run ./cmd/app` compiles the dev handler; `go build -tags production` embeds `web/dist`.
- `web/src/` splits into `app` (bootstrap/router/global styles), `pages` (route pages), `features` (per-domain UI and API contracts), `shared` (api client, components, types).
- Tests live beside packages: `*_test.go` and `*.test.ts(x)`. `dist/`, `web/dist/`, `web/node_modules/` are generated and ignored.

## Commands

```sh
make dev                 # backend (127.0.0.1:8080) + Vite (127.0.0.1:5173, proxies /api)
make dev-backend         # Go only; make dev-frontend — Vite only
make build               # tsc check + vite build, then go build -tags production -> dist/learning-roadmap
make test                # go test ./... + vitest run
make test-backend        # go test ./...
make test-frontend       # npm --prefix web test -- --run
make lint                # gofmt -l check, go vet, tsc --noEmit
make migrate / make seed # apply and exit
make clean
```

`make` targets run `npm ci` automatically when `web/node_modules` is missing. Focused verification: `go test ./internal/modules/roadmap -run TestName`, `npm --prefix web run typecheck`. After embedded-asset changes run `go test -tags production ./web`.

## Gotchas

- `github.com/mattn/go-sqlite3` requires CGO and a C compiler; Go 1.22+.
- The `production`/`!production` build tags swap which frontend handler is compiled, so embedded-asset changes must be verified with `-tags production`.
- `internal/database` runs migrations and the idempotent seed on every startup, so `make seed` and first run behave the same.

## Coding Style

- Run `gofmt` on Go changes; idiomatic Go, small packages, `context.Context` for database/request work.
- TypeScript strict: PascalCase React components, camelCase functions, kebab-case CSS classes.
- API uses the `/api/v1` prefix and `snake_case` JSON; errors are `{"error":{"code":"...","message":"...","details":{}}}`.
- Prefer existing helpers (e.g. `internal/shared/httpx`) and platform features over new dependencies.

## Testing

Go standard `testing` for API, migration, seed, and validation tests; Vitest for frontend behavior/rendering. Name tests after the behavior they protect, e.g. `TestMigrateAndSeedAreIdempotent` or `RoadmapTree.test.tsx`. Run `make test`; run `go test -tags production ./web` after embedded-asset changes.

## Commits

Git history and a GitHub remote (`origin`) exist. Use imperative messages such as `Fix roadmap selection`, keep commits narrow, check `git status` first, and leave unrelated work untouched. Commit every completed change before handoff. Never commit databases, `node_modules`, credentials, or local configuration; tokens from `~/.hermes` must never be copied into the repository.

## Security & Configuration

Keep the server bound to `127.0.0.1`; never change the default to `0.0.0.0`. Local data belongs under the XDG config/data directories, not in the repository. Treat API input and local paths as untrusted, keep request size limits, and do not log secrets or full note contents.

## Architecture Map

Every architectural change (new module, endpoint, data flow, build path) must also be reflected in `web/public/architecture.html` in the same change; keep its nested sections and file responsibilities accurate.
