#!/bin/sh -e
# taken from https://github.com/Debian/debhelper/blob/master/dh

UNIT='oc-daemon.service'

case "$1" in
  'remove')
    if [ -d /run/systemd/system ] ; then
      systemctl --system daemon-reload >/dev/null || true
    fi
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