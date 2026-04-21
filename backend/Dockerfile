FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /kite ./cmd/http

FROM alpine:3.20 AS production

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -S kite && adduser -S kite -G kite
USER kite

COPY --from=builder /kite /usr/local/bin/kite

EXPOSE 8080
ENTRYPOINT ["kite"]
