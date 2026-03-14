FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY internal/ ./internal/
COPY analytics-service/ ./analytics-service/
WORKDIR /app/analytics-service
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o analytics-service .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/analytics-service/analytics-service .
CMD ["./analytics-service"]
