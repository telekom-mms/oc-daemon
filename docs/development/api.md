# Daemon API

The daemon API is used by the oc-client and the oc-daemon-vpncscript to
communicate with the oc-daemon.

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

* Unix Socket API
  * OC-Daemon runs server
  * OC-Client and OC-Daemon-VPNCScript are clients
  * Socket file: `/run/oc-daemon/daemon.sock`
* Request/Response protocol
  * Only 1 request/response per connection
  * Type-Length-Value (TLV) messages
  * Little-Endian byte order

```
+---------------+-----------------+---------------------+
| Type (16 bit) | Length (16 bit) | Value (Length byte) |
+---------------+-----------------+---------------------+
```

Message format:

 * Type: 16 Bits
 * Length: 16 Bits
 * Value: Length Bytes

Message Types:

* 1: OK (Server Response - OK)
* 2: Error (Server Response - Error)
* 3: VPN Connect (Client Request - Connect to VPN)
* 4: VPN Disconnect (Client Request - Disconnect from VPN)
* 5: VPN Query (Client Request - Query Status of Daemon/VPN)
* 6: VPN Config Update (Client Request - Update VPN Network Configuration)

Value depends on message type:

* JSON object,
* empty, or
* in case of Error: error message string

## VPN Connect

* Request
  * Type: VPN Connect
  * Value: JSON with login information
* Response
  * Type: OK
  * Value: empty

Go-representation of the login information:

```go
type LoginInfo struct {
	Cookie      string
	Host        string
	Fingerprint string
}
```

The login information represents the information returned by `openconnect
-authenticate`: `Cookie` is an access token containing information for
authentication and authorization on the VPN server. `Host` is the VPN server.
`Fingerprint` is the fingerprint of the server's certificate.

## VPN Disconnect

* Request
  * Type: VPN Disconnect
  * Value: empty
* Response
  * Type: OK
  * Value: empty

## VPN Query

* Request
  * Type: VPN Query
  * Value: empty
* Response
  * Type: OK
  * Value: JSON with VPN/daemon status

Go-representation of the status:

```go
type Status struct {
	TrustedNetwork bool
	Running        bool
	Connected      bool
	Config         *Config
}
```

`TrustedNetwork` indicates if a trusted network has been detected. `Running`
indicates if the openconnect process is running. `Connected` indicates if the
VPN connection was established using the openconnect process. `Config` is the
VPN network configuration. For the go-representation of the configuration see
[VPN Network Configuration](vpn-network-config.md).

## VPN Config Update

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
	Token  string
	Config *Config
}
```

`Reason` is the reason of the update: `connect` or `disconnect`. `Token` is
secret shared between the oc-daemon and the client. It is used to verify a
legitimate request to change the VPN configuration. `Config` is the VPN network
configuration. For the go-representation of the configuration see [VPN Network
Configuration](vpn-network-config.md).

Note: the token is passed from the oc-daemon to openconnect via an environment
variable. This variable is also passed by openconnect to oc-daemon-vpncscript,
that then uses it in its Config Update request.
