# Usage

This document contains the usage information for the `oc-client` tool, that you
use to interact with the `oc-daemon`. Usually, it is the only tool you need to
connect to your VPN. But this document also contains usage information for the
`oc-daemon` itself and the helper tool `oc-daemon-vpncscript` for the sake of
completeness and rare cases you might need it.

## oc-client

You can use `oc-client` to interact with the `oc-daemon` in order to connect to
or disconnect from your VPN. You can run `oc-client` with the following command
line arguments:

```
Usage:
  oc-client [options] [command]

Options:
  -ca file
        set additional CA certificate file
  -cert file
        set client certificate file or PKCS11 URI
  -config file
        set config file
  -group usergroup
        set usergroup
  -key file
        set client key file or PKCS11 URI
  -profile file
        set XML profile file
  -server address
        set server address
  -system-settings
        use system settings instead of user configuration
  -user username
        set username
  -user-cert file
        set user certificate file or PKCS11 URI.
        Note: requires OpenConnect v9.00 or higher
  -user-key file
        set user key file or PKCS11 URI.
        Note: requires OpenConnect v9.00 or higher
  -version
        print version

Commands:
  connect
        connect to the VPN (default)
  disconnect
        disconnect from the VPN
  reconnect
        reconnect to the VPN
  list
        list VPN servers in XML Profile
  status
        show VPN status
  monitor
        monitor VPN status updates
  save
        save current settings to user configuration

Examples:
  oc-client connect
  oc-client disconnect
  oc-client reconnect
  oc-client status
  oc-client list
  oc-client -server "My SSL VPN Server" connect
  oc-client -server "My SSL VPN Server" save
  oc-client -user exampleuser connect
  oc-client -user $USER save
  oc-client -system-settings save
```

### Configuration

The user-specific configuration is stored in the JSON file
`~/.config/oc-daemon/oc-client.json`. The system-wide configuration is stored
in the JSON file `/var/lib/oc-daemon/oc-client.json`. If the user-specific
configuration file exists, it is preferred. Otherwise, the system-wide
configuration is used. You can override this priority with the `oc-client`
option `-system-settings` and load the system-wide configuration instead of the
user-specific configuration.

You can override settings in the configuration files with the `oc-client`
command line options. You can use the `oc-client` command `save` to save the
currently loaded settings into your user-specific configuration.

The JSON format of the system-wide and user-specific configuration is
identical. Your system-wide or user-specific configuration file could, for
example, look like this:

```json
{
    "ClientCertificate": "/path/to/mycert",
    "ClientKey": "/path/to/mykey",
    "CACertificate": "",
    "VPNServer": "My VPN Server"
}
```

Since `openconnect` supports file names and PKCS11 URIs, you can also use
PKCS11 URIs for your certificate and key.

### Connecting

You can connect to the VPN with your current settings with:

```console
$ oc-client
```

or

```console
$ oc-client connect
```

### Disconnecting

You can disconnect the VPN with:

```console
$ oc-client disconnect
```

### Reconnecting

You can disconnect and reconnect the VPN with:

```console
$ oc-client reconnect
```

### Showing Status

You can show the current status with:

```console
$ oc-client status
```

### Monitoring Status

You can monitor the current status with:

```console
$ oc-client monitor
```

### Listing Servers

You can list VPN servers in your XML profile (`/var/lib/oc-daemon/profile.xml`)
with:

```console
$ oc-client list
```

## oc-daemon

Usually, `oc-daemon` runs as a systemd service and you interact with it using
`oc-client`. As a user, you should not have to interact with it directly. If
you have to run `oc-daemon` manually, you can run it with the following command
line arguments:

```
Usage of oc-daemon:
  -config file
        set config file (default "/var/lib/oc-daemon/oc-daemon.json")
  -verbose
        enable verbose output
  -version
        print version
```

## oc-daemon-vpncscript

Usually, `oc-daemon-vpncscript` is used internally by `oc-daemon` to pass the
VPN configuration from `openconnect` to `oc-daemon`. As a user, you should not
have to interact with `oc-daemon-vpncscript`. In case you have to, You can run
`oc-daemon-vpncscript` with the following command line arguments:

```
Usage of oc-daemon-vpncscript:
  -verbose
        enable verbose output
  -version
        print version
```
