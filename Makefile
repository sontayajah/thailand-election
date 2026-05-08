.PHONY: up down build logs migrate seed generate-keys generate-sim-key simulate \
        simulate-online simulate-verify test test-cover lint swagger

# ─── Docker ───────────────────────────────────────────────────
up:
	docker-compose up -d

down:
	docker-compose down

build:
	docker-compose build --no-cache

logs:
	docker-compose logs -f api worker

ps:
	docker-compose ps

# ─── Database ─────────────────────────────────────────────────
migrate:
	docker-compose run --rm api sh -c "migrate -path /app/db/migrations -database $$DATABASE_URL up"

migrate-down:
	docker-compose run --rm api sh -c "migrate -path /app/db/migrations -database $$DATABASE_URL down 1"

seed:
	docker-compose exec api go run ./scripts/seed/main.go

# ─── Code Generation ──────────────────────────────────────────
generate:
	cd backend && sqlc generate

# Generate RSA-4096 key pair for JWT signing (voter + admin tokens).
# Requires openssl on PATH.
generate-keys:
	@mkdir -p backend/keys
	@echo "Generating RSA-4096 JWT key pair..."
	openssl genrsa -out backend/keys/jwt_private.pem 4096
	openssl rsa -in backend/keys/jwt_private.pem -pubout -out backend/keys/jwt_public.pem
	@echo "Keys written to backend/keys/ (never commit these)"

# Generate Ed25519 key pair used by the vote simulator and seeded into province_keys.
generate-sim-key:
	@echo "Generating Ed25519 simulator signing key..."
	cd backend && go run ./scripts/genkey/main.go
	@echo "Copy the SIMULATOR_ED25519_PRIVATE_KEY line above into your .env file"

# ─── Simulator ────────────────────────────────────────────────
simulate:
	cd backend && go run ./cmd/simulator --mode=physical --rps=50 --duration=30s

simulate-online:
	cd backend && go run ./cmd/simulator --mode=online --rps=10 --duration=30s

simulate-verify:
	cd backend && go run ./cmd/simulator --mode=physical --rps=50 --duration=30s --verify

# ─── Tests ────────────────────────────────────────────────────
test:
	cd backend && go test ./... -race -count=1 -timeout=120s

test-cover:
	cd backend && go test ./... -race -coverprofile=coverage.out && go tool cover -html=coverage.out

lint:
	cd backend && go vet ./...
	cd frontend && npm run lint

# ─── Swagger ──────────────────────────────────────────────────
swagger:
	cd backend && swag init -g cmd/api/main.go -o docs/swagger
