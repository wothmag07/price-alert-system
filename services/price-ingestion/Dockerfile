FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY internal/ ./internal/
COPY price-ingestion/ ./price-ingestion/
WORKDIR /app/price-ingestion
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o price-ingestion .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/price-ingestion/price-ingestion .
CMD ["./price-ingestion"]
