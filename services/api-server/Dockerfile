FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY internal/ ./internal/
COPY api-server/ ./api-server/
WORKDIR /app/api-server
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o api-server .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/api-server/api-server .
EXPOSE 3000
CMD ["./api-server"]
