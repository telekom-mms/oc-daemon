# Tests with ocserv

These tests test OC-Daemon with [ocserv][ocserv] in a test setup with two
networks and multiple nodes created with podman-compose.

## Requirements

- podman
  - tested with version 4.3.1 (Debian 12) and 5.5.1
  - [rootless][rootless] with configured [subuid/subgid][subuid] for your user
- podman-compose

## Quick Run

Run in the root directory of the Git repository:

- Build Debian package with `./scripts/build-podman-test.sh`
- Run all tests with `./test/ocserv/tests.sh all`

Important: remember to re-build the Debian package before running the tests
when the OC-Daemon code changed.

See Examples below for other build options, running specific tests etc.

## Test Setup

The test setup is shown in the following figure:

```
.................................................................................................
: oc-daemon-test-       : oc-daemon-test-                               : oc-daemon-test-       :
: ext-portal            : ext                                           : int                   :
:                       :                                               :                       :
:   _________________   :   _________________       _________________   :   _________________   :
:  |                 |  :  |                 |     |                 |  :  |                 |  :
:  | oc-daemon-test- |__:__| oc-daemon-test- |_____| oc-daemon-test- |__:__| oc-daemon-test- |  :
:  | oc-daemon       |  :  | portal          |  |  | ocserv          |  :  | web-int         |  :
:  |_________________|  :  |_________________|  |  |_________________|  :  |_________________|  :
:                       :                       |                       :                       :
:                       :                       |   _________________   :                       :
:                       :                       |  |                 |  :                       :
:                       :                       |__| oc-daemon-test- |  :                       :
:                       :                       |  | web-ext         |  :                       :
:                       :                       |  |_________________|  :                       :
:                       :                       |                       :                       :
:                       :                       |   _________________   :                       :
:                       :                       |  |                 |  :                       :
:                       :                       |__| oc-daemon-test- |  :                       :
:                       :                          | conncheck       |  :                       :
:                       :                          |_________________|  :                       :
:                       :                                               :                       :
:.......................:...............................................:.......................:
```

The test setup consists of the three networks `oc-daemon-test-ext-portal`,
`oc-daemon-test-ext` and `oc-daemon-test-int` as well as the six nodes
(services, containers) `oc-daemon-test-oc-daemon`, `oc-daemon-test-portal`,
`oc-daemon-test-ocserv`, `oc-daemon-test-web-ext`, `oc-daemon-test-conncheck`
and `oc-daemon-test-web-int`. For brevity, the common prefix `oc-daemon-test-`
in the names is omitted in the following description. `oc-daemon` is only in
network `ext-portal`. `web-ext` and `conncheck` are only in network `ext`.
`web-int` is only in network `int`. `portal` is in the two networks
`ext-portal` and `ext`. `ocserv` is in both networks `ext` and `int`.
`oc-daemon` runs OC-Daemon and acts as VPN client. `portal` runs [Caddy][caddy]
and acts as captive portal and router between the networks `ext-portal` and
`ext`. `ocserv` runs [ocserv][ocserv] and acts as VPN server. `web-ext` and
`web-int` both run [Caddy][caddy] and act as web servers with HTTPS, so they
can also be used as Trusted Network Detection (TND) servers. `conncheck` also
runs [Caddy][caddy] and can be used as connectivity check server for Captive
Portal Detection (CPD). `ocserv` connects VPN clients to the network `int` and,
thus, to `web-int`. So, `oc-daemon` can reach `web-int` when it is connected to
the VPN via `ocserv`. Otherwise, it can only reach the other nodes like
`web-ext` (if permitted by `portal`, as by default). So, `ext` acts as
external, untrusted network and `int` as internal, trusted network.
`ext-portal` acts as an additional network with captive portal for CPD tests.

Currently, there are two versions of the test setup: the IPv4 version and the
IPv6 version. In the IPv4 version, the nodes only use IPv4 addresses. In the
IPv6 version, the nodes use both IPv4 and IPv6 addresses.

The test setup versions are used by the individual tests. Each test is
responsible for starting and stopping the test setup and running the steps
necessary for the specific test like setting a configuration, connecting and
disconnecting the VPN as well as running checks. For example, a test can run
steps similar to the following:

- start networks and containers
- configure networks and containers, e.g., routing, dns, configs
- run checks without VPN connection on `oc-daemon`
  - connectivity to `web-ext` and `web-int` with ping and curl
  - errors in OC-Daemon logs
- establish VPN connection on `oc-daemon`
- run checks with VPN connection on `oc-daemon`
  - connectivity to `web-ext` and `web-int` with ping and curl
  - errors in OC-Daemon logs
- stop networks and containers

See test cases in `tests.sh` for more info on the specific tests.

## Examples

### Building OC-Daemon for Tests

Building Debian package of regular OC-Daemon version:

```console
$ ./scripts/build-podman.sh
```

Building Debian package of OC-Daemon with race detector and coverage enabled
(recommended for testing):

```console
$ ./scripts/build-podman-test.sh
```

### Running all Tests

Running all tests:

```console
$ ./test/ocserv/tests.sh all
```

### Listing available Tests

Listing all available test:

```console
$ ./test/ocserv/tests.sh list
test_default test_default_ipv6 test_splitrt test_splitrt_ipv6 test_restart
test_reconnect test_disconnect test_occlient_config test_profile_alwayson
test_profile_tnd
```

### Running specific Test

Running specific test `test_default`:

```console
$ ./test/ocserv/tests.sh test_default
```

### Viewing more Test Output

You can view more detailed test output in the log file
`./tests/ocserv/tests.log`, e.g.:

```console
$ less ./tests/ocserv/tests.log
```

### Starting and Stopping Test Setup

Starting the test setup without running any tests, e.g., for debugging:

```console
$ # IPv4 version
$ ./test/ocserv/tests.sh up ipv4
$ # or IPv6 version
$ ./test/ocserv/tests.sh up ipv6
```

Remember to stop the test setup before running tests.

Stopping the test setup:

```console
$ # IPv4 version
$ ./test/ocserv/tests.sh down ipv4
$ # or IPv6 version
$ ./test/ocserv/tests.sh down ipv6
```

[ocserv]: https://ocserv.openconnect-vpn.net/
[rootless]: https://github.com/containers/podman/blob/main/docs/tutorials/rootless_tutorial.md
[subuid]: https://github.com/containers/podman/blob/main/docs/tutorials/rootless_tutorial.md#etcsubuid-and-etcsubgid-configuration
[caddy]: https://caddyserver.com/
