# Tests with ocserv

## Requirements

- podman
  - tested with version 4.3.1 (Debian 12) and 5.4.1
  - [rootless][rootless] with configured [subuid/subgid][subuid] for your user
- podman-compose

[rootless]: https://github.com/containers/podman/blob/main/docs/tutorials/rootless_tutorial.md
[subuid]: https://github.com/containers/podman/blob/main/docs/tutorials/rootless_tutorial.md#etcsubuid-and-etcsubgid-configuration 

## Test Setup

The test setup is shown in the following figure:

```
.........................................................................
: oc-daemon-test-                   :                   oc-daemon-test- :
: ext                               :                               int :
:                                   :                                   :
:   _________________               :                                   :
:  |                 |__            :                                   :
:  | oc-daemon-test- |  |   ________:________       _________________   :
:  | oc-daemon       |  |  |                 |     |                 |  :
:  |_________________|  |__| oc-daemon-test- |_____| oc-daemon-test- |  :
:   _________________   |  | ocserv          |     | web-int         |  :
:  |                 |  |  |_________________|     |_________________|  :
:  | oc-daemon-test- |__|           :                                   :
:  | web-ext         |              :                                   :
:  |_________________|              :                                   :
:                                   :                                   :
:...................................:...................................:
```

The test setup consists of the two networks `oc-daemon-test-ext` and
`oc-daemon-test-int` as well as the four nodes (services, containers)
`oc-daemon-test-oc-daemon`, `oc-daemon-test-web-ext`, `oc-daemon-test-ocserv`
and `oc-daemon-test-web-int`. For brevity, the common prefix `oc-daemon-test-`
in the names is omitted in the following description. `oc-daemon` and `web-ext`
are only in network `ext`. `web-int` is only in network `int`. `ocserv` is in
both networks. `oc-daemon` runs OC-Daemon and acts as VPN client.  `ocserv`
runs [ocserv][ocserv] and acts as VPN server. `web-ext` and `web-int` both run
[Caddy][caddy] and act as Webserver with HTTPS, so they can also be used as TND
servers. `ocserv` connects VPN clients to the network `int` and, thus, to
`web-int`. So, `oc-daemon` can reach `web-int` when it is connected to the VPN
via `ocserv`. Otherwise, it can only reach `web-ext`.

[ocserv]: https://ocserv.openconnect-vpn.net/
[caddy]: https://caddyserver.com/

## Building OC-Daemon for Tests

Building Debian package of regular OC-Daemon version:

```console
$ ./scripts/build-podman.sh
```

Building Debian package of OC-Daemon with race detector and coverage enabled:

```console
$ ./scripts/build-podman-test.sh
```

## Running all Tests

Running all tests, showing all output:

```console
$ ./test/ocserv/tests.sh all
```

Running all tests, piping all output into a file called `log`, not showing
output of other programs:

```console
$ ./test/ocserv/tests.sh all 2>&1 | tee log | grep "^==="
```

## Listing available Tests

Listing all available test:

```console
$ ./test/ocserv/tests.sh list
test_default test_default_ipv6 test_splitrt test_splitrt_ipv6 test_restart
test_reconnect test_disconnect test_occlient_config test_profile_alwayson
test_profile_tnd
```

## Running specific Test

Running specific test `test_default`, showing all output:

```console
$ ./test/ocserv/tests.sh test_default
```

Running specific test `test_default`, piping all output into a file called
`log`, not showing output of other programs:

```console
$ ./test/ocserv/tests.sh test_default 2>&1 | tee log | grep "^==="
```
