# syntax=docker/dockerfile:1

FROM golang:1.22-alpine AS builder
WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/config-service ./main.go

FROM alpine:3.20
WORKDIR /app

RUN adduser -D -H -u 10001 appuser
COPY --from=builder /out/config-service /app/config-service
COPY Config /app/Config

ENV PORT=8080
ENV CONFIG_SOURCE_PATH=Config
EXPOSE 8080

USER appuser
ENTRYPOINT ["/app/config-service"]
