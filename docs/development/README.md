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
  - TODO: Machine Certificate + User Certificate (with newer OpenConnect version)?

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
|  +------------+  +-------------+  +-------+  +--------------------------+  |
|  | Traffic    |  | Trusted     |  | DNS   |  | VPN Setup                |  |
|  | Policing   |  | Network     |  | Proxy |  |                          |  |
|  |            |  | Detection   |  |       |  | +-----------+ +--------+ |  |
|  | - Devmon   |  +-------------+  |       |  | | Split     | | DNS    | |  |
|  | - DNSMon   |  +-------------+  |       |  | | Routing   | | Setup  | |  |
|  | - CPD      |  | Profile     |  |       |  | |           | |        | |  |
|  | - Resolver |  | Monitor     |  |       |  | | - DevMon  | |        | |  |
|  |            |  +-------------+  |       |  | | - AddrMon | +--------+ |  |
|  |            |  +-------------+  |       |  | |           | +--------+ |  |
|  |            |  | Sleep       |  |       |  | |           | | Device | |  |
|  |            |  | Monitor     |  |       |  | |           | | Setup  | |  |
|  |            |  +-------------+  |       |  | |           | |        | |  |
|  |            |  +-------------+  |       |  | |           | |        | |  |
|  |            |  | OpenConnect |  |       |  | |           | +--------+ |  |
|  |            |  | Runner      |  |       |  | +-----------+            |  |
|  +------------+  +-------------+  +-------+  +--------------------------+  |
|                                                                            |
+---------------------[D-Bus API]----------[Socket API]----------------------+
                           |                    |
                           |                    |
                     +------------+      +------------+
                     |   Client   |      | VPNCScript |
                     +------------+      +------------+

+---------------------------------------------------------------------------+
| Daemon                                                                    |
|                                  +--------------------------+             |
|                                  | VPN Setup                |             |
|                                  |                          |             |
|                                  | +-----------+ +--------+ |             |
|                                  | | Split     | | DNS    | |             |
|                                  | | Routing   | | Setup  | |             |
|                                  | |           | +--------+ |             |
|                                  | | - DevMon  | +--------+ |             |
|                                  | | - AddrMon | | Device | |             |
|                                  | |           | | Setup  | |             |
|                                  | +-----------+ +--------+ |             |
|                                  +--------------------------+             |
|                                                                           |
|  +------------+  +------------+  +-------+  +-------------+  +---------+  |
|  | Traffic    |  | Trusted    |  | DNS   |  | OpenConnect |  | Profile |  |
|  | Policing   |  | Network    |  | Proxy |  | Runner      |  | Monitor |  |
|  |            |  | Detection  |  |       |  |             |  |         |  |
|  | - Devmon   |  |            |  |       |  |             |  |         |  |
|  | - DNSMon   |  |            |  |       |  |             |  +---------+  |
|  | - CPD      |  |            |  |       |  |             |  +---------+  |
|  | - Resolver |  |            |  |       |  |             |  | Sleep   |  |
|  |            |  |            |  |       |  |             |  | Monitor |  |
|  |            |  |            |  |       |  |             |  |         |  |
|  |            |  |            |  |       |  |             |  |         |  |
|  +------------+  +------------+  +-------+  +-------------+  +---------+  |
|                                                                           |
+---------------------[D-Bus API]----------[Socket API]---------------------+
                           |                    |
                           |                    |
                     +------------+      +------------+
                     |   Client   |      | VPNCScript |
                     +------------+      +------------+


TODO: Sub-documents D-Bus API, Socket API
```

- [Overview](overview.md)
- [Daemon API](api.md)
- [VPN Network Configuration](vpn-network-config.md)
- [Split Routing](split-routing.md)
- [DNS Configuration](dns-config.md)
- [Trusted Network Detection](trusted-network.md)
- [Traffic Policing](traffic-policing.md)
