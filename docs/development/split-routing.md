# Split Routing

A combination of Linux Policy Routing and nftables is used to route everything
over the VPN tunnel unless there is an exception.

Linux Policy Routing:

* Extra routing table `42111` for VPN connection
  * Contains default route over the tunnel
* Use main routing table for packets coming from the tunnel
* Use routes in table `42111` except for packets with mark `42111`
* Route packets with mark `42111` using routes in main table
* Effect
  * marked traffic is not routed over the tunnel
  * unmarked traffic is routed over the tunnel

Nftables:

* Extra nftables table containing the configuration for the VPN
* Mark packets belonging to the tunnel itself (connection to VPN server)
* Mark packets going to Split Exclude addresses
* Mark packets going to DNS-based Split Exclude addresses (configured in a
  nftables set data structure)
* Use SNAT/Masquerading for marked packets to make sure source address matches
  outgoing network interface
* Use Connection Tracking for marked packets to allow removing Exclude
  addresses without affecting currently active network connections

Note: this configuration is only active as long as the VPN tunnel is active.
When the connection is terminated, this configuration is also removed. Also,
this is just used for routing. It is not meant to perform "firewalling",
enforce anything or prevent the user from anything.

## Static Split Excludes

Implemented using the scheme above.

## Dynamic DNS-based Split Excludes

DNS-Proxy is configured as resolver when VPN connection is up. DNS-Proxy checks
DNS traffic for DNS-based Split Exclude domains and reports the IP addresses
and TTLs of the DNS entries. The IP addresses are added to the nftables set for
the DNS-based Split Exclude addresses.

The TTL can be used to realize cleaning of old entries in the DNS-based Split
Exclude set. Nftables provides a timeout mechanism that automatically removes
entries after the timeout. However, the timeout cannot be reset from userspace
(only from the datapath with a nftables rule). Re-adding an existing entry does
not reset the timeout. This could lead to the following race condition:

1. A DNS query yields an address that should be in the set
2. The IP is added to the set
3. The IP already exists in the set, so it is not added/updated
4. The timeout of the IP in the set expires and so the IP is removed
5. The first packet related to the DNS query/the IP arrives in nftables
6. There is no entry in the set and the packet is not handled as desired

Thus, we do not use the nftables timeout mechanism in the set. Instead, we
maintain a copy of the set in user space and handle the TTLs/timeouts there.
If new entries are added or old entries are removed, we reconfigure the
nftables set atomically.

## Static Exclude Local Networks

Static Exclude Local Networks is implemented using the policy routing scheme
above. Packets are routed according to the following cases:

- Static Exclude Local Networks not configured
  - Everything should be routed over the tunnel when it is up
- `0.0.0.0/32`
  - Everything except local network traffic should be routed over the tunnel
    when it is up
  - Network addresses of local networks are added to the exclude lists so that
    they are marked for split routing
- `0.0.0.0/32` + `BypassVirtualSubnetsOnlyV4`
  - Everything except local virtual network traffic should be routed over the
    tunnel when it is up
  - Network addresses of local virtual networks are added to the exclude lists
    so that they are marked for split routing
