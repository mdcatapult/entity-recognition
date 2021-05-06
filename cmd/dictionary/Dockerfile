FROM golang AS builder

WORKDIR /build
COPY . .
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o dictionary cmd/dictionary/main.go

FROM golang:alpine

WORKDIR /app
COPY --from=builder /build/dictionary /app/dictionary
ENTRYPOINT [ "/app/dictionary" ]
