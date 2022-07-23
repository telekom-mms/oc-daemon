# Overview

The OpenConnect Daemon components and their interaction are described in the
following sections.

## Components

User-facing components
* OpenConnect Daemon (`oc-daemon`)
* OpenConnect Client (`oc-client`)

Internally-used components and dependencies
* OpenConnect Daemon vpnc-script (`oc-daemon-vpncscript`)
* Official OpenConnect client (`openconnect`)
* IP tool (`ip`)
* Nftables tool (`nft`)
* Resolvectl tool (`resolvectl`)

In the following sections the components are described in more detail and the
interaction of the components are shown for connecting to the VPN and
disconnecting from the VPN.

### OC-Daemon

* started during system boot as systemd service
* root privileges
* connects to VPN server
  * using login info from oc-client
  * runs `openconnect --script oc-daemon-vpncscript`
* configures VPN networking
  * `ip`, `nft`, `resolvectl`

### OC-Client

The user interacts with the oc-daemon via oc-client.

* run by user
* user's privileges
* triggers connecting to VPN
  * authenticates user with VPN server
  * runs `openconnect --authenticate`
  * parses login info from openconnect
  * sends login info to oc-daemon
* triggers disconnecting from VPN
* lists VPN servers in XML Profile
* checks status of oc-daemon
  * Trusted Network Detection
  * VPN connection status

### OpenConnect

* run by oc-daemon and oc-client
* interacts with daemon via oc-daemon-vpncscript

### OC-Daemon-VPNCScript

* run by openconnect
* parses configuration in environment passed by openconnect
* sends configuration to oc-daemon

### Connecting to the VPN

```
User    OC-Client      OC-Daemon     OpenConnect        OC-Daemon-VPNCScript
 |          |              |              |                      |
 | -------> | --------> Status            |                      |
 |          | <--------    |              |                      |
 |          | ---------------------> Authenticate                |
 |          | <---------------------      |                      |
 |          | --------> Connect -----> Connect ----------> Config Update
 |          |           Network <-------------------------       |
 |          |           Config            |                      |
 |          |              |              |                      |
```

1. user runs oc-client with "connect" command
2. oc-client authenticates user with VPN server
   1. oc client retrieves status from oc-daemon
      1. abort, if trusted network
      2. abort, if VPN is already running
   2. runs `openconnect -authenticate`
   3. parses login info from openconnect
   4. sends "connect" message with login info to oc-daemon
3. oc-daemon connects to VPN server
   1. receives login info in oc-client "connect" message
   2. runs `openconnect -script oc-daemon-vpncscript`
4. openconnect starts running
   1. establishes tunnel
   2. runs `oc-daemon-vpncscript` with VPN network settings
5. oc-daemon-vpncscript passes settings to oc-daemon
   1. parses VPN network settings
   2. sends "config update" message with reason "connect" and VPN network
      settings to oc-daemon
6. oc-daemon configures VPN network
   1. receives VPN network settings in "config update" message from
      oc-daemon-vpncscript
   2. configures VPN network based on settings

### Disconnecting from the VPN

```
User    OC-Client      OC-Daemon     OpenConnect        OC-Daemon-VPNCScript
 |          |              |              |                      |
 | -------> | --------> Status            |                      |
 |          | <--------    |              |                      |
 |          | ------> Disconnect -----> Stop ------------> Config Update
 |          |           Network <-------------------------       |
 |          |           Config            |                      |
 |          |              |              |                      |
```

1. user runs oc-client with "disconnect" command
2. oc-client sends "disconnect" request
   1. oc client retrieves status from oc-daemon
      1. abort, if VPN is not running
   2. sends "disconnect" message to oc-daemon
3. oc-daemon disconnects VPN
   1. receives "disconnect" message from oc-client
   2. terminates running openconnect process
4. openconect stops running
   1. disconnects tunnel
   2. runs `oc-daemon-vpnscript`
5. oc-daemon-vpncscript passes settings to oc-daemon
   1. parses VPN network settings (empty)
   2. sends "config update" message with reason "disconnect" to oc-daemon
6. oc-daemon removes VPN network configuration
   1. receives "config update" message from oc-daemon-vpncscript
   2. removes VPN network settings

If oc-daemon detects, that the openconnect process terminated (abnormally),
step 6 is also triggered to make sure, there is no invalid VPN configuration
active.
