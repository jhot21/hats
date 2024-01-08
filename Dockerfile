FROM golang:1 as builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /hats
RUN CGO_ENABLED=0 GOOS=linux go build -o /healthcheck ./cmd/healthcheck/main.go

FROM builder as tester

RUN go test -v

FROM scratch

COPY --from=builder --chmod=755 /hats /hats
COPY --from=builder --chmod=755 /healthcheck /healthcheck
ENTRYPOINT [ "/hats" ]