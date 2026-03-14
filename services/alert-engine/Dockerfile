FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY internal/ ./internal/
COPY alert-engine/ ./alert-engine/
WORKDIR /app/alert-engine
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o alert-engine .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/alert-engine/alert-engine .
CMD ["./alert-engine"]
