# ---- Build stage ----
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /price-ingestion .

# ---- Runtime stage ----
FROM alpine:3.19

RUN apk add --no-cache ca-certificates
COPY --from=builder /price-ingestion /price-ingestion

ENTRYPOINT ["/price-ingestion"]
