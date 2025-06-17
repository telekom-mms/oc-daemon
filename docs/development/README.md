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

```
+----------------------------------------------+
| Daemon                                       |
|                                              |
|  +---------------+            +-----------+  |
|  | VPN Network   |            | Trusted   |  |
|  | Configuration |            | Network   |  |
|  |               |            | Detection |  |
|  +---------------+            +-----------+  |
|  +---------+ +---------------+ +----------+  |
|  | Split   | | DNS           | | Traffic  |  |
|  | Routing | | Configuration | | Policing |  |
|  +---------+ +---------------+ +----------+  |
|                                              |
+----------------[Daemon API]------------------+
                   /      \
                  /        \
      +------------+      +------------+
      |   Client   |      | VPNCScript |
      +------------+      +------------+
```

```

VPN Setup
- Device Configuration
- DNS Configuration
- Split Routing
  - Device Monitor
  - Address Monitor

DNS Proxy

Traffic Policing
- Device Monitor
- DNS Monitor
- Captive Portal Detection
- DNS Resolver

Trusted Network Detection

+---------------------------------------------------------------------------+
| Daemon                                                                    |
|                                                                           |
| - Profile Monitor                                                         |
| - Sleep Monitor                                                           |
| - OpenConnect Runner                                                      |
|                                                                           |
|  +------------+  +------------+  +-------+  +--------------------------+  |
|  | Traffic    |  | Trusted    |  | DNS   |  | VPN Setup                |  |
|  | Policing   |  | Network    |  | Proxy |  |                          |  |
|  |            |  | Detection  |  |       |  | +-----------+ +--------+ |  |
|  | - Devmon   |  |            |  |       |  | | Split     | | DNS    | |  |
|  | - DNSMon   |  |            |  |       |  | | Routing   | | Setup  | |  |
|  | - CPD      |  |            |  |       |  | |           | +--------+ |  |
|  | - Resolver |  |            |  |       |  | | - DevMon  | +--------+ |  |
|  |            |  |            |  |       |  | | - AddrMon | | Device | |  |
|  |            |  |            |  |       |  | |           | | Setup  | |  |
|  |            |  |            |  |       |  | +-----------+ +--------+ |  |
|  +------------+  +------------+  +-------+  +--------------------------+  |
|                                                                           |
+---------------------[D-Bus API]----------[Socket API]---------------------+
                           |                    |
                           |                    |
                     +------------+      +------------+
                     |   Client   |      | VPNCScript |
                     +------------+      +------------+

+----------------------------------------------------------------------------+
| Daemon                                                                     |
|                                                                            |
|  +-------------+  +------------+  +-------+  +--------------------------+  |
|  | Trusted     |  | Traffic    |  | DNS   |  | VPN Setup                |  |
|  | Network     |  | Policing   |  | Proxy |  |                          |  |
|  | Detection   |  |            |  |       |  | +-----------+ +--------+ |  |
|  +-------------+  | - Devmon   |  |       |  | | Split     | | DNS    | |  |
|  +-------------+  | - DNSMon   |  |       |  | | Routing   | | Setup  | |  |
|  | Profile     |  | - CPD      |  |       |  | |           | |        | |  |
|  | Monitor     |  | - Resolver |  |       |  | | - DevMon  | |        | |  |
|  +-------------+  |            |  |       |  | | - AddrMon | |        | |  |
|  +-------------+  |            |  |       |  | |           | +--------+ |  |
|  | Sleep       |  |            |  |       |  | |           | +--------+ |  |
|  | Monitor     |  |            |  |       |  | |           | | Device | |  |
|  +-------------+  |            |  |       |  | |           | | Setup  | |  |
|  +-------------+  |            |  |       |  | |           | |        | |  |
|  | OpenConnect |  |            |  |       |  | |           | |        | |  |
|  | Runner      |  |            |  |       |  | +-----------+ +--------+ |  |
|  +-------------+  +------------+  +-------+  +--------------------------+  |
|                                                                            |
+---------------------[D-Bus API]----------[Socket API]----------------------+
                           |                    |
                           |                    |
                     +------------+      +------------+
                     |   Client   |      | VPNCScript |
                     +------------+      +------------+

+----------------------------------------------------------------------------+
| Daemon                                                                     |
|                                                                            |
|  +-------------+  +------------+  +-------+  +--------------------------+  |
|  | Trusted     |  | Traffic    |  | DNS   |  | VPN Setup                |  |
|  | Network     |  | Policing   |  | Proxy |  |                          |  |
|  | Detection   |  |            |  |       |  | +-----------+ +--------+ |  |
|  +-------------+  | +--------+ |  |       |  | | Split     | | DNS    | |  |
|  +-------------+  | | DevMon | |  |       |  | | Routing   | | Setup  | |  |
|  | Profile     |  | +--------+ |  |       |  | |           | |        | |  |
|  | Monitor     |  | +--------+ |  |       |  | | +-------+ | |        | |  |
|  +-------------+  | | DNSMon | |  |       |  | | |DevMon | | |        | |  |
|  +-------------+  | +--------+ |  |       |  | | +-------+ | +--------+ |  |
|  | Sleep       |  | +--------+ |  |       |  | | +-------+ | +--------+ |  |
|  | Monitor     |  | | CPD    | |  |       |  | | |AddrMon| | | Device | |  |
|  +-------------+  | +--------+ |  |       |  | | +-------+ | | Setup  | |  |
|  +-------------+  | +--------+ |  |       |  | |           | |        | |  |
|  | OpenConnect |  | |Resolver| |  |       |  | |           | |        | |  |
|  | Runner      |  | +--------+ |  |       |  | +-----------+ +--------+ |  |
|  +-------------+  +------------+  +-------+  +--------------------------+  |
|                                                                            |
+---------------------[D-Bus API]----------[Socket API]----------------------+
                           |                    |
                           |                    |
                     +------------+      +------------+
                     |   Client   |      | VPNCScript |
                     +------------+      +------------+

TODO: Sub-documents D-Bus API, Socket API
```

- [Overview](overview.md)
- [Daemon D-Bus/Socket API](api.md)
- [VPN Network Configuration](vpn-network-config.md)
- [Split Routing](split-routing.md)
- [DNS Configuration](dns-config.md)
- [Trusted Network Detection](trusted-network.md)
- [Traffic Policing](traffic-policing.md)
