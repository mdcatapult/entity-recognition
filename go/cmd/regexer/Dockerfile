FROM golang AS builder

WORKDIR /build
COPY . .
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o regexer cmd/regexer/main.go

FROM golang:alpine

WORKDIR /app
COPY --from=builder /build/regexer /app/regexer
ENTRYPOINT [ "/app/regexer" ]
