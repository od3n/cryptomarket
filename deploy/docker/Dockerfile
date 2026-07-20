# Build stage
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG SERVICE=api
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -buildid=" \
    -trimpath \
    -o /bin/service ./cmd/${SERVICE}

# Runtime stage — distroless static (no shell, no package manager)
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /bin/service /usr/local/bin/service
COPY --from=builder /app/migrations /migrations
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

USER nonroot:nonroot

ENTRYPOINT ["service"]
