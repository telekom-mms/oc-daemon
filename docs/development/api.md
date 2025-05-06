# D-Bus API and Socket API

The D-Bus API and Socket API are used to communicate with oc-daemon. The D-Bus
API is used for user interaction and the Socket API is used for communication
with oc-daemon-vpncscript.

## D-Bus API

The D-Bus API relies on D-Bus and is used by oc-client or other client
implementations to communicate with oc-daemon.

```console
$ gdbus introspect --system --dest com.telekom_mms.oc_daemon.Daemon --object-path /com/telekom_mms/oc_daemon/Daemon

node /com/telekom_mms/oc_daemon/Daemon {
  interface com.telekom_mms.oc_daemon.Daemon {
    methods:
      Connect(in  s server,
              in  s cookie,
              in  s host,
              in  s connect_url,
              in  s fingerprint,
              in  s resolve);
      Disconnect();
      DumpState(out s state);
    signals:
    properties:
      readonly u TrustedNetwork = 1;
      readonly u ConnectionState = 1;
      readonly s IP = '';
      readonly s Device = '';
      readonly s Server = '';
      readonly s ServerIP = '';
      readonly x ConnectedAt = 0;
      readonly as Servers = ['VPN Server 1', 'VPN Server 2'];
      readonly u OCRunning = 1;
      readonly u OCPID = 0;
      readonly u TrafPolState = 2;
      readonly as AllowedHosts = ['example.com', 'vpn1.company.net', 'vpn2.company.net', 'tnd1.company.lan', 'tnd2.company.lan', '192.168.3.3', '192.168.4.0/24'];
      readonly u CaptivePortal = 1;
      readonly u TNDState = 2;
      readonly as TNDServers = ['https://tnd1.company.lan:443:ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789', 'https://tnd2.company.lan:443:0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF'];
      readonly s VPNConfig = '';
  };
};
```

### Methods

`Connect()` is used to connect to a VPN server. The parameter `server` is the
name of the VPN server. The remaining parameters are the login information
returned by `openconnect -authenticate`: `cookie` is an access token containing
information for authentication and authorization on the VPN server. `host` is
the VPN server address. `connect_url` is the VPN server URL. `fingerprint` is
the fingerprint of the server's certificate. `resolve` maps the server's host
name to its IP address to bypass DNS resolution.

`Disconnect()` is used to disconnect from the current VPN server.

`DumpState()` is used to retrieve the internal state of oc-daemon. The
parameter `state` is the current state returned by oc-daemon.

### Properties

All properties emit `org.freedesktop.DBus.Properties.PropertiesChanged`
signals.

`TrustedNetwork` indicates whether a trusted network has been detected.

`ConnectionState` indicates whether the VPN connection was established using
the OpenConnect process.

`IP` is the local client's IP address in the VPN.

`Device` is the name of the local client's VPN network device (default:
`oc-daemon-tun0`).

`Server` is the name of the current VPN server.

`ServerIP` is the IP address of the current VPN server.

`ConnectedAt` is the time when the VPN connection was established.

`Servers` is the list of names of available VPN servers.

`OCRunning` indicates whether the OpenConnect process is running.

`OCPID` is the Process ID of the running OpenConnect process.

`TrafPolState` indicates whether Traffic Policing is active.

`AllowedHosts` is the list of allowed hosts configured in Traffic Policing.

`CaptivePortal` indicates whether a captive portal was detected by Traffic
Policing.

`TNDState` indicates whether Trusted Network Detection is active.

`TNDServers` is the list of server URLs with certificate hashes configured in
Trusted Network Detection.

`VPNConfig` is the VPN network configuration. For the go-representation of the
configuration see [VPN Network Configuration](vpn-network-config.md).

## Socket API

The Socket API uses a Unix Domain Socket and is used by oc-daemon-vpncscript to
communicate with oc-daemon.

```
Client    OC-Daemon
  |           |
  |        Listen
  |           |
  | -----> Connect
  | -----> Request
  | <----- Response
  | <---- Disconnect
  |           |
```

* Socket API
  * OC-Daemon runs server
  * OC-Daemon-VPNCScript is client
  * Socket file: `/run/oc-daemon/daemon.sock`
* Request/Response protocol
  * Only 1 request/response per connection
  * Type-Length-Value (TLV) messages with additional token
  * Little-Endian byte order

```
+---------------+-----------------+-----------------+---------------------+
| Type (16 bit) | Length (32 bit) | Token (16 byte) | Value (Length byte) |
+---------------+-----------------+-----------------+---------------------+
```

Message format:

 * Type: 16 Bits
 * Length: 32 Bits
 * Token: 16 Bytes
 * Value: Length Bytes

`Type` is the message type.

Message Types:

* 1: OK (Server Response - OK)
* 2: Error (Server Response - Error)
* 3: VPN Config Update (Client Request - Update VPN Network Configuration)

`Length` is the length of the payload in bytes.

`Token` is a secret shared between oc-daemon and the client. It is used to
verify a legitimate request to change the VPN configuration.

Note: the token is passed from oc-daemon to OpenConnect via an environment
variable. This variable is also passed by OpenConnect to oc-daemon-vpncscript,
that then uses it in its Config Update message.

`Value` is the payload.

Value depends on message type:

* JSON object,
* empty, or
* in case of Error: error message string

### VPN Config Update

* Request
  * Type: VPN Config Update
  * Value: JSON with config update
* Response
  * Type: OK
  * Value: empty

Go-representation of the config update:

```go
type ConfigUpdate struct {
	Reason string
	Config *vpnconfig.Config
}
```

`Reason` is the reason of the update: `connect`, `disconnect`,
`attempt-reconnect` or `reconnect`. `Config` is the VPN network configuration.
For the go-representation of the configuration see [VPN Network
Configuration](vpn-network-config.md).
