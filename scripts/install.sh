#!/bin/bash

ETCDIR="/var/lib/oc-daemon"
BINDIR="/usr/bin"

# make sure directories exist
sudo mkdir -p $ETCDIR
sudo mkdir -p $BINDIR

# install daemon
go build ./cmd/oc-daemon
sudo rm $BINDIR/oc-daemon 2>/dev/null
sudo cp oc-daemon $BINDIR/oc-daemon

# install xml profile
sudo cp configs/profile.xml $ETCDIR

# install default/example config
sudo cp configs/oc-client.json $ETCDIR

# set user access to config files
sudo chgrp -R dialout $ETCDIR
sudo chmod 664 $ETCDIR/profile.xml
sudo chmod 644 $ETCDIR/oc-client.json

# add user to dialout group
sudo usermod -a -G dialout $USER

# install vpncscript
go build ./cmd/oc-daemon-vpncscript
sudo cp oc-daemon-vpncscript $BINDIR/oc-daemon-vpncscript

# install oc-client
go build ./cmd/oc-client
sudo cp oc-client $BINDIR/oc-client

# install systemd service
sudo cp init/oc-daemon.service /etc/systemd/system
sudo systemctl reenable oc-daemon.service
sudo systemctl restart oc-daemon.service
