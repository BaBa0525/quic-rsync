FROM golang:1.20.5-bullseye AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /app/bin/ ./cmd/...

FROM debian:bullseye-slim AS runner
COPY --from=builder /app/bin/ /app/bin/
CMD ["/app/bin/rsyncd"]