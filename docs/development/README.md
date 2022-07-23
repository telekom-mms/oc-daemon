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

- [Overview](overview.md)
- [Daemon API](api.md)
- [VPN Network Configuration](vpn-network-config.md)
- [Split Routing](split-routing.md)
- [DNS Configuration](dns-config.md)
- [Trusted Network Detection](trusted-network.md)
- [Traffic Policing](traffic-policing.md)
