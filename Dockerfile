ARG ALPINE_VERSION=3.10

FROM golang:1.12-alpine${ALPINE_VERSION} AS builder

ARG ACME_DNS_VERSION=v0.7.2

WORKDIR /go/src/github.com/joohoi

RUN apk add --no-cache gcc musl-dev git && \
    git clone -b ${ACME_DNS_VERSION} --depth 1 https://github.com/joohoi/acme-dns.git acme-dns

WORKDIR /go/src/github.com/joohoi/acme-dns

RUN CGO_ENABLED=1 go build

RUN install -d -o 1000 -g 0 -m 0775 /build/etc/acme-dns /build/var/lib/acme-dns && \
    install -D -o 1000 -g 0 -m 0554 /go/src/github.com/joohoi/acme-dns/acme-dns /build/usr/local/bin/acme-dns && \
    install -D -o 1000 -g 0 -m 0664 /go/src/github.com/joohoi/acme-dns/config.cfg /build/etc/acme-dns/config.cfg && \
    sed -Ei -e "s/(port ?= ?)\"80\"/\1\"8080\"/" \
            -e "s/(listen ?= ?)\".*:53\"/\1\":5353\"/" /build/etc/acme-dns/config.cfg


FROM alpine:${ALPINE_VERSION}

LABEL maintainer="joona@kuori.org"
RUN apk add --no-cache ca-certificates
COPY --from=builder /build /
USER 1000:0
EXPOSE 5353/udp 5353 8080 8443
HEALTHCHECK --interval=10s --timeout=5s CMD wget --spider --quiet -S -T 1 http://127.0.0.1:8080/health || exit 1
VOLUME ["/etc/acme-dns", "/var/lib/acme-dns"]
ENTRYPOINT ["/usr/local/bin/acme-dns"]
