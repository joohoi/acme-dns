[![Build Status](https://travis-ci.org/joohoi/acme-dns.svg?branch=master)](https://travis-ci.org/joohoi/acme-dns) [![Coverage Status](https://coveralls.io/repos/github/joohoi/acme-dns/badge.svg?branch=master)](https://coveralls.io/github/joohoi/acme-dns?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/joohoi/acme-dns)](https://goreportcard.com/report/github.com/joohoi/acme-dns)
# acme-dns

A simplified DNS server with a RESTful HTTP API to provide a simple way to automate ACME DNS challenges.

## Why?

Many DNS servers do not provide an API to enable automation for the ACME DNS challenges. Those which do, give the keys way too much power.
Leaving the keys laying around your random boxes is too often a requirement to have a meaningful process automation.

Acme-dns provides a simple API exclusively for TXT record updates and should be used with ACME magic "\_acme-challenge" - subdomain CNAME records. This way, in the unfortunate exposure of API keys, the effetcs are limited to the subdomain TXT record in question.

So basically it boils down to **accessibility** and **security**

## Features
- Simplified DNS server, serving your ACME DNS challenges (TXT)
- Custom records (have your required A, AAAA, NS, etc. records served)
- HTTP API automatically acquires and uses Let's Encrypt TLS certificate
- Limit /update API endpoint access to specific CIDR mask(s), defined in the /register request
- Supports SQLite & PostgreSQL as DB backends
- Rolling update of two TXT records to be able to answer to challenges for certificates that have both names: `yourdomain.tld` and `*.yourdomain.tld`, as both of the challenges point to the same subdomain.
- Simple deployment (it's Go after all)

## Usage
[![asciicast](https://asciinema.org/a/94903.png)](https://asciinema.org/a/94903)

Using acme-dns is a three-step process (provided you already have the self-hosted server set up):

- Get credentials and unique subdomain (simple POST request to eg. https://auth.acme-dns.io/register)
- Create a (ACME magic) CNAME record to your existing zone, pointing to the subdomain you got from the registration. (eg. `_acme-challenge.domainiwantcertfor.tld. CNAME a097455b-52cc-4569-90c8-7a4b97c6eba8.auth.example.org` )
- Use your credentials to POST a new DNS challenge values to an acme-dns server for the CA to validate them off of.
- Crontab and forget.

## API

### Register endpoint

The method returns a new unique subdomain and credentials needed to update your record.
Fulldomain is where you can point your own `_acme-challenge` subdomain CNAME record to.
With the credentials, you can update the TXT response in the service to match the challenge token, later referred as \_\_\_validation\_token\_recieved\_from\_the\_ca\_\_\_, given out by the Certificate Authority.

**Optional:**: You can POST JSON data to limit the /update requests to predefined source networks using CIDR notation.

```POST /register```

#### OPTIONAL Example input
```json
{
    "allowfrom": [
        "192.168.100.1/24",
        "1.2.3.4/32",
        "2002:c0a8:2a00::0/40"
    ]
}
```


```Status: 201 Created```
```json
{
    "allowfrom": [
        "192.168.100.1/24",
        "1.2.3.4/32",
        "2002:c0a8:2a00::0/40"
    ],
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
```json
{
    "subdomain": "8e5700ea-a4bf-41c7-8a77-e990661dcc6a",
    "txt": "___validation_token_recieved_from_the_ca___"
}
```

#### Response

```Status: 200 OK```
```json
{
    "txt": "___validation_token_recieved_from_the_ca___"
}
```

## Self-hosted

You are encouraged to run your own acme-dns instance, because you are effectively authorizing the acme-dns server to act on your behalf in providing the answer to challengeing CA, making the instance able to request (and get issued) a TLS certificate for the domain that has CNAME pointing to it.

Check out how in the INSTALL section.


## Installation

1) Install [Go 1.9 or newer](https://golang.org/doc/install)

2) Clone this repo: `git clone https://github.com/joohoi/acme-dns $GOPATH/src/acme-dns`

3) Build ACME-DNS: `go build`

4) Edit config.cfg to suit your needs (see [configuration](#configuration))

5) Run acme-dns. Please note that acme-dns needs to open a privileged port (53, domain), so it needs to be run with elevated privileges.

## Using Docker

1) Pull the latest acme-dns Docker image: `docker pull joohoi/acme-dns` 

2) Create directories: `config` for the configuration file, and `data` for the sqlite3 database.

3) Copy [configuration template](https://raw.githubusercontent.com/joohoi/acme-dns/master/config.cfg) to `config/config.cfg` 

4) Modify the config.cfg to suit your needs.

5) Run Docker, this example expects that you have `port = "80"` in your config.cfg:
```
docker run --rm --name acmedns                 \
 -p 53:53                                      \
 -p 80:80                                      \
 -v /path/to/your/config:/etc/acme-dns:ro      \
 -v /path/to/your/data:/var/lib/acme-dns       \
 -d joohoi/acme-dns
```

## Docker Compose

1) Create directories: `config` for the configuration file, and `data` for the sqlite3 database.

2) Copy [configuration template](https://raw.githubusercontent.com/joohoi/acme-dns/master/config.cfg) to `config/config.cfg` 

3) Copy [docker-compose.yml from the project](https://raw.githubusercontent.com/joohoi/acme-dns/master/docker-compose.yml), or create your own.

4) Edit the `config/config.cfg` and `docker-compose.yml` to suit your needs, and run `docker-compose up -d`

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
# autocert HTTP port, eg. 80 for answering Let's Encrypt HTTP-01 challenges. Mandatory if using tls = "letsencrypt".
autocert_port = "80"
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
# use HTTP header to get the client ip
use_header = false
# header name to pull the ip address / list of ip addresses from
header_name = "X-Forwarded-For"
```

## Changelog
- v0.3.2 Dockerfile was fixed for users using autocert feature
- v0.3.1 Added goreleaser for distributing binary builds of the releases
- v0.3 Changed autocert to use HTTP-01 challenges, as TLS-SNI is disabled by Let's Encrypt
- v0.2 Now powered by httprouter, support wildcard certificates, Docker images
- v0.1 Initial release

## TODO

- Logging to a file
- DNSSEC
- Want to see something implemented, make a feature request!

## Contributing

acme-dns is open for contributions. 
If you have an idea for improvement, please open an new issue or feel free to write a PR!

## License

acme-dns is released under the [MIT License](http://www.opensource.org/licenses/MIT).
