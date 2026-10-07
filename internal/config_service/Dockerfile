FROM golang:1.27-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CSGO_ENABLED=0 GOOS=linux go build -o config_service cmd/main.go

FROM alpine:3.20

RUN adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder /build/config_service .

RUN chown -R appuser:appuser /app

USER appuser

EXPOSE 50051

EXPOSE 8080

CMD [ "./config_service" ]