FROM golang:1 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o mockha ./docker/mockha/

FROM gcr.io/distroless/static
COPY --from=builder /app/mockha /mockha
ENTRYPOINT ["/mockha"]
