FROM golang AS builder

WORKDIR /build
COPY go.* ./
COPY go ./go
RUN mkdir recognition-api && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ./go/cmd/recognition-api/...

FROM alpine

WORKDIR /app
COPY --from=builder /build/recognition-api /app/recognition-api
ENTRYPOINT [ "/app/recognition-api" ]
