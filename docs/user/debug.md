# Debugging and Troubleshooting

This document contains information about how you can retrieve additional
information for debugging and troubleshooting.

## Additional Status Information and Connection Details

You can gather additional information with `oc-client status`, from the
OC-Client and OC-Daemon logs, and with other tools. Some Examples are:

### Connected Server

You can see the server you are currently connected to in the `Current Server`
line in `oc-client status`.

### Current Profile

Currently, the log output of OC-Client contains `Location:` that also contains
the profile name.

### Connection State

You can see the current connection state in the `Connection State` line in
`oc-client status`.

### Local IP

You can see the current IP used inside the tunnel in the `IP` line in
`oc-client status`.

### Server IP

The `Gateway` entry in the `VPN Config` line in `oc-client status -verbose` is
the IP address of the server you are currently connected to.

### Sent/Received Bytes

You can view statistics about sent and received bytes on the tunnel device with
`ip -statistics a show dev $DEV`, where $DEV is the VPN device name (default:
`oc-daemon-tun0`).

### Information about Encryption

You can find information about the used encryption, e.g., cipher suite, in the
OC-Client and OC-Daemon log output.

## Split Routing Details

You can view details about split routing with `oc-client status -verbose` in
the `VPN Config` line and then in the `Split` entry.

`Split` contains the Split Routing configuration with IPv4 address ranges in
`ExcludeIPv4`, IPv6 address ranges in `ExcludeIPv6` and  DNS-based Excludes in
`ExcludeDNS`.

This is only the static configuration you receive when connecting to the VPN.
DNS-based Split Excludes are dynamic. You can view the current static and
dynamic IPv4 and IPv6 excludes with the following commands:

```console
$ # view IPv4 excludes
$ sudo nft list set inet oc-daemon-routing excludes4
$ # view IPv6 excludes
$ sudo nft list set inet oc-daemon-routing excludes6
```
