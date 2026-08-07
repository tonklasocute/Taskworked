.PHONY: dev backend frontend infra e2e

infra: ## start Postgres, Redis, MinIO
	docker compose up postgres redis minio

backend: ## run the Go API alone
	cd backend && go run ./cmd/api

e2e: ## run the E2E suite against local infra (needs `make infra` running)
	cd backend && go test -tags e2e ./e2e/... -v

frontend: ## run the Vite dev server alone
	cd frontend && npm run dev

dev: ## run backend + frontend together (Ctrl+C stops both)
	@trap 'kill 0' EXIT; \
	$(MAKE) backend & \
	$(MAKE) frontend & \
	wait
