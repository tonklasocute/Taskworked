.PHONY: dev backend frontend infra

infra: ## start Postgres, Redis, MinIO
	docker compose up postgres redis minio

backend: ## run the Go API alone
	cd backend && go run ./cmd/api

frontend: ## run the Vite dev server alone
	cd frontend && npm run dev

dev: ## run backend + frontend together (Ctrl+C stops both)
	@trap 'kill 0' EXIT; \
	$(MAKE) backend & \
	$(MAKE) frontend & \
	wait
