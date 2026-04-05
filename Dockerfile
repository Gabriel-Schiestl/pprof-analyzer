# Stage 1: Build
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /pprof-analyzer ./cmd/pprof-analyzer

# Stage 2: Runtime
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /pprof-analyzer /usr/local/bin/pprof-analyzer

ENTRYPOINT ["pprof-analyzer"]
