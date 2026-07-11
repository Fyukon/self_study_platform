# Repository Guidelines

## Project Structure

- `cmd/app/main.go` is the Go entry point and localhost HTTP server.
- `internal/config`, `internal/database`, and `internal/app` contain configuration, SQLite/migrations, and API/domain logic.
- `migrations/` stores embedded SQL migrations; `seeds/backend_go_roadmap.json` is the idempotent first-run roadmap seed.
- `web/src/` contains the React/TypeScript UI, `web/public/` contains static assets such as `architecture.html`, and `web/web_*` embeds or serves frontend assets.
- Go tests live beside their packages; frontend tests use `*.test.ts` or `*.test.tsx`. `dist/` and `web/node_modules/` are generated and ignored.

## Build, Test, and Development Commands

```sh
make dev                 # Go backend + Vite development server
make build               # frontend build and embedded production binary
make test                # Go tests plus Vitest tests
make lint                # gofmt check, go vet, and TypeScript check
make migrate             # apply SQLite migrations and exit
make seed                # apply migrations/seed and exit
```

The production binary is `dist/learning-roadmap` and listens on `127.0.0.1:8080` by default. Use `-port` only when needed.

## Coding Style & Naming

Run `gofmt` on Go changes. Use idiomatic Go names and small packages; keep SQL in migrations and use `context.Context` for database/request work. TypeScript is strict: use PascalCase for React components, camelCase for functions, and kebab-case CSS classes. Preserve the API’s `snake_case` JSON fields. Prefer existing helpers and platform features over new dependencies.

## Testing Guidelines

Use Go’s standard `testing` package for API, migration, seed, and validation tests. Use Vitest for frontend behavior and rendering tests. Name tests after the behavior they protect, for example `TestMigrateAndSeedAreIdempotent` or `RoadmapTree.test.tsx`. Run `make test`; run `go test -tags production ./web` after embedded-asset changes.

## Commit & Pull Request Guidelines

This workspace has no usable Git history; use imperative messages such as `Fix roadmap selection`. Every completed change must be committed before handoff. Check `git status` first and leave unrelated work untouched. Keep commits narrow. Pull requests should explain the change, list tests, note migration/seed changes, and include UI screenshots when relevant. Never commit databases, `node_modules`, credentials, or local configuration; tokens from `~/.hermes` must never be copied into the repository.

## Security & Configuration

Keep the server bound to `127.0.0.1`; never change the default to `0.0.0.0`. Local data belongs under the XDG config/data directories, not in the repository. Treat API input and local paths as untrusted, keep request size limits, and do not log secrets or full note contents.

## Architecture Map Maintenance

Every new module, endpoint, data flow, build path, or other architectural change must also be added to `web/public/architecture.html` in the same change. Keep its nested sections and file responsibilities accurate.
