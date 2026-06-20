# acme-dns

[![Build Status](https://travis-ci.org/joohoi/acme-dns.svg?branch=master)](https://travis-ci.org/joohoi/acme-dns) [![Coverage Status](https://coveralls.io/repos/github/joohoi/acme-dns/badge.svg?branch=master)](https://coveralls.io/github/joohoi/acme-dns?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/joohoi/acme-dns)](https://goreportcard.com/report/github.com/joohoi/acme-dns)

A simplified DNS server with a RESTful HTTP API to provide a simple way to automate ACME DNS challenges.

## Why?

Many DNS servers do not provide an API to enable automation for the ACME DNS challenges. Those which do, give the keys way too much power.
Leaving the keys laying around your random boxes is too often a requirement to have a meaningful process automation.

Acme-dns provides a simple API exclusively for TXT record updates and should be used with ACME magic "\_acme-challenge" - subdomain CNAME records. This way, in the unfortunate exposure of API keys, the effects are limited to the subdomain TXT record in question.

So basically it boils down to **accessibility** and **security**.

For longer explanation of the underlying issue and other proposed solutions, see a blog post on the topic from EFF deeplinks blog: https://www.eff.org/deeplinks/2018/02/technical-deep-dive-securing-automation-acme-dns-challenge-validation

## Features

- Simplified DNS server, serving your ACME DNS challenges (TXT)
- Custom records (have your required A, AAAA, NS, etc. records served)
- HTTP API automatically acquires and uses Let's Encrypt TLS certificate
- Limit /update API endpoint access to specific CIDR mask(s), defined in the /register request
- Supports SQLite & PostgreSQL as DB backends
- Rolling update of two TXT records to be able to answer to challenges for certificates that have both names: `yourdomain.tld` and `*.yourdomain.tld`, as both of the challenges point to the same subdomain.
- Simple deployment (it's Go after all)

## Usage

A client application for acme-dns with support for Certbot authentication hooks is available at: [https://github.com/acme-dns/acme-dns-client](https://github.com/acme-dns/acme-dns-client).

[![asciicast](https://asciinema.org/a/94903.png)](https://asciinema.org/a/94903)

Using acme-dns is a three-step process (provided you already have the self-hosted server set up):

- Get credentials and unique subdomain (simple POST request to eg. https://auth.acme-dns.io/register)
- Create a (ACME magic) CNAME record to your existing zone, pointing to the subdomain you got from the registration. (eg. `_acme-challenge.domainiwantcertfor.tld. CNAME a097455b-52cc-4569-90c8-7a4b97c6eba8.auth.example.org` )
- Use your credentials to POST new DNS challenge values to an acme-dns server for the CA to validate from.
- Crontab and forget.

## API

### Register endpoint

The method returns a new unique subdomain and credentials needed to update your record.
Fulldomain is where you can point your own `_acme-challenge` subdomain CNAME record to.
With the credentials, you can update the TXT response in the service to match the challenge token, later referred as \_\_\_validation\_token\_received\_from\_the\_ca\_\_\_, given out by the Certificate Authority.

**Optional:**: You can POST JSON data to limit the `/update` requests to predefined source networks using CIDR notation.

`POST /register`

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

`Status: 201 Created`

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

`POST /update`

#### Required headers

| Header name   | Description                                | Example                                               |
| ------------- |--------------------------------------------|-------------------------------------------------------|
| X-Api-User    | UUIDv4 username received from registration | `X-Api-User: c36f50e8-4632-44f0-83fe-e070fef28a10`    |
| X-Api-Key     | Password received from registration        | `X-Api-Key: htB9mR9DYgcu9bX_afHF62erXaH2TS7bg9KW3F7Z` |

#### Example input

```json
{
    "subdomain": "8e5700ea-a4bf-41c7-8a77-e990661dcc6a",
    "txt": "___validation_token_received_from_the_ca___"
}
```

#### Response

`Status: 200 OK`

```json
{
    "txt": "___validation_token_received_from_the_ca___"
}
```

### Health check endpoint

The method can be used to check readiness and/or liveness of the server. It will return status code 200 on success or won't be reachable.

`GET /health`

## Self-hosted

You are encouraged to run your own acme-dns instance, because you are effectively authorizing the acme-dns server to act on your behalf in providing the answer to the challenging CA, making the instance able to request (and get issued) a TLS certificate for the domain that has CNAME pointing to it.

See the INSTALL section for information on how to do this.

## Installation

1) Install [Go 1.14 or newer](https://golang.org/doc/install).

2) Build acme-dns:  

    ```sh
    git clone https://github.com/joohoi/acme-dns
    cd acme-dns
    export GOPATH=/tmp/acme-dns
    go build
    ```

3) Move the built acme-dns binary to a directory in your $PATH, for example:
`sudo mv acme-dns /usr/local/bin`

4) Edit config.cfg to suit your needs (see [configuration](#configuration)). `acme-dns` will read the configuration file from `/etc/acme-dns/config.cfg` or `./config.cfg`, or a location specified with the `-c` flag.

5) If your system has systemd, you can optionally install acme-dns as a service so that it will start on boot and be tracked by systemd. This also allows us to add the `CAP_NET_BIND_SERVICE` capability so that acme-dns can be run by a user other than root.

    1) Make sure that you have moved the configuration file to `/etc/acme-dns/config.cfg` so that acme-dns can access it globally.

    2) Move the acme-dns executable from `~/go/bin/acme-dns` to `/usr/local/bin/acme-dns` (Any location will work, just be sure to change `acme-dns.service` to match).

    3) Create a minimal acme-dns user: `sudo adduser --system --gecos "acme-dns Service" --disabled-password --group --home /var/lib/acme-dns acme-dns`.

    4) Move the systemd service unit from `acme-dns.service` to `/etc/systemd/system/acme-dns.service`.

    5) Reload systemd units: `sudo systemctl daemon-reload`.

    6) Enable acme-dns on boot: `sudo systemctl enable acme-dns.service`.

    7) Run acme-dns: `sudo systemctl start acme-dns.service`.

