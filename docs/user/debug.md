# Debugging and Troubleshooting

This document contains information about how you can retrieve additional
information for debugging and troubleshooting.

## Additional Status Information and Connection Details

You can gather additional information with `oc-client status`, from the
OC-Client and OC-Daemon logs, and with other tools. Some Examples are:

### Connected Server

You can see the server you are currently connected to in the OC-Client and
OC-Daemon logs.

### Current Profile

Currently, the log output of OC-Client contains `Location:` that also contains
the profile name.

### Connection State

You can see the current connection state with `Connected` in `oc-client
status`.

### Local IP

There are two ways to see the current IP used inside the tunnel:

- `IPv4` or `IPv6` in `Config` in `oc-client status`
- `ip address show dev $DEV`, where `$DEV` is the VPN device name found in
  `Device -> Name` in `Config` in `oc-client status`

### Server IP

The `Gateway` in `Config` in `oc-client status` is the IP address of the server
you are currently connected to.

### Sent/Received Bytes

You can view statistics about sent and received bytes on the tunnel device with
`ip -statistics a show dev $DEV`, where $DEV is the VPN device name.

### Information about Encryption

You can find Information about the used encryption, e.g., cipher suite, in the
OC-Client and OC-Daemon log output.

## Split Routing Details

You can view details about split routing with `oc-client status` and then in
`Config -> Split`.

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
