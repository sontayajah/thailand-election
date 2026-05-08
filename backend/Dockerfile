# ─────────────────────────────────────────────────────────────
#  Backend Dockerfile — Multi-stage build
#  Stage: development  → hot-reload with Air
#  Stage: builder      → compile binary
#  Stage: production   → minimal distroless image
# ─────────────────────────────────────────────────────────────

# ── Development stage (hot-reload) ───────────────────────────
FROM golang:1.25-alpine AS development
RUN apk add --no-cache git curl && \
    go install github.com/air-verse/air@latest
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
EXPOSE 8080
CMD ["air", "-c", ".air.toml"]

# ── Builder stage ─────────────────────────────────────────────
FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG CMD_TARGET=api
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o /bin/app ./cmd/${CMD_TARGET}

# ── Production stage ──────────────────────────────────────────
FROM gcr.io/distroless/static:nonroot AS production
COPY --from=builder /bin/app /app
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app"]
