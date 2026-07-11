APP := dist/learning-roadmap
VERSION ?= 0.1.0

.PHONY: dev dev-backend dev-frontend frontend-deps build test test-backend test-frontend lint migrate seed clean

dev:
	$(MAKE) -j2 dev-backend dev-frontend

dev-backend:
	go run ./cmd/app

dev-frontend: frontend-deps
	npm --prefix web run dev

frontend-deps:
	@test -d web/node_modules || npm --prefix web ci

build: frontend-deps
	npm --prefix web run build
	mkdir -p dist
	go build -buildvcs=false -tags production -ldflags "-X main.version=$(VERSION)" -o $(APP) ./cmd/app

test: test-backend test-frontend

test-backend:
	go test ./...

test-frontend: frontend-deps
	npm --prefix web test -- --run

lint: frontend-deps
	@test -z "$$(gofmt -l $$(find cmd internal migrations seeds web -name '*.go' -type f))"
	go vet ./...
	npm --prefix web run typecheck

migrate:
	go run ./cmd/app -migrate-only

seed:
	go run ./cmd/app -seed-only

clean:
	go clean
	rm -rf dist web/dist
