.PHONY: dev backend frontend infra e2e migrate backup backup-verify restore-postgres restore-minio

infra: ## start Postgres, Redis, MinIO
	docker compose up postgres redis minio

migrate: ## apply pending DB migrations against local infra (needs `make infra` running)
	cd backend && go run ./cmd/migrate

backend: ## run the Go API alone (run `make migrate` first)
	cd backend && go run ./cmd/api

e2e: ## run the E2E suite against local infra (needs `make infra` running)
	cd backend && go test -tags e2e ./e2e/... -v

frontend: ## run the Vite dev server alone
	cd frontend && npm run dev

dev: ## run backend + frontend together (Ctrl+C stops both)
	$(MAKE) -j2 backend frontend

backup: ## run one full backup now (postgres + minio, verified, pruned)
	docker compose run --rm backup /scripts/run-full-backup.sh

backup-verify: ## re-verify the most recent backups without taking new ones
	docker compose run --rm backup sh -c '/scripts/verify-postgres-backup.sh && /scripts/verify-minio-backup.sh'

restore-postgres: ## restore a dump: make restore-postgres FILE=/backups/postgres/xxx.dump TARGET_DB=taskworked_restore
	docker compose run --rm backup /scripts/restore-postgres.sh "$(FILE)" "$(TARGET_DB)" $(CONFIRM)

restore-minio: ## restore a snapshot: make restore-minio DIR=/backups/minio/xxx TARGET_BUCKET=taskworked-restore
	docker compose run --rm backup /scripts/restore-minio.sh "$(DIR)" "$(TARGET_BUCKET)" $(CONFIRM)
