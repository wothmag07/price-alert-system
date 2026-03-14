FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY internal/ ./internal/
COPY notification-service/ ./notification-service/
WORKDIR /app/notification-service
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o notification-service .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/notification-service/notification-service .
CMD ["./notification-service"]
