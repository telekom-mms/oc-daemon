# Tests with ocserv

## Requirements

- podman
- podman-compose

## Test Setup

The test setup is shown in the following figure:

```
.................................................
: oc-daemon-test_ext    :    oc-daemon-test_int :
:                       :                       :
:   _________           :                       :
:  |         |__        :                       :
:  | deb12   |  |   ____:___        _________   :
:  |_________|  |  |        |      |         |  :
:               |__| ocserv |______| web-int |  :
:   _________   |  |________|      |_________|  :
:  |         |  |       :                       :
:  | web-ext |__|       :                       :
:  |_________|          :                       :
:                       :                       :
:.......................:.......................:
```

The test setup consists of the two networks `oc-daemon-test_ext` and
`oc-daemon-test_int` as well as the four nodes (services, containers) `deb12`,
`web-ext`, `ocserv` and `web-int`. `deb12` and `web-ext` are only in network
`oc-daemon-test_ext`. `web-int` is only in network `oc-daemon-test_int`.
`ocserv` is in both networks. `deb12` runs OC-Daemon and acts as VPN client.
`ocserv` runs ocserv and acts as VPN server. `web-ext` and `web-int` both run
Caddy and act as Webserver with HTTPS, so they can also be used as TND servers.
`ocserv` connects VPN clients to the network `oc-daemon-test_int` and, thus, to
`web-int`. So, `deb12` can reach `web-int` when it is connected to the VPN via
`ocserv`. Otherwise, it can only reach `web-ext`.

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
