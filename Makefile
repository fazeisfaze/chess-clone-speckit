.PHONY: test backend-test frontend-test dev-db backend-dev frontend-dev migrate

test: backend-test frontend-test

backend-test:
	go test ./...

frontend-test:
	cd web && npm test

dev-db:
	docker compose up -d db

backend-dev:
	go run ./cmd/server

frontend-dev:
	cd web && npm run dev

migrate:
	go run ./cmd/server -migrate
