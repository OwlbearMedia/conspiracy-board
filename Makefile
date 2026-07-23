.PHONY: up down logs migrate test lint typecheck

up: ## start the local stack (app at http://localhost:8080)
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs -f api

migrate: ## run migrations against the local db
	docker compose run --rm migrate

test:
	cd api && go test -race ./...

lint:
	cd api && go vet ./...

typecheck:
	cd web && pnpm typecheck
