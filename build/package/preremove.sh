#!/bin/sh -e
# taken from https://git.launchpad.net/ubuntu/+source/debhelper/tree/autoscripts/preinst-systemd-stop?h=applied/13.6ubuntu1

UNIT='oc-daemon.service'

case "$1" in
  'remove')
    if [ -z "${DPKG_ROOT:-}" ] && [ -d /run/systemd/system ] ; then
      deb-systemd-invoke stop $UNIT >/dev/null || true
    fi
    ;;
esac