6) If you did not install the systemd service, run `acme-dns`. Please note that acme-dns needs to open a privileged port (53, domain), so it needs to be run with elevated privileges.

### Using Docker

1) Pull the latest acme-dns Docker image: `docker pull joohoi/acme-dns`.

2) Create directories: `config` for the configuration file, and `data` for the sqlite3 database.

3) Copy [configuration template](https://raw.githubusercontent.com/joohoi/acme-dns/master/config.cfg) to `config/config.cfg`.

4) Modify the `config.cfg` to suit your needs.

5) Run Docker, this example expects that you have `port = "80"` in your `config.cfg`:  

    ```sh
    docker run --rm --name acmedns                 \
    -p 53:53                                      \
    -p 53:53/udp                                  \
    -p 80:80                                      \
    -v /path/to/your/config:/etc/acme-dns:ro      \
    -v /path/to/your/data:/var/lib/acme-dns       \
    -d joohoi/acme-dns
    ```

### Docker Compose

1) Create directories: `config` for the configuration file, and `data` for the sqlite3 database.

2) Copy [configuration template](https://raw.githubusercontent.com/joohoi/acme-dns/master/config.cfg) to `config/config.cfg`.

3) Copy [docker-compose.yml from the project](https://raw.githubusercontent.com/joohoi/acme-dns/master/docker-compose.yml), or create your own.

4) Edit the `config/config.cfg` and `docker-compose.yml` to suit your needs, and run `docker-compose up -d`.

## DNS Records

Note: In this documentation:

- `auth.example.org` is the hostname of the acme-dns server, this is the `nsname` value;
- acme-dns will serve `*.auth.example.org` records;
- `198.51.100.1` is the **public** IPv4 address of the system running acme-dns.

These values should be changed based on your environment.

You will need to add some DNS records on your domain's regular DNS server:

