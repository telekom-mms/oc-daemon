# Installation

This document contains information about the installation of the `oc-daemon`
and its components.

## Requirements

- Ubuntu 22.04
- nftables and openconnect
```console
$ sudo apt install nftables openconnect -y
```

## Using Debian/Ubuntu package

Download the package from releases page and use the following instructions to install and activate the daemon:

```console
# install package
$ sudo apt install ./oc-daemon.deb

# prepare config
$ sudo mkdir /var/lib/oc-daemon
$ sudo cp /usr/share/doc/oc-daemon/examples/oc-client.json /var/lib/oc-daemon/ # and adjust config parameters
$ sudo cp profile.xml /var/lib/oc-daemon/
$ sudo chgrp -R dialout /var/lib/oc-daemon/
$ sudo chmod 664 /var/lib/oc-daemon/profile.xml
$ sudo chmod 644 /var/lib/oc-daemon/oc-client.json

# setup user to use vpn
$ sudo usermod -a -G dialout $USER

# start daemon
$ sudo systemctl start oc-daemon.service
```

## Using tar.gz archive

Download the archive from releases page and use the following instructions to install and activate the daemon:

```console
# extract archive
$ tar -xf oc-daemon.tar.gz && cd <extracted directory>

# prepare config
$ sudo mkdir /var/lib/oc-daemon
$ sudo cp /example_config.json /var/lib/oc-daemon/oc-client.json # and adjust config parameters
$ sudo cp profile.xml /var/lib/oc-daemon/
$ sudo chgrp -R dialout /var/lib/oc-daemon/
$ sudo chmod 664 /var/lib/oc-daemon/profile.xml
$ sudo chmod 644 /var/lib/oc-daemon/oc-client.json

# setup user to use vpn
$ sudo usermod -a -G dialout $USER
$ sudo cp oc-client oc-daemon oc-daemon-vpncscript /usr/bin/
$ sudo cp oc-daemon.service /lib/systemd/system/

# enable and start daemon
$ sudo systemctl --system daemon-reload
$ sudo systemctl enable oc-daemon.service
$ sudo systemctl start oc-daemon.service
```