FROM golang:1.24.1-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/wallets ./cmd/wallets

FROM alpine:latest

RUN apk add --no-cache tzdata ca-certificates

RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

WORKDIR /app

COPY --from=builder --chown=appuser:appgroup /app/wallets .

EXPOSE 8080

CMD ["/app/wallets"]