- `NS` record for `auth.example.org` pointing to `auth.example.org` (this means, that `auth.example.org` is responsible for any `*.auth.example.org` records)
- `A` record for `auth.example.org` pointing to `198.51.100.1`
- If using IPv6, an `AAAA` record pointing to the IPv6 address.
- Each domain you will be authenticating will need a `_acme-challenge` `CNAME` subdomain added. If you are authenticating `example.com`, you will need a `CNAME` record for `_acme-challenge.example.com` pointing to the value of the `fulldomain` that you received from the `/register` endpoint.

You will need to update the configuration for acme-dns:

- `domain` should be `auth.example.org` as acme-dns will serve answers for `*.auth.example.org`;
- `nsame` should be `auth.example.org` as it's the domain name of the acme-dns server;
- the `NS`, and at least one of the `A` or `AAAA` records in the `records` array should be updated match the values in your environment.

It is unlikely that you'll need to specify any extra records, but you can append them as strings in the `records` array. The strings must be in [RFC1035](https://www.rfc-editor.org/rfc/rfc1035.html) format.

## Testing It Out

You may want to test that acme-dns is working before using it for real queries.

1) Confirm that DNS lookups for the acme-dns subdomain works as expected: `dig auth.example.org`.

2) Call the `/register` API endpoint to register a test domain:  

    ```sh
    $ curl -X POST https://auth.example.org/register
    {"username":"eabcdb41-d89f-4580-826f-3e62e9755ef2","password":"pbAXVjlIOE01xbut7YnAbkhMQIkcwoHO0ek2j4Q0","fulldomain":"d420c923-bbd7-4056-ab64-c3ca54c9b3cf.auth.example.org","subdomain":"d420c923-bbd7-4056-ab64-c3ca54c9b3cf","allowfrom":[]}
    ```

3) Call the `/update` API endpoint to set a test TXT record. Pass the `username`, `password` and `subdomain` received from the `register` call performed above:  

    ```sh
    $ curl -X POST \
    -H "X-Api-User: eabcdb41-d89f-4580-826f-3e62e9755ef2" \
    -H "X-Api-Key: pbAXVjlIOE01xbut7YnAbkhMQIkcwoHO0ek2j4Q0" \
    -d '{"subdomain": "d420c923-bbd7-4056-ab64-c3ca54c9b3cf", "txt": "___validation_token_received_from_the_ca___"}' \
    https://auth.example.org/update
    ```

    Note: The `txt` field must be exactly 43 characters long, otherwise acme-dns will reject it

4) Perform a DNS lookup to the test subdomain to confirm the updated TXT record is being served:  

    ```sh
    $ dig -t txt @auth.example.org d420c923-bbd7-4056-ab64-c3ca54c9b3cf.auth.example.org
    ```

## Configuration

