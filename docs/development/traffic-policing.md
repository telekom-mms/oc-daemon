# Traffic Policing

Traffic Policing ensures that only VPN and VPN-related traffic is allowed
unless we are connected to a trusted network or there is an exception. It uses
the Always-On settings in the XML Profile to determine if it is enabled or
disabled and to configure allowed hosts. Also, there is an exception flag in
the VPN Configuration. It is received when connecting to the VPN and, if set,
temporarily disables Traffic Policing until the OpenConnect Daemon is
restarted. Traffic Policing is disabled on a trusted network.

On startup, Traffic Policing performs the following configuration steps:

* Set firewall rules using nftables
  * Add a set for allowed network devices
  * Add sets for allowed IPv4/6 hosts
  * Allow traffic of related and established connections
  * Allow traffic on allowed devices
  * Allow incoming and outgoing ICMPv4 and ICMPv6 traffic
  * Allow incoming and outgoing DHCPv4 and DHCPv6 traffic
  * Allow outgoing DNS traffic (port 53)
  * Allow outgoing traffic to allowed IPv4/6 hosts
  * Block all other traffic
* Start Device Monitor
  * Monitor Netlink device events: added, removed devices
  * Check if device is real or virtual
  * Add/Remove virtual devices in set of allowed network devices
* Start DNS Monitor
  * Watch DNS resolv.conf file changes
  * Resolve/Update IP addresses in sets of allowed IPv4/6 hosts
  * Trigger Captive Portal Detection
* Start Captive Portal Detection (CPD)
  * If portal is detected:
    * Allow HTTP(S) traffic (ports 80 and 443)
  * If portal is not detected anymore (after login):
    * Remove HTTP(S) traffic exception
    * Resolve/Update all IPs in sets of allowed IPv4/6 hosts

## Captive Portal Detection

Captive Portal Detection (CPD) detects a captive portal and adds respective
firewall exceptions in Traffic Policing, so we can log onto the network.

Captive Portal Detection uses Ubuntu's portal detection scheme. It sends an
HTTP request to `connectivity-check.ubuntu.com` and expects the response `204
(No Content)`. If CPD receives a different response, it assumes, there is a
portal.

In order to allow Ubuntu's and other portal detection schemes, the following
CPD hosts are added to the allowed IPv4/6 hosts:

- `connectivity-check.ubuntu.com` (Ubuntu)
- `detectportal.firefox.com` (Firefox)
- `www.gstatic.com` (Chrome)
- `clients3.google.com` (Chromium)
- `nmcheck.gnome.org` (Gnome)

## ICMP

ICMPv4 and ICMPv6 configuration with Traffic Policing:

| ICMPv4 Types            | Incoming | Outgoing |
|-------------------------|----------|----------|
| echo-reply              | ok       | reject   |
| destination-unreachable | ok       | reject   |
| source-quench           | ok       | ok       |
| redirect                | ok       | reject   |
| echo-request            | drop     | ok       |
| time-exceeded           | ok       | reject   |
| parameter-problem       | ok       | reject   |
| timestamp-request       | drop     | ok       |
| timestamp-reply         | ok       | reject   |
| info-request            | drop     | ok       |
| info-reply              | ok       | reject   |
| address-mask-request    | drop     | ok       |
| address-mask-reply      | ok       | reject   |
| router-advertisement    | ok       | reject   |
| router-solicitation     | drop     | ok       |

| ICMPv6 Types            | Incoming | Outgoing |
|-------------------------|----------|----------|
| destination-unreachable | ok       | reject   |
| packet-too-big          | ok       | reject   |
| time-exceeded           | ok       | reject   |
| echo-request            | drop     | ok       |
| echo-reply              | ok       | reject   |
| mld-listener-query      | ok       | reject   |
| mld-listener-report     | ok       | ok       |
| mld2-listener-report    | ok       | ok       |
| mld-listener-done       | ok       | ok       |
| nd-router-solicit       | drop     | ok       |
| nd-router-advert        | ok       | reject   |
| nd-neighbor-solicit     | ok       | ok       |
| nd-neighbor-advert      | ok       | ok       |
| ind-neighbor-solicit    | ok       | ok       |
| ind-neighbor-advert     | ok       | ok       |
| nd-redirect             | ok       | reject   |
| parameter-problem       | ok       | reject   |
| router-renumbering      | ok       | reject   |
