FROM golang AS builder

WORKDIR /build
COPY go.* ./
COPY go ./go
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o rest-api go/cmd/rest-api/main.go

FROM golang:alpine

WORKDIR /app
COPY --from=builder /build/rest-api /app/rest-api
ENTRYPOINT [ "/app/rest-api", "--config", "/app/config.yml" ]
