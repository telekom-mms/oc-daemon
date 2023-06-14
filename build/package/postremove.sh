#!/bin/sh -e
# taken from https://git.launchpad.net/ubuntu/+source/debhelper/tree/autoscripts/postrm-systemd?h=applied/13.6ubuntu1 and

UNIT='oc-daemon.service'

case "$1" in
  'remove')
    if [ -x "/usr/bin/deb-systemd-helper" ]; then
      deb-systemd-helper mask $UNIT >/dev/null || true
    fi
    ;;

  'purge')
    if [ -x "/usr/bin/deb-systemd-helper" ]; then
      deb-systemd-helper purge $UNIT >/dev/null || true
      deb-systemd-helper unmask $UNIT >/dev/null || true
    fi
    ;;
esac