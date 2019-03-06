FROM golang:1.12.0-alpine3.9 AS builder

ARG ACME_DNS_VERSION=v0.7.2

WORKDIR /go/src/github.com/joohoi

RUN apk add --no-cache gcc musl-dev git && \
    git clone -b ${ACME_DNS_VERSION} --depth 1 https://github.com/joohoi/acme-dns.git acme-dns

WORKDIR /go/src/github.com/joohoi/acme-dns

RUN CGO_ENABLED=1 go build -ldflags="-s -w -linkmode external -extldflags -static"

RUN install -d -o 1000 -g 0 -m 0775 /build/etc/acme-dns /build/var/lib/acme-dns && \
    install -o 1000 -g 0 -m 0554 /go/src/github.com/joohoi/acme-dns/acme-dns /build/acme-dns && \
    install -D -o 1000 -g 0 -m 0664 /go/src/github.com/joohoi/acme-dns/config.cfg /build/etc/acme-dns/config.cfg && \
    sed -Ei -e "s/(port ?= ?)\"80\"/\1\"8080\"/" \
            -e "s/(listen ?= ?)\".*:53\"/\1\":5353\"/" /build/etc/acme-dns/config.cfg


FROM scratch

LABEL maintainer="joona@kuori.org"
COPY --from=builder /build /
USER 1000:0
EXPOSE 5353/udp 5353 8080 8443
VOLUME ["/etc/acme-dns", "/var/lib/acme-dns"]
ENTRYPOINT ["/acme-dns"]
