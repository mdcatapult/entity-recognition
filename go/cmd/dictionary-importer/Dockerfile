FROM golang AS builder

WORKDIR /build
COPY go.* ./
COPY go ./go
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ./go/cmd/dictionary-importer/...

FROM alpine

WORKDIR /app
COPY --from=builder /build/dictionary-importer /app/dictionary-importer
ENTRYPOINT [ "/app/dictionary-importer" ]
