# Development Documentation

This document contains information about the current features and components.

## Features

Features of the current implementation:

- Split Routing
  - Static Split Excludes
  - Dynamic DNS-based Split Excludes
  - Static Exclude Local Networks
- Trusted Network Detection
- Always-On Configuration/Traffic Policing
  - Captive Portal Detection
- Protocol Support
  - TLS, DTLS
- Authentication Methods
  - Machine Certificate + Username/Password
  - Machine Certificate + User Certificate (with OpenConnect v9.00 or later)

## Components

OC-Daemon consists of the components depicted in the following figure:

```
+---------------------------------------------------------------------+
| Daemon                                                              |
|                                                                     |
|  +-------------+  +--------------+  +-------+  +-----------------+  |
|  | Trusted     |  | Traffic      |  | DNS   |  | VPN Setup       |  |
|  | Network     |  | Policing     |  | Proxy |  |                 |  |
|  | Detection   |  |              |  |       |  | +-------------+ |  |
|  +-------------+  | +----------+ |  |       |  | | Split       | |  |
|  +-------------+  | | DevMon   | |  |       |  | | Routing     | |  |
|  | Profile     |  | +----------+ |  |       |  | |             | |  |
|  | Monitor     |  | +----------+ |  |       |  | | +---------+ | |  |
|  +-------------+  | | DNSMon   | |  |       |  | | | DevMon  | | |  |
|  +-------------+  | +----------+ |  |       |  | | +---------+ | |  |
|  | Sleep       |  | +----------+ |  |       |  | | +---------+ | |  |
|  | Monitor     |  | | CPD      | |  |       |  | | | AddrMon | | |  |
|  +-------------+  | +----------+ |  |       |  | | +---------+ | |  |
|  +-------------+  | +----------+ |  |       |  | |             | |  |
|  | OpenConnect |  | | Resolver | |  |       |  | |             | |  |
|  | Runner      |  | +----------+ |  |       |  | +-------------+ |  |
|  +-------------+  +--------------+  +-------+  +-----------------+  |
|                                                                     |
+------------------[D-Bus API]----------[Socket API]------------------+
                        |                    |
                        |                    |
                  +------------+      +------------+
                  |   Client   |      | VPNCScript |
                  +------------+      +------------+
```

OC-Daemon consists of the three interacting components: `Daemon`, `Client` and
`VPNCScript`. The `Daemon` contains the following subcomponents: `Trusted
Network Detection`, `Profile Monitor`, `Sleep Monitor`, `OpenConnect Runner`,
`Traffic Policing`, `DNS Proxy`, `VPN Setup`, `D-Bus API` and `Socket API`.
`Trusted Network Detection` detects whether the host is connected to a trusted
network. `Profile Monitor` detects changes to the XML Profile in the host's
file system. `Sleep Monitor` detects whether the host is going to and waking up
from sleep/hibernation mode. `OpenConnect Runner` manages the OpenConnect
subprocess for the VPN connection. If enabled, `Traffic Policing` ensures that
only VPN(-related) traffic is allowed. `Traffic Policing` again consists of the
following subcomponents: `DevMon`, `DNSMon`, `CPD`, `Resolver`. The device
monitor `DevMon` detects addition or removal of network devices. The DNS
monitor `DNSMon` detects changes to the host`s DNS configuration. The Captive
Portal Detection `CPD` detects whether the host is behind a captive portal.
`Resolver` resolves allowed DNS names to IP addresses. When the VPN connection
is active, `DNS Proxy` is used as the host's DNS resolver that forwards and
monitors DNS queries to enable DNS-based Split Excludes. When the VPN
connection is active, `VPN Setup` is responsible for the VPN network
configuration including the DNS configuration and the setup of the VPN network
device. `VPN Setup` uses the subcomponent `Split Routing`. `Split Routing`
manages split routing with Split Excludes. It retrieves updates of DNS-based
Split Excludes from the `DNS Proxy`. `Split Routing` contains the subcomponents
`DevMon` and `AddrMon`. Like in `Traffic Policing` the device monitor `DevMon`
detects addition and removal of network devices. The address monitor `AddrMon`
detects changes of the host's IP addresses. In order to interact with the
`Daemon`, the `Client` uses the `D-Bus API` and the `VPNCScript` uses the
`Socket API`.

Additional information can be found in the following subdocuments:

- [Overview](overview.md)
- [D-Bus API and Socket API](api.md)
- [VPN Network Configuration](vpn-network-config.md)
- [Split Routing](split-routing.md)
- [DNS Configuration](dns-config.md)
- [Trusted Network Detection](trusted-network.md)
- [Traffic Policing](traffic-policing.md)
