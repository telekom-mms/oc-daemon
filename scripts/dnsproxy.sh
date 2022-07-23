#!/bin/bash

go run ./cmd/dnsproxy \
	-remotes ".:127.0.0.53:53,google.com.:127.0.0.53:53" \
	-watches "google.com.,google.de."
