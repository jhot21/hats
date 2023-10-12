FROM golang:1 as builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /hats

FROM builder as tester

RUN go test -v

FROM scratch

COPY --from=builder --chmod=755 /hats /hats
ENTRYPOINT [ "/hats" ]