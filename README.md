ACME-DNS.io
===========

Simplified DNS server with convinient HTTP API for ACME DNS authentication handling in large environments or in environments with DNS servers without API.

Problems ACME-DNS is addressing
-------------------------------------------------
**Enabling ACME DNS authentication for domains hosted in environment without convinient API**

Many DNS servers don't provide good enough API for this kind record management. And/or support is finicky or experimental.

**Making automating DNS authenticated renewal more secure**

Traditional DNS servers / services that have a good API just are not designed around this kind of a need, and using them would require leaving your API credentials laying around every box that uses them. Completely compromising your whole zone, and possibly more (through compromising your email using MX record)


Self-hosted of as a service?
--------------------------------------
ACME-DNS is open source with appropriate license, and you are encouraged to host an instance yourself. If however you would like to use it as a service, we're hosting. 


Features
------------
* Simplified DNS server, serving your ACME DNS challenges (TXT)
* Custom records (have your required A, AAAA, NS, whatever records served)
* HTTP API automatically gets and uses Let's Encrypt certificate
* Written in GO, so super simple deployment
* Easy configuration
* Supports SQLite & PostgreSQL

How does it work?
--------------------------
**1) Register an account**

Sounds more fancy than it is, basically means: do a GET request and recieve credentials, and your unique subdomain.
>$ curl https://auth.acme-dns.io/register
>
>{
>    "fulldomain": "23752ef1-118a-4ed8-912d-74dcad2178d9.auth.acme-dns.io",
>    "username": "e9afe5a9-d3c5-b57f-d3c5-25975fa367c5",
>    "password": "DoVJaBgx0ps2bxy7UoffZ41KcgT15oLCZj1k353q",
>    "subdomain": "23752ef1-118a-4ed8-912d-74dcad2178d9"
>}

And recieve your account:

- "fulldomain" - Your CNAME alias target
- "username" - Your username, send this in "X-Api-User" - HTTP header with update requests
- "password" - Your password, send this in "X-Api-Key" - HTTP header with update requests
- "subdomain" - This is your subdomain, provided for more easily crafting update request data

**2) Point your _acme-challenge.example.org magic subdomain CNAME to the "fulldomain" received from the registration above.**

This has to be done only once, when setting the domain up for the first time. 

Here, if I would like to get certificate for domain "my.example.org", I would create a CNAME record "_acme-challenge.my.example.org" for zone "example.org" pointing to the fulldomain I recieved earlier, like this: 
>_acme-challenge.my.example.org. CNAME 23752ef1-118a-4ed8-912d-74dcad2178d9.auth.acme-dns.io.

CNAME works like a link when CA queries your DNS for authentication token.

**3) Hook it up to your ACME client**

Make your ACME client to update the TXT record on ACME-DNS when requesting / renewing a certficate. For example:

> $ curl -X POST https://auth.acme-dns.io/update
>-H "X-Api-User: e9afe5a9-d3c5-b57f-d3c5-25975fa367c5" 
>-H "X-Api-Key: DoVJaBgx0ps2bxy7UoffZ41KcgT15oLCZj1k353q" 
>--data '{"subdomain": "23752ef1-118a-4ed8-912d-74dcad2178d9", 
>         "txt": "'"$DNS_AUTHENTICATION_TOKEN"'"}'
>         
>{"txt":"70ymFptYJA_Cz63ADEaES5-8NqNV74NbEqD62Ap_dMo"}%

Self-hosted setup
===========

You decided to host your own, that's awesome! However there's some details you should prepare first before getting on with the actual software installation.

Now, we're going to install a DNS server that will take care of serving DNS for your domain, or most likely subdomain. We really suggest using delegated subdomain, because ACME-DNS really isn't designed to be used as an actual DNS server.

#### Subdomain NS

You'll need to create two records to your existing zone. For this example, I'm using the setup of hosted ACME-DNS.io

Create A record for your to-become-acme-dns-server to your domain zone:
`ns1.auth.acme-dns.io.     A     10.1.1.53`
Create a NS record to delegate everything under your subdomain to the ACME-DNS server:
`auth.acme-dns.io.  NS  ns1.auth.acme-dns.io.`

You'll want to make sure you have the NS records in your actual ACME-DNS configuration as well:

> ...
> records = [
>   # default A
>   "auth.acme-dns.io. A 10.1.1.53",
>   # A
>   "ns1.auth.acme-dns.io. A 10.1.1.53",
>   # NS
>   "auth.acme-dns.io. NS ns1.auth.acme-dns.io.",
>]
>...


Installation
------------

1) Install [Go](https://golang.org/doc/install) and set your `$GOPATH` environment variable
2) Clone this repo: `git clone https://github.com/joohoi/acme-dns $GOPATH/src/acme-dns`
3) Get dependencies:  `cd $GOPATH/src/acme-dns` and `go get -u`
4) Build ACME-DNS: `go build`
5) Edit config.cfg to suit your needs (see [configuration](#configuration))
6) Run acme-dns `sudo ./acme-dns` in most cases you need to run it as privileged user, because we usually need privileged ports.

Configuration
-------------------

...


TODO
----

- PostgreSQL support
- Let user to POST to registration with CIDR masks to allow updates from

Contributing
------------

ACME-DNS is open for contributions. So please if you have something you would wish to see improved, or would like to improve yourself, submit an issue or pull request!

License
--------

ACME-DNS is released under the [MIT License](http://www.opensource.org/licenses/MIT).
