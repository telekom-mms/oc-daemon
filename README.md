# OpenConnect Daemon

OpenConnect Daemon allows a user to connect to a Cisco AnyConnect VPN. It uses
[openconnect](https://www.infradead.org/openconnect/), Linux policy-based
routing and [nftables](https://nftables.org/projects/nftables/) to support
static as well as DNS-based exclusion of traffic from the tunnel (split
tunneling) and prevention of unprotected network access on untrusted networks
(Always-On VPN). The OpenConnect Daemon runs as systemd service and the user
interacts with it using the oc-client tool.

## Installation

Please see [Installation](docs/user/install.md) for installation instructions.

## Usage

You can connect to the VPN with your current settings with:

```console
$ oc-client
```

or

```console
$ oc-client connect
```

You can list VPN servers in your XML profile (`/var/lib/oc-daemon/profile.xml`)
with:

```console
$ oc-client list
```

You can show the current status with:

```console
$ oc-client status
```

You can disconnect the VPN with:

```console
$ oc-client disconnect
```

Please see [Usage](docs/user/usage.md) for more usage and configuration
information.

## Documentation

Please see the [docs](docs/) folder for user and development documentation.