```bash
[general]
# DNS interface. Note that systemd-resolved may reserve port 53 on 127.0.0.53
# In this case acme-dns will error out and you will need to define the listening interface
# for example: listen = "127.0.0.1:53"
listen = "127.0.0.1:53"
# protocol, "both", "both4", "both6", "udp", "udp4", "udp6" or "tcp", "tcp4", "tcp6"
protocol = "both"
# root domain under which the challenge subdomains will be hosted
domain = "auth.example.org"
# fully qualified domain name of the acme-dns server. Does not need to match the value of `domain`
nsname = "auth.example.org"
# admin email address, where @ is substituted with .
nsadmin = "admin.example.org"
# predefined records served in addition to the TXT
records = [
    # IP address for the `nsname` value. The dot after the value is important.
    # Format is "`nsname`. A IP.ADDRESS.GOES.HERE"
    "auth.example.org. A 198.51.100.1",
    # Optionally, you can have a AAAA record for IPv6
    # "auth.example.org. AAAA ::1",
    # This records makes `nsname` authoritative for `domain`.
    # Format is "`domain`. NS `nsname`."
    "auth.example.org. NS auth.example.org.",
]
# debug messages from CORS etc
debug = false

[database]
# Database engine to use, sqlite3 or postgres
engine = "sqlite3"
# Connection string, filename for sqlite3 and postgres://$username:$password@$host/$db_name for postgres
# Please note that the default Docker image uses path /var/lib/acme-dns/acme-dns.db for sqlite3
connection = "/var/lib/acme-dns/acme-dns.db"
# connection = "postgres://user:password@localhost/acmedns_db"

[api]
# listen ip eg. 127.0.0.1
ip = "0.0.0.0"
# disable registration endpoint
disable_registration = false
# listen port, eg. 443 for default HTTPS
port = "443"
# possible values: "letsencrypt", "letsencryptstaging", "cert", "none"
tls = "letsencryptstaging"
# only used if tls = "cert"
tls_cert_privkey = "/etc/tls/example.org/privkey.pem"
tls_cert_fullchain = "/etc/tls/example.org/fullchain.pem"
# only used if tls = "letsencrypt"
acme_cache_dir = "api-certs"
# optional e-mail address to which Let's Encrypt will send expiration notices for the API's cert
notification_email = ""
# CORS AllowOrigins, wildcards can be used
corsorigins = [
    "*"
]
# use HTTP header to get the client ip
use_header = false
# header name to pull the ip address / list of ip addresses from
header_name = "X-Forwarded-For"

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

## HTTPS API

The RESTful acme-dns API can be exposed over HTTPS in two ways:

1. Using `tls = "letsencrypt"` and letting acme-dns issue its own certificate
   automatically with Let's Encrypt.
1. Using `tls = "cert"` and providing your own HTTPS certificate chain and
   private key with `tls_cert_fullchain` and `tls_cert_privkey`.

Where possible the first option is recommended. This is the easiest and safest
way to have acme-dns expose its API over HTTPS.

**Warning**: If you choose to use `tls = "cert"` you must take care that the
certificate *does not expire*! If it does and the ACME client you use to issue the
certificate depends on the ACME DNS API to update TXT records you will be stuck
in a position where the API certificate has expired but it can't be renewed
because the ACME client will refuse to connect to the ACME DNS API it needs to
use for the renewal.

## Clients

- acme.sh: [https://github.com/Neilpang/acme.sh](https://github.com/Neilpang/acme.sh)
- Certify The Web: [https://github.com/webprofusion/certify](https://github.com/webprofusion/certify)
- cert-manager: [https://github.com/jetstack/cert-manager](https://github.com/jetstack/cert-manager)
- Lego: [https://github.com/xenolf/lego](https://github.com/xenolf/lego)
- Posh-ACME: [https://github.com/rmbolger/Posh-ACME](https://github.com/rmbolger/Posh-ACME)
- Sewer: [https://github.com/komuw/sewer](https://github.com/komuw/sewer)
- Traefik: [https://github.com/containous/traefik](https://github.com/containous/traefik)
- Windows ACME Simple (WACS): [https://www.win-acme.com](https://www.win-acme.com)

### Authentication hooks

- acme-dns-client with Certbot authentication hook: [https://github.com/acme-dns/acme-dns-client](https://github.com/acme-dns/acme-dns-client)
- Certbot authentication hook in Python:  [https://github.com/joohoi/acme-dns-certbot-joohoi](https://github.com/joohoi/acme-dns-certbot-joohoi)
- Certbot authentication hook in Go: [https://github.com/koesie10/acme-dns-certbot-hook](https://github.com/koesie10/acme-dns-certbot-hook)

### Libraries

- Generic client library in Python ([PyPI](https://pypi.python.org/pypi/pyacmedns/)): [https://github.com/joohoi/pyacmedns](https://github.com/joohoi/pyacmedns)
- Generic client library in Go: [https://github.com/cpu/goacmedns](https://github.com/cpu/goacmedns)



## TODO

- Logging to a file
- DNSSEC
- Want to see something implemented, make a feature request!

## Contributing

acme-dns is open for contributions.
If you have an idea for improvement, please open an new issue or feel free to write a PR!

## License

acme-dns is released under the [MIT License](https://www.opensource.org/licenses/MIT).