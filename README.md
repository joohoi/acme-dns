[![Build Status](https://travis-ci.org/joohoi/acme-dns.svg?branch=master)](https://travis-ci.org/joohoi/acme-dns) [![Coverage Status](https://coveralls.io/repos/github/joohoi/acme-dns/badge.svg?branch=master)](https://coveralls.io/github/joohoi/acme-dns?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/joohoi/acme-dns)](https://goreportcard.com/report/github.com/joohoi/acme-dns)
# acme-dns

A simplified DNS server with a RESTful HTTP API to provide a simple way to automate ACME DNS challenges.

## Why?

Many DNS servers do not provide an API to enable automation for the ACME DNS challenges. Those which do, give the keys way too much power.
Leaving the keys laying around your random boxes is too often a requirement to have a meaningful process automation.

So basically it boils down to **accessibility** and **security**

## Features
- Simplified DNS server, serving your ACME DNS challenges (TXT)
- Custom records (have your required A, AAAA, NS, etc. records served)
- HTTP API automatically acquires and uses Let's Encrypt TLS certificate
- Simple deployment (it's Go after all)
- Supports SQLite & PostgreSQL as DB backends

## Usage

[![asciicast](https://asciinema.org/a/94462.png)](https://asciinema.org/a/94462)

Using acme-dns is a three-step process (provided you already have the self-hosted server set up, or are using a service like acme-dns.io):

- Get credentials and unique subdomain (simple GET request to https://auth.exmaple.org/register)
- Create a (ACME magic) CNAME record to your existing zone, pointing to the subdomain you got from the registration. (eg. `_acme-challenge.domainiwantcertfor.tld. CNAME a097455b-52cc-4569-90c8-7a4b97c6eba8.auth.example.org` )
- Use your credentials to POST a new DNS challenge values to an acme-dns server for the CA to validate them off of.
- Crontab and forget.

## API

### Register endpoint

The method returns a new unique subdomain and credentials needed to update your record.
Subdomain is where you can point your own `_acme-challenge` subdomain CNAME record to.
With the credentials, you can update the TXT response in the service to match the challenge token, later referred as ______my_43_char_dns_validation_token______, given out by the Certificate Authority.

```GET /register```

#### Parameters

None

```Status: 201 Created```
```
{
    "fulldomain": "8e5700ea-a4bf-41c7-8a77-e990661dcc6a.auth.acme-dns.io",
    "password": "htB9mR9DYgcu9bX_afHF62erXaH2TS7bg9KW3F7Z",
    "subdomain": "8e5700ea-a4bf-41c7-8a77-e990661dcc6a",
    "username": "c36f50e8-4632-44f0-83fe-e070fef28a10"
}
```

### Update endpoint

The method allows you to update the TXT answer contents of your unique subdomain. Usually carried automatically by automated ACME client.

```POST /update```

#### Required headers
| Header name   | Description                                | Example                                               |
| ------------- |--------------------------------------------|-------------------------------------------------------|
| X-Api-User    | UUIDv4 username recieved from registration | `X-Api-User: c36f50e8-4632-44f0-83fe-e070fef28a10`    |
| X-Api-Key     | Password recieved from registration        | `X-Api-Key: htB9mR9DYgcu9bX_afHF62erXaH2TS7bg9KW3F7Z` |

#### Example input
```
{
    "subdomain": "8e5700ea-a4bf-41c7-8a77-e990661dcc6a",
    "txt": "______my_43_char_dns_validation_token______"
}
```

#### Response

```Status: 200 OK```
```json
{
    "txt": "______my_43_char_dns_validation_token______"
}
```

## Self-hosted

You are encouraged to run your own acme-dns instance, because you are effectively authorizing the acme-dns server to act on your behalf in providing the answer to challengeing CA, making the instance able to request (and get issued) a TLS certificate for the domain that has CNAME pointing to it.

Check out how in the INSTALL section.

## As a service

Acme-dns instance is running as a service for everyone wanting to get on in fast. You can find it at `auth.acme-dns.io`, so to get started, try:
```curl -X GET https://auth.acme-dns.io/register```


## Installation

1) Install [Go](https://golang.org/doc/install)

2) Clone this repo: `git clone https://github.com/joohoi/acme-dns $GOPATH/src/acme-dns`

3) Install govendor.  ‘go get -u github.com/kardianos/govendor’ . This is used for dependency handling.

4) Get dependencies:  `cd $GOPATH/src/acme-dns` and `govendor sync`

5) Build ACME-DNS: `go build`

6) Edit config.cfg to suit your needs (see [configuration](#configuration))

7) Run acme-dns. Please note that acme-dns needs to open a privileged port (53, domain), so it needs to be run with elevated privileges.


## Configuration

```bash
[general]
# dns interface
listen = ":53"
# protocol, "udp", "udp4", "udp6" or "tcp", "tcp4", "tcp6"
protocol = "udp"
# domain name to serve the requests off of 
domain = "auth.example.org"
# zone name server 
nsname = "ns1.auth.example.org"
# admin email address, where @ is substituted with .
nsadmin = "admin.example.org"
# predefined records served in addition to the TXT
records = [
    # default A
    "auth.example.org. A 192.168.1.100",
    # A 
    "ns1.auth.example.org. A 192.168.1.100",
    "ns2.auth.example.org. A 192.168.1.100",
    # NS
    "auth.example.org. NS ns1.auth.example.org.",
    "auth.example.org. NS ns2.auth.example.org.",
]
# debug messages from CORS etc
debug = false

[database]
# Database engine to use, sqlite3 or postgres
engine = "sqlite3"
# Connection string, filename for sqlite3 and postgres://$username:$password@$host/$db_name for postgres
connection = "acme-dns.db"
# connection = "postgres://user:password@localhost/acmedns_db"

[api]
# domain name to listen requests for, mandatory if using tls = "letsencrypt"
api_domain = ""
# listen port, eg. 443 for default HTTPS
port = "8080"
# possible values: "letsencrypt", "cert", "none"
tls = "none"
# only used if tls = "cert"
tls_cert_privkey = "/etc/tls/example.org/privkey.pem"
tls_cert_fullchain = "/etc/tls/example.org/fullchain.pem"
# CORS AllowOrigins, wildcards can be used
corsorigins = [
    "*"
]

[logconfig]
# logging level: "error", "warning", "info" or "debug"
loglevel = "debug"
# possible values: stdout, TODO file & integrations
logtype = "stdout"
# file path for logfile TODO
# logfile = "./acme-dns.log"
# format, either "json" or "text" 
logformat = "text"
```

## TODO

- Ability to define the CIDR mask in POST request to /register endpoint which is authorized to make /update requests with the created user-key-pair.
- Want to see something implemented, make a feature request!

## Contributing

acme-dns is open for contributions. 
If you have an improvement, please open a Pull Request.

## License

acme-dns is released under the [MIT License](http://www.opensource.org/licenses/MIT).
