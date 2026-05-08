FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod .
COPY main.go .
RUN go build -o dopa-mock .

FROM alpine:3.20
RUN adduser -D -u 1000 appuser
COPY --from=builder /app/dopa-mock /usr/local/bin/dopa-mock
USER appuser
EXPOSE 9090
CMD ["dopa-mock"]
