# Installation

This document contains information about the installation of the `oc-daemon`
and its components.

## Requirements

- Ubuntu 22.04
- golang 1.17 (only for building)
  - `$ sudo apt install golang`
- nftables
  - `$ sudo apt install nftables`

## Quick Start

Note: Please see the [install script](/scripts/install.sh) for the individual
install steps and be sure you are OK with the changes this script makes to your
system before you follow these instructions!

Prepare configuration files:

- Add your XML profile to `configs/profile.xml`
- Edit/adapt/replace the example [oc-client.json](/configs/oc-client.json) to
  match your configuration

Then, you can use the simple [install script](/scripts/install.sh) to build and
install everything:

```console
$ ./scripts/install.sh
```
