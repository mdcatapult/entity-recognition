FROM golang AS builder

WORKDIR /build
COPY go.* ./
COPY go ./go
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o dictionary-init go/cmd/dictionary-init/main.go

FROM golang:alpine

WORKDIR /app
COPY --from=builder /build/dictionary-init /app/dictionary-init
ENTRYPOINT [ "/app/dictionary-init", "--config", "/app/config.yml" ]
