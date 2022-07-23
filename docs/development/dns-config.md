# DNS Configuration

The basic DNS configuration sets up the DNS-Proxy as resolver when the VPN
tunnel is active.

## DNS-Proxy

The DNS-Proxy runs inside the oc-daemon process. It acts as DNS resolver and
handles the DNS queries of the host. Its main purpose is retrieving IP
addresses for Dynamic DNS-based Split Excludes and, thus, is configured
accordingly with the VPN network configuration and communicates with other
components inside the oc-daemon. It performs the following operations:

* Forwarding of DNS queries to remote DNS servers
  * DNS-Servers in VPN configuration
* Monitoring of Dynamic DNS-based Split Exclude domain names
  * Check domain names in DNS queries using watch list
  * Report A records to oc-daemon
  * Report AAAA records to oc-daemon
  * Store CNAMES in watch list (with a timeout)

## Split-DNS

If Split-DNS is enabled in the VPN Network Configuration, DNS queries that
match the configured DNS names should not be routed to the VPN DNS server.
Instead, they should be routed to an appropriate existing DNS server that is
not in the VPN.

Not implemented.
