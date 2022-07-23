# Trusted Network Detection

The Trusted Network Detection is implemented with the following mechanisms:

* Watches resolv.conf files in the file system
* Watches routing table changes
* In case of changes
  * Establish HTTPS connection to configured test servers
  * Verifies fingerprint of server's certificate using configured value

The Trusted Network Detection runs inside the oc-daemon. The trusted HTTPS
servers and fingerprints are configured using the values in the XML profile
(AnyConnect Profile) in `/var/lib/oc-daemon/profile.xml`.
