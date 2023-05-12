#!/bin/sh -e
# taken from https://github.com/Debian/debhelper/blob/master/dh

UNIT='oc-daemon.service'

case "$1" in
	'configure' |	'abort-upgrade' | 'abort-deconfigure' | 'abort-remove')
    # This will only remove masks created by d-s-h on package removal.
    deb-systemd-helper unmask $UNIT >/dev/null || true

    # was-enabled defaults to true, so new installations run enable.
    if deb-systemd-helper --quiet was-enabled $UNIT; then
      # Enables the unit on first installation, creates new
      # symlinks on upgrades if the unit file has changed.
      deb-systemd-helper enable $UNIT >/dev/null || true
    else
      # Update the statefile to add new symlinks (if any), which need to be
      # cleaned up on purge. Also remove old symlinks.
      deb-systemd-helper update-state $UNIT >/dev/null || true
    fi
    ;;
esac