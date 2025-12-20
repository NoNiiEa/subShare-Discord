FROM golang:1.24-alpine AS builder

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -o /app/bin/api ./cmd/api

FROM alpine:latest

RUN apk --no-cache add ca-certificates sqlite tzdata

WORKDIR /app

COPY --from=builder /app/bin/api ./api
# Config directory is mounted as volume in docker-compose, so we don't need to copy it
RUN mkdir -p ./config ./data

EXPOSE 8080

CMD ["./api"]

