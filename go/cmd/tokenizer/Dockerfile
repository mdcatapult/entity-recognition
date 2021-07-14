FROM golang AS builder

WORKDIR /build
COPY go.* ./
COPY go ./go
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build go/cmd/tokenizer/main.go

FROM alpine

WORKDIR /app
COPY --from=builder /build/tokenizer /app/tokenizer
ENTRYPOINT [ "/app/tokenizer" ]
