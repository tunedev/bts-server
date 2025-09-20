FROM golang:1.24.4-bullseye AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -o server .
RUN CGO_ENABLED=1 GOOS=linux go build -o seed ./cmd/seed

FROM debian:bookworm-slim

WORKDIR /app

COPY --from=builder /app/ .

EXPOSE 8080

CMD ["/bin/sh", "-c", "/app/seed || echo 'seed failed (continuing)'; exec /app/server"]