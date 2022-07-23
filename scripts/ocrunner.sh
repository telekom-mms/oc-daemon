#!/bin/bash

go build -race ./cmd/ocrunner
sudo ./ocrunner \
	-authenticate \
	-connect \
	-disconnect \
	-ca "$PWD/ca.crt" \
	-cert "$PWD/client.crt" \
	-key "$PWD/client.key" \
	-profile "$PWD/profile.xml" \
	-script "$PWD/oc-daemon-vpncscript" \
	-server "My VPN Server Name"
