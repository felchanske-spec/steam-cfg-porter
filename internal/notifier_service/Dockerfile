FROM golang:1.27-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o notifier_service cmd/notifier/main.go

FROM alpine:3.20

RUN adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder /build/notifier_service .

USER appuser

CMD [ "./notifier_service" ]