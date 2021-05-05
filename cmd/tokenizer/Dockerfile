FROM golang AS builder

WORKDIR /build
COPY . .
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o tokenizer

FROM golang:alpine

COPY --from=builder /build/tokenizer /tokenizer
ENTRYPOINT [ "/tokenizer" ]
