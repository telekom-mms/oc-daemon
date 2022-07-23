# Known Issues

This document contains information about known issues.

## Signer not found in OC-Daemon Log

On successful connect, you can see the following error message in the OC-Daemon
log:

```
Server certificate verify failed: signer not found
```

This is OK, as explained here:
https://bugzilla.redhat.com/show_bug.cgi?id=1385286#c2

## Verbose Output of OC-Client

The current version of openconnect in Ubuntu 22.04 seems to ignore the
`--quiet` command line argument and produces rather verbose output.  This could
be related to https://gitlab.com/openconnect/openconnect/-/issues/401 and be
fixed in a later version of openconnect.
