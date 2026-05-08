# Thailand General Election 2026 — Portfolio Project

A full-stack, production-architecture election platform demonstrating real-time vote counting, anonymous online voting, and distributed systems design.

---

## Architecture

```
Browser (Next.js 16)
     │  REST + WebSocket
     ▼
Kong API Gateway (rate limiting · API key auth · JWT validation)
     │
     ├──▶ Go API Server (Gin) ──▶ PostgreSQL 16
     │         │                       ▲
     │         ├──▶ Redis ×3           │
     │         │   persistent │  ←── Worker (Kafka consumers)
     │         │   cache      │         │
     │         │   asynq      │         └──▶ Centrifugo (WebSocket push)
     │         │
     │         └──▶ Redpanda (Kafka-compatible)
     │
     └──▶ Centrifugo ──▶ Browser (live results)
```

### Key design choices

| Concern | Approach |
|---|---|
| Vote anonymisation | `vote_events` and `voter_rights_used` share **no FK** — the `anonymous_token` bridges them but cannot be reversed to a national ID |
| Idempotency | Redis `SET NX EX 86400` on every ingestion request + unique constraint fallback |
| Real-time | Centrifugo WebSocket push after every Kafka worker commit; TanStack Query polls every 10 s as fallback |
| Concurrency | `SET vote_lock:{sessionID} NX EX 10` distributed lock around the atomic cast TX |
| Party-list seats | Largest-remainder (d'Hondt variant) recalculated every 60 s by an asynq job |
| Circuit breaker | `sony/gobreaker` wraps Redis reads; falls back to PostgreSQL when tripped |

---

## Quick Start

### Prerequisites

- Docker Desktop (Windows/macOS) or Docker + Docker Compose (Linux)
- Go 1.22+ (only needed if running outside Docker)
- Node.js 20+ (only needed if running frontend outside Docker)

### 1. Clone and configure

```bash
git clone https://github.com/your-handle/thailand-election.git
cd thailand-election
cp .env.example .env
```

Edit `.env` — change every `change_me_*` value. Required before first run:

```
POSTGRES_PASSWORD=...
REDIS_PERSISTENT_PASSWORD=...
REDIS_CACHE_PASSWORD=...
REDIS_ASYNQ_PASSWORD=...
CENTRIFUGO_TOKEN_SECRET=...
NATIONAL_ID_PEPPER=...       # openssl rand -hex 32
PHONE_ENCRYPTION_KEY=...     # openssl rand -hex 32
```

### 2. Generate JWT signing keys

```bash
make generate-keys      # writes backend/keys/jwt_private.pem + jwt_public.pem
```

### 3. Generate simulator Ed25519 key

```bash
make generate-sim-key   # prints key pair — paste into .env
```

### 4. Start all services

```bash
make up
# Waits for: postgres → redis ×3 → redpanda → centrifugo → kong → api → worker → frontend
```

### 5. Run migrations and seed data

```bash
make migrate            # runs all DB migrations
make seed               # seeds parties, provinces, constituencies, candidates, test voters
```

### 6. Open the app

| URL | What |
|---|---|
| http://localhost:3000 | Public election dashboard |
| http://localhost:3000/vote | Online voting portal |
| http://localhost:8001 | Centrifugo admin |
| http://localhost:8081 | Asynqmon (job queue monitor) |
| http://localhost:16686 | (optional) Grafana dashboard |

### 7. Generate live demo data

```bash
make simulate           # 50 RPS physical votes for 30 s — watch the dashboard update
make simulate-online    # 10 concurrent voter sessions (requires OTP_DEV_MODE=true)
make simulate-verify    # run + verify DB ↔ Redis totals match
```

---

## Feature Map

| PRD ID | Feature | Status |
|---|---|---|
| B-01 | Physical vote ingestion (Ed25519 signed, Kafka) | ✅ |
| B-02 | National summary API | ✅ |
| B-03 | Province summary API | ✅ |
| B-04 | Party-list seat calculator | ✅ |
| B-05 | Referendum summary API | ✅ |
| B-09 | Health check (DB + Redis + Kafka + Centrifugo) | ✅ |
| B-10 | Centrifugo JWT token issuer | ✅ |
| B-11 | Admin batch vote entry | ✅ |
| B-12 | Voter identity verification (DOPA mock) | ✅ |
| B-13 | OTP request + asynq delivery | ✅ |
| B-14 | OTP verification + anonymous RS256 JWT | ✅ |
| B-15 | Voter eligibility check | ✅ |
| B-16 | Ballot retrieval | ✅ |
| B-17 | Atomic vote cast (distributed lock + TX) | ✅ |
| B-18 | Receipt hash generation + verification | ✅ |
| F-01 | Thailand province tile map | ✅ |
| F-02 | National leaderboard (real-time) | ✅ |
| F-03 | Province drill-down | ✅ |
| F-04 | Party-list seat allocation display | ✅ |
| F-05 | Referendum chart | ✅ |
| F-09 | Connection status banner (reconnecting/offline) | ✅ |
| F-10 | Skeleton loading screens | ✅ |
| W-01..07 | Kafka vote consumers + atomic DB updater | ✅ |
| W-08..13 | asynq scheduled jobs (reconcile, seat calc, cleanup) | ✅ |

---

## Security Architecture — Vote Anonymisation

```
Voter                  API                        DB
  │                     │                          │
  │  national_id ──────▶│  SHA-256(id + pepper)   │
  │                     │  ──────────────────────▶ voter_registry (hash only)
  │                     │                          │
  │  OTP verified       │  anonymous_token (UUID)  │
  │◀── RS256 JWT ───────│  ──────────────────────▶ voter_sessions
  │    (no PII)         │                          │
  │                     │  BEGIN TX                │
  │  POST /cast ───────▶│  INSERT vote_events      │  ← anonymous_token, NO national_id
  │                     │  INSERT voter_rights_used │  ← session_id, ballot_type, NO choice
  │                     │  COMMIT                   │
  │◀── receipt_hash ────│                          │
  │                     │                          │
  │                     │  ⚠ vote_events ───────── NO FK ──▶ voter_rights_used
  │                     │  ⚠ voter_rights_used ─── NO FK ──▶ vote_events
```

A national ID can never be linked to a vote choice — the two tables share no foreign key and the `anonymous_token` is a one-way bridge generated fresh for each ballot session.

---

## What's Mocked / Simplified

| Real System | Portfolio Approach |
|---|---|
| DOPA government API | `dopa-mock` container returns `{valid:true}` for any 13-digit ID |
| Twilio/SNS SMS | `OTP_DEV_MODE=true` — OTP returned in API response, displayed in browser |
| HSM / Vault secrets | RSA-4096 key pair in local `.env` file |
| Ed25519 PKI (77 polling stations) | Single test key pair shared by the simulator |
| argon2id (production params) | Same algorithm, lighter params for dev speed |

---

## Project Structure

```
thailand-election/
├── backend/
│   ├── cmd/api/           Go API server entry point
│   ├── cmd/worker/        Kafka consumers + asynq scheduler
│   ├── cmd/simulator/     Load-testing CLI
│   ├── scripts/genkey/    Ed25519 key generator
│   ├── internal/
│   │   ├── api/           Gin handlers, middleware, server setup
│   │   ├── cache/         Redis clients + circuit breaker
│   │   ├── config/        Viper config loader
│   │   ├── db/sqlc/       Generated type-safe DB layer
│   │   ├── domain/        Pure business logic (reporting, voting, auth)
│   │   ├── kafka/         Producer + message types
│   │   ├── realtime/      Centrifugo gocent publisher
│   │   └── worker/        Kafka consumers, atomic updater, asynq tasks
│   └── db/
│       ├── migrations/    golang-migrate SQL (18 files)
│       └── queries/       sqlc source queries
├── frontend/
│   ├── app/               Next.js 16 App Router pages
│   │   ├── page.tsx       Election dashboard (SSR + WebSocket)
│   │   ├── province/[id]/ Province detail page
│   │   ├── referendum/    Referendum dashboard
│   │   └── vote/          Online voting portal (5 steps)
│   ├── components/
│   │   ├── map/           Province tile map + detail panel
│   │   ├── leaderboard/   National results + seat allocation
│   │   ├── referendum/    Referendum chart
│   │   └── shared/        Skeleton, ErrorBoundary, ConnectionStatus
│   └── lib/
│       ├── api/client.ts  TanStack Query hooks
│       ├── ws/centrifuge  Centrifuge v5 WebSocket singleton
│       ├── store/ui.ts    Zustand UI state
│       └── voting/        Session storage + voting API
├── docker/
│   ├── kong/kong.yml      Declarative API gateway config
│   ├── nginx/nginx.conf   TLS termination (dev: HTTP only)
│   └── dopa-mock/         Fake DOPA government API
├── .github/workflows/ci.yml  GitHub Actions pipeline
├── docker-compose.yml
├── Makefile
└── .env.example
```

---

## Development Commands

```bash
make up                  # Start all Docker services
make down                # Stop all services
make migrate             # Run DB migrations
make seed                # Seed master data + test voters
make test                # Go tests with race detector
make lint                # go vet + eslint
make simulate            # Physical vote load test (50 RPS, 30 s)
make simulate-online     # Online voter session test
make simulate-verify     # Load test + DB↔Redis consistency check
make generate-keys       # Create RSA-4096 JWT key pair
make generate-sim-key    # Create Ed25519 simulator signing key
make generate            # Regenerate sqlc query code
```

---

## Tech Stack

**Backend:** Go 1.22 · Gin · pgx/sqlc · golang-migrate · segmentio/kafka-go · go-redis · asynq · gobreaker · zerolog · Prometheus · JWT (RS256 + HMAC)

**Frontend:** Next.js 16 · React 19 · TypeScript · Tailwind v4 · TanStack Query v5 · Centrifuge.js v5 · Zustand v5 · Recharts · react-hook-form · shadcn/ui

**Infrastructure:** PostgreSQL 16 · Redis 7 (×3) · Redpanda (Kafka) · Centrifugo · Kong · Docker Compose
