FROM golang:1.9.2-alpine AS builder

RUN apk add --update gcc musl-dev git

RUN go get github.com/joohoi/acme-dns
WORKDIR /go/src/github.com/joohoi/acme-dns
RUN CGO_ENABLED=1 go build

FROM alpine:latest

WORKDIR /root/
COPY --from=builder /go/src/github.com/joohoi/acme-dns .

ENTRYPOINT ["./acme-dns"]
EXPOSE 8080
