# Changelog

## Unreleased
- Bump `golang.org/x/crypto` to v0.56.0, `golang.org/x/net` to v0.58.0 and `golang.org/x/text` to v0.41.0, and raise the minimum Go version to 1.26 and the toolchain to 1.27.1, to pick up the upstream security fixes these versions carry (see PR for the CVE list)
- Add index on `txt(Subdomain)` so DNS lookups no longer degrade to full table scans as registrations grow (created idempotently on startup, applies to existing databases)
- Split DB timeouts: a short read timeout on the DNS hot path so a stalled connection no longer pins a worker for 20s after the resolver has already given up
- Recycle idle PostgreSQL connections within a minute (`SetConnMaxIdleTime`) so a silently-dropped TCP connection is retired before it stalls the next query borrowing it

## v2.0
- Update goreleaser configuration and add a GitHub action to build a release on new version tags (#395)
- Huge refactoring and modernization (#325)
- Breaking change:
  - config option `database.engine` value `sqlite3` change to `sqlite`

## v1.1
- Add timeout to golangci job (#369)
- Update deps to support go 1.23 (#368)
- Bump dependencies (#334)

## v1.0 
   - New
      - Refactoring of the codebase to something more robust
   - Changed
      - Updated dependencies
- v0.8
   - NOTE: configuration option: "api_domain" deprecated!
   - New
      - Automatic HTTP API certificate provisioning using DNS challenges making acme-dns able to acquire certificates even with HTTP api not being accessible from public internet.
      - Configuration value for "tls": "letsencryptstaging". Setting it will help you to debug possible issues with HTTP API certificate acquiring process. This is the new default value.
   - Changed
      - Fixed: EDNS0 support
      - Migrated from autocert to [certmagic](https://github.com/mholt/certmagic) for HTTP API certificate handling
- v0.7.2
   - Changed
      - Fixed: Regression error of not being able to answer to incoming random-case requests.
      - Fixed: SOA record added to a correct header field in NXDOMAIN responses.
- v0.7.1
   - Changed
      - Fixed: SOA record correctly added to the TCP DNS server when using both, UDP and TCP servers.
- v0.7
   - New
      - Added an endpoint to perform health checks
   - Changed
      - A new protocol selection for DNS server "both", that binds both - UDP and TCP ports.
      - Refactored DNS server internals.
      - Handle some aspects of DNS spec better.
- v0.6
   - New
      - Command line flag `-c` to specify location of config file.
      - Proper refusal of dynamic update requests.
      - Release signing
   - Changed
      - Better error messages for goroutines
- v0.5
   - New
      - Configurable certificate cache directory
   - Changed
      - Process wide umask to ensure created files are only readable by the user running acme-dns
      - Replaced package that handles UUIDs because of a flaw in the original package
      - Updated dependencies
      - Better error messages
- v0.4 Clear error messages for bad TXT record content, proper handling of static CNAME records, fixed IP address parsing from the request, added option to disable registration endpoint in the configuration.
- v0.3.2 Dockerfile was fixed for users using autocert feature
- v0.3.1 Added goreleaser for distributing binary builds of the releases
- v0.3 Changed autocert to use HTTP-01 challenges, as TLS-SNI is disabled by Let's Encrypt
- v0.2 Now powered by httprouter, support wildcard certificates, Docker images
- v0.1 Initial release
