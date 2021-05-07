FROM golang AS builder

WORKDIR /build
COPY go.* .
COPY go .
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o tokenizer go/cmd/tokenizer/main.go

FROM golang:alpine

WORKDIR /app
COPY --from=builder /build/tokenizer /app/tokenizer
ENTRYPOINT [ "/app/tokenizer" ]
