FROM golang:1 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /hats
RUN CGO_ENABLED=0 GOOS=linux go build -o /healthcheck ./cmd/healthcheck/main.go

FROM builder AS tester

RUN go test -v

FROM debian:12-slim

COPY --from=builder --chmod=755 /hats /hats
COPY --from=builder --chmod=755 /healthcheck /healthcheck
ENTRYPOINT [ "/hats" ]