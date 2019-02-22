FROM golang:1.11.4-alpine3.8 AS builder
LABEL maintainer="joona@kuori.org"

RUN apk add --update gcc musl-dev git

RUN go get github.com/joohoi/acme-dns
WORKDIR /go/src/github.com/joohoi/acme-dns
RUN CGO_ENABLED=1 go build

FROM alpine:latest

WORKDIR /root/
COPY --from=builder /go/src/github.com/joohoi/acme-dns .
RUN mkdir -p /etc/acme-dns
RUN mkdir -p /var/lib/acme-dns
RUN rm -rf ./config.cfg
RUN apk --no-cache add ca-certificates && update-ca-certificates

VOLUME ["/etc/acme-dns", "/var/lib/acme-dns"]
ENTRYPOINT ["./acme-dns"]
EXPOSE 53 80 443
EXPOSE 53/udp
