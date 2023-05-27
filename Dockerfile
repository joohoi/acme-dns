FROM golang:alpine AS builder
LABEL maintainer="joona@kuori.org"

# required for sqlite/CGO:
RUN apk add --update gcc musl-dev git
WORKDIR /tmp/acme-dns
ADD go.mod go.sum ./
RUN go mod download
# generally cacheable steps until here
ADD *.go ./
RUN CGO_ENABLED=1 go build


FROM alpine:latest
# required for getting LE certs for the API:
RUN apk --no-cache add ca-certificates && update-ca-certificates
RUN adduser -D -u 1000 -h /var/lib/acme-dns acme-dns
RUN mkdir -p /etc/acme-dns
WORKDIR /var/lib/acme-dns
USER 1000
COPY --from=builder /tmp/acme-dns/acme-dns /usr/local/bin/acme-dns

VOLUME ["/etc/acme-dns", "/var/lib/acme-dns"]
ENTRYPOINT ["acme-dns"]
EXPOSE 8053 8053/udp 8080 8443
