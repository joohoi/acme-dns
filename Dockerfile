FROM golang:1.11.4-alpine3.8 AS builder

ENV ACME_DNS_VERSION=v0.6

WORKDIR /go/src/github.com/joohoi

RUN apk add --no-cache gcc musl-dev git && \
    git clone -b ${ACME_DNS_VERSION} --depth 1 https://github.com/joohoi/acme-dns.git acme-dns

WORKDIR /go/src/github.com/joohoi/acme-dns

RUN CGO_ENABLED=1 go build -ldflags="-s -w -linkmode external -extldflags -static"

RUN mkdir -p /build/etc/acme-dns && \
    mkdir -p /build/var/lib/acme-dns && \
    cp /go/src/github.com/joohoi/acme-dns/acme-dns /build && \
    cp /go/src/github.com/joohoi/acme-dns/config.cfg /build/etc/acme-dns && \
    sed -Ei -e "s/(port ?= ?)\"80\"/\1\"8080\"/" \
            -e "s/(listen ?= ?)\":53\"/\1\":5353\"/" /build/etc/acme-dns/config.cfg && \
    chown 1000:0 /build/etc/acme-dns /build/var/lib/acme-dns && \
    chmod ug+w /build/etc/acme-dns /build/var/lib/acme-dns && \
    chmod +x /build/acme-dns


FROM scratch

LABEL maintainer="joona@kuori.org"
COPY --from=builder /build /
USER 1000:0
EXPOSE 5353/udp 5353 8080 8443
VOLUME ["/etc/acme-dns", "/var/lib/acme-dns"]
CMD ["/acme-dns"]
