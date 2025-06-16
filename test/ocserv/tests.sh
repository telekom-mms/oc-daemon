#!/bin/bash

# executables
PODMAN="podman"
PODMAN_COMPOSE="podman-compose"
GREP="grep"
DATE="date --rfc-3339=seconds"
TIMESTAMP="date +%s"

# networks
NETWORK_EXT_NAME="oc-daemon-test-ext"
NETWORK_EXT_PORTAL_NAME="oc-daemon-test-ext-portal"
NETWORK_INT_NAME="oc-daemon-test-int"

# containers
OCSERV_NAME="oc-daemon-test-ocserv"
OC_DAEMON_NAME="oc-daemon-test-oc-daemon"
PORTAL_NAME="oc-daemon-test-portal"
CONNCHECK_NAME="oc-daemon-test-conncheck"
WEB_EXT_NAME="oc-daemon-test-web-ext"
WEB_INT_NAME="oc-daemon-test-web-int"

# log file
LOG_FILE="test/ocserv/tests.log"

###############################################################################
###                                 Helpers                                 ###
###############################################################################

# number of tests, OKs and fails
TESTS=0
OKS=0
FAILS=0

# print test output
out() {
	echo "=== $($DATE): $1"
}

# get container settings
get_settings() {
	# ocserv
	OCSERV_PID=$($PODMAN inspect --format "{{.State.Pid}}" $OCSERV_NAME)
	OCSERV_IPv4_EXT=$($PODMAN inspect \
		--format "{{range \$k,\$v := .NetworkSettings.Networks}}{{if eq \$k \"$NETWORK_EXT_NAME\"}}{{\$v.IPAddress}}{{end}}{{end}}" \
		$OCSERV_NAME)
	OCSERV_IPv6_EXT=$($PODMAN inspect \
		--format "{{range \$k,\$v := .NetworkSettings.Networks}}{{if eq \$k \"$NETWORK_EXT_NAME\"}}{{\$v.GlobalIPv6Address}}{{end}}{{end}}" \
		$OCSERV_NAME)
	OCSERV_IPv4_INT=$($PODMAN inspect \
		--format "{{range \$k,\$v := .NetworkSettings.Networks}}{{if eq \$k \"$NETWORK_INT_NAME\"}}{{\$v.IPAddress}}{{end}}{{end}}" \
		$OCSERV_NAME)
	OCSERV_IPv6_INT=$($PODMAN inspect \
		--format "{{range \$k,\$v := .NetworkSettings.Networks}}{{if eq \$k \"$NETWORK_INT_NAME\"}}{{\$v.GlobalIPv6Address}}{{end}}{{end}}" \
		$OCSERV_NAME)

	# oc-daemon
	OC_DAEMON_PID=$($PODMAN inspect --format "{{.State.Pid}}" $OC_DAEMON_NAME)
	OC_DAEMON_IPv4_EXT_PORTAL=$($PODMAN inspect \
		--format "{{range \$k,\$v := .NetworkSettings.Networks}}{{if eq \$k \"$NETWORK_EXT_PORTAL_NAME\"}}{{\$v.IPAddress}}{{end}}{{end}}" \
		$OC_DAEMON_NAME)
	OC_DAEMON_IPv6_EXT_PORTAL=$($PODMAN inspect \
		--format "{{range \$k,\$v := .NetworkSettings.Networks}}{{if eq \$k \"$NETWORK_EXT_PORTAL_NAME\"}}{{\$v.GlobalIPv6Address}}{{end}}{{end}}" \
		$OC_DAEMON_NAME)

	# portal
	PORTAL_PID=$($PODMAN inspect --format "{{.State.Pid}}" $PORTAL_NAME)
	PORTAL_IPv4_EXT_PORTAL=$($PODMAN inspect \
		--format "{{range \$k,\$v := .NetworkSettings.Networks}}{{if eq \$k \"$NETWORK_EXT_PORTAL_NAME\"}}{{\$v.IPAddress}}{{end}}{{end}}" \
		$PORTAL_NAME)
	PORTAL_IPv6_EXT_PORTAL=$($PODMAN inspect \
		--format "{{range \$k,\$v := .NetworkSettings.Networks}}{{if eq \$k \"$NETWORK_EXT_PORTAL_NAME\"}}{{\$v.GlobalIPv6Address}}{{end}}{{end}}" \
		$PORTAL_NAME)
	PORTAL_IPv4_EXT=$($PODMAN inspect \
		--format "{{range \$k,\$v := .NetworkSettings.Networks}}{{if eq \$k \"$NETWORK_EXT_NAME\"}}{{\$v.IPAddress}}{{end}}{{end}}" \
		$PORTAL_NAME)
	PORTAL_IPv6_EXT=$($PODMAN inspect \
		--format "{{range \$k,\$v := .NetworkSettings.Networks}}{{if eq \$k \"$NETWORK_EXT_NAME\"}}{{\$v.GlobalIPv6Address}}{{end}}{{end}}" \
		$PORTAL_NAME)

	# web-ext
	WEB_EXT_PID=$($PODMAN inspect --format "{{.State.Pid}}" $WEB_EXT_NAME)
	WEB_EXT_IPv4_EXT=$($PODMAN inspect \
		--format "{{range \$k,\$v := .NetworkSettings.Networks}}{{if eq \$k \"$NETWORK_EXT_NAME\"}}{{\$v.IPAddress}}{{end}}{{end}}" \
		$WEB_EXT_NAME)
	WEB_EXT_IPv6_EXT=$($PODMAN inspect \
		--format "{{range \$k,\$v := .NetworkSettings.Networks}}{{if eq \$k \"$NETWORK_EXT_NAME\"}}{{\$v.GlobalIPv6Address}}{{end}}{{end}}" \
		$WEB_EXT_NAME)

	# web-int
	WEB_INT_PID=$($PODMAN inspect --format "{{.State.Pid}}" $WEB_INT_NAME)
	WEB_INT_IPv4_INT=$($PODMAN inspect \
		--format "{{range \$k,\$v := .NetworkSettings.Networks}}{{if eq \$k \"$NETWORK_INT_NAME\"}}{{\$v.IPAddress}}{{end}}{{end}}" \
		$WEB_INT_NAME)
	WEB_INT_IPv6_INT=$($PODMAN inspect \
		--format "{{range \$k,\$v := .NetworkSettings.Networks}}{{if eq \$k \"$NETWORK_INT_NAME\"}}{{\$v.GlobalIPv6Address}}{{end}}{{end}}" \
		$WEB_INT_NAME)

	# conncheck
	CONNCHECK_PID=$($PODMAN inspect --format "{{.State.Pid}}" $CONNCHECK_NAME)
	CONNCHECK_IPv4_EXT=$($PODMAN inspect \
		--format "{{range \$k,\$v := .NetworkSettings.Networks}}{{if eq \$k \"$NETWORK_EXT_NAME\"}}{{\$v.IPAddress}}{{end}}{{end}}" \
		$CONNCHECK_NAME)
	CONNCHECK_IPv6_EXT=$($PODMAN inspect \
		--format "{{range \$k,\$v := .NetworkSettings.Networks}}{{if eq \$k \"$NETWORK_EXT_NAME\"}}{{\$v.GlobalIPv6Address}}{{end}}{{end}}" \
		$CONNCHECK_NAME)

	# print infos about containers
	out "Networks:"
	out "- $NETWORK_EXT_NAME"
	out "- $NETWORK_EXT_PORTAL_NAME"
	out "- $NETWORK_INT_NAME"
	out "Containers:"
	out "- $OCSERV_NAME:"
	out "    PID: $OCSERV_PID"
	out "    IPv4_EXT: $OCSERV_IPv4_EXT"
	out "    IPv6_EXT: $OCSERV_IPv6_EXT"
	out "    IPv4_INT: $OCSERV_IPv4_INT"
	out "    IPv6_INT: $OCSERV_IPv6_INT"
	out "- $OC_DAEMON_NAME:"
	out "    PID: $OC_DAEMON_PID"
	out "    IPv4_EXT_PORTAL: $OC_DAEMON_IPv4_EXT_PORTAL"
	out "    IPv6_EXT_PORTAL: $OC_DAEMON_IPv6_EXT_PORTAL"
	out "- $PORTAL_NAME:"
	out "    PID: $PORTAL_PID"
	out "    IPv4_EXT_PORTAL: $PORTAL_IPv4_EXT_PORTAL"
	out "    IPv6_EXT_PORTAL: $PORTAL_IPv6_EXT_PORTAL"
	out "    IPv4_EXT: $PORTAL_IPv4_EXT"
	out "    IPv6_EXT: $PORTAL_IPv6_EXT"
	out "- $WEB_EXT_NAME:"
	out "    PID: $WEB_EXT_PID"
	out "    IPv4_EXT: $WEB_EXT_IPv4_EXT"
	out "    IPv6_EXT: $WEB_EXT_IPv6_EXT"
	out "- $WEB_INT_NAME:"
	out "    PID: $WEB_INT_PID"
	out "    IPv4_INT: $WEB_INT_IPv4_INT"
	out "    IPv6_INT: $WEB_INT_IPv6_INT"
	out "- $CONNCHECK_NAME:"
	out "    PID: $CONNCHECK_PID"
	out "    IPv4_EXT: $CONNCHECK_IPv4_EXT"
	out "    IPv6_EXT: $CONNCHECK_IPv6_EXT"
}

# configure routing
configure_routing() {
	out "Configuring routing in containers..."
	# web-int
	$PODMAN exec "$WEB_INT_NAME" ip -4 route add default via "$OCSERV_IPv4_INT"
	$PODMAN exec "$WEB_INT_NAME" ip -4 route show
	if [ -n "$WEB_INT_IPv6_INT" ]; then
		$PODMAN exec "$WEB_INT_NAME" ip -6 route add default via "$OCSERV_IPv6_INT"
		$PODMAN exec "$WEB_INT_NAME" ip -6 route show
	fi

	# oc-daemon
	$PODMAN exec "$OC_DAEMON_NAME" ip -4 route add default via "$PORTAL_IPv4_EXT_PORTAL"
	$PODMAN exec "$OC_DAEMON_NAME" ip -4 route show
	if [ -n "$OC_DAEMON_IPv6_EXT_PORTAL" ]; then
		$PODMAN exec "$OC_DAEMON_NAME" ip -6 route add default via "$PORTAL_IPv6_EXT_PORTAL"
		$PODMAN exec "$OC_DAEMON_NAME" ip -6 route show
	fi

	# web-ext
	$PODMAN exec "$WEB_EXT_NAME" ip -4 route add default via "$PORTAL_IPv4_EXT"
	$PODMAN exec "$WEB_EXT_NAME" ip -4 route show
	if [ -n "$WEB_EXT_IPv6_EXT" ]; then
		$PODMAN exec "$WEB_EXT_NAME" ip -6 route add default via "$PORTAL_IPv6_EXT"
		$PODMAN exec "$WEB_EXT_NAME" ip -6 route show
	fi

	# ocserv
	$PODMAN exec "$OCSERV_NAME" ip -4 route add default via "$PORTAL_IPv4_EXT"
	$PODMAN exec "$OCSERV_NAME" ip -4 route show
	if [ -n "$OCSERV_IPv6_EXT" ]; then
		$PODMAN exec "$OCSERV_NAME" ip -6 route add default via "$PORTAL_IPv6_EXT"
		$PODMAN exec "$OCSERV_NAME" ip -6 route show
	fi

	# conncheck
	$PODMAN exec "$CONNCHECK_NAME" ip -4 route add default via "$PORTAL_IPv4_EXT"
	$PODMAN exec "$CONNCHECK_NAME" ip -4 route show
	if [ -n "$CONNCHECK_IPv6_EXT" ]; then
		$PODMAN exec "$CONNCHECK_NAME" ip -6 route add default via "$PORTAL_IPv6_EXT"
		$PODMAN exec "$CONNCHECK_NAME" ip -6 route show
	fi
}

# configure dns on oc-daemon: systemd-resolved and /etc/hosts.
configure_dns() {
	out "Configuring systemd-resolved in oc-daemon container..."

	# get dns server ip
	# assume gateway address is dns address
	local oc_daemon_dns_ext
	oc_daemon_dns_ext=$($PODMAN inspect \
		--format "{{range \$k,\$v := .NetworkSettings.Networks}}{{if eq \$k \"$NETWORK_EXT_PORTAL_NAME\"}}{{\$v.Gateway}}{{end}}{{end}}" \
		$OC_DAEMON_NAME)

	# configure resolved
	$PODMAN exec "$OC_DAEMON_NAME" ln -sf ../run/systemd/resolve/stub-resolv.conf /etc/resolv.conf
	$PODMAN exec "$OC_DAEMON_NAME" mkdir /etc/systemd/resolved.conf.d
	local config="[Resolve]
DNS=$oc_daemon_dns_ext
Domains=dns.podman"
	$PODMAN exec "$OC_DAEMON_NAME" sh -c "echo '$config' > /etc/systemd/resolved.conf.d/podman.conf"
	$PODMAN exec "$OC_DAEMON_NAME" systemctl restart systemd-resolved

	# check resolved
	$PODMAN exec "$OC_DAEMON_NAME" resolvectl status

	# set names in /etc/hosts on oc-daemon
	$PODMAN exec "$OC_DAEMON_NAME" sh -c "echo '$OCSERV_IPv4_EXT $OCSERV_NAME' >> /etc/hosts"
	if [ -n "$OCSERV_IPv6_EXT" ]; then
		$PODMAN exec "$OC_DAEMON_NAME" sh -c "echo '$OCSERV_IPv6_EXT $OCSERV_NAME' >> /etc/hosts"
	fi

	$PODMAN exec "$OC_DAEMON_NAME" sh -c "echo '$CONNCHECK_IPv4_EXT $CONNCHECK_NAME' >> /etc/hosts"
	if [ -n "$CONNCHECK_IPv6_EXT" ]; then
		$PODMAN exec "$OC_DAEMON_NAME" sh -c "echo '$CONNCHECK_IPv6_EXT $CONNCHECK_NAME' >> /etc/hosts"
	fi

	$PODMAN exec "$OC_DAEMON_NAME" sh -c "echo '$WEB_INT_IPv4_INT $WEB_INT_NAME' >> /etc/hosts"
	$PODMAN exec "$OC_DAEMON_NAME" sh -c "echo '$WEB_EXT_IPv4_EXT $WEB_EXT_NAME' >> /etc/hosts"

	# check /etc/hosts
	$PODMAN exec "$OC_DAEMON_NAME" cat /etc/hosts
}

# start networks and containers, common parts
start_containers_common() {
	local file=$1

	out "Starting networks and containers..."
	COMPOSE_PARALLEL_LIMIT=1 $PODMAN_COMPOSE \
		--file "$file" \
		up \
		--build \
		--detach
	get_settings
	configure_routing
	configure_dns
}

# start networks and containers
start_containers() {
	start_containers_common "$PWD/test/ocserv/podman/compose.yml"
}

# start networks and containers, ipv6 version
start_containers_ipv6() {
	start_containers_common "$PWD/test/ocserv/podman/compose-ipv6.yml"
}

# shut down networks and containers, common parts
stop_containers_common() {
	local file=$1

	out "Stopping networks and containers..."
	$PODMAN_COMPOSE --file "$file" down
}

# shut down networks and containers
stop_containers() {
	stop_containers_common "$PWD/test/ocserv/podman/compose.yml"
}

# shut down networks and containers, ipv6_version
stop_containers_ipv6() {
	stop_containers_common "$PWD/test/ocserv/podman/compose-ipv6.yml"
}

# connect vpn, default settings
connect_vpn_default() {
	out "Connecting to VPN..."
	$PODMAN exec "$OC_DAEMON_NAME" oc-client \
		connect
}

# connect vpn, with settings from command line
connect_vpn_cmdline() {
	out "Connecting to VPN..."
	$PODMAN exec "$OC_DAEMON_NAME" oc-client \
		-ca ca-cert.pem \
		-key client-key.pem \
		-cert client-cert.pem \
		-server "$OCSERV_NAME" \
		connect
}

# disconnect vpn, default settings
disconnect_vpn_default() {
	out "Disconnecting from VPN..."
	$PODMAN exec "$OC_DAEMON_NAME" oc-client \
		disconnect
}

# disconnect vpn, settings from command line
disconnect_vpn_cmdline() {
	out "Disconnecting from VPN..."
	$PODMAN exec "$OC_DAEMON_NAME" oc-client \
		-ca ca-cert.pem \
		-key client-key.pem \
		-cert client-cert.pem \
		-server "$OCSERV_NAME" \
		disconnect
}

# reconnect vpn, default settings
reconnect_vpn_default() {
	out "Reconnecting to VPN..."
	$PODMAN exec "$OC_DAEMON_NAME" oc-client \
		reconnect
}

# reconnect vpn, settings from command line
reconnect_vpn_cmdline() {
	out "Reconnecting to VPN..."
	$PODMAN exec "$OC_DAEMON_NAME" oc-client \
		-ca ca-cert.pem \
		-key client-key.pem \
		-cert client-cert.pem \
		-server "$OCSERV_NAME" \
		reconnect
}

# save oc-client user settings
save_oc_client_user_settings() {
	out "Saving oc-client user settings..."
	$PODMAN exec "$OC_DAEMON_NAME" oc-client \
		-ca ca-cert.pem \
		-key client-key.pem \
		-cert client-cert.pem \
		-server "$OCSERV_NAME" \
		save

	# check
	$PODMAN exec "$OC_DAEMON_NAME" sh -c "cat ~/.config/oc-daemon/oc-client.json"
}

# save oc-client system settings
save_oc_client_system_settings() {
	out "Saving oc-client system settings..."
	local config="{
	\"ClientCertificate\": \"/client-cert.pem\",
	\"ClientKey\": \"/client-key.pem\",
	\"CACertificate\": \"/ca-cert.pem\",
	\"VPNServer\": \"$OCSERV_NAME\"
}"
	$PODMAN exec "$OC_DAEMON_NAME" sh -c "echo '$config' > /var/lib/oc-daemon/oc-client.json"

	# check
	$PODMAN exec "$OC_DAEMON_NAME" cat /var/lib/oc-daemon/oc-client.json
}

# get profile with WEB_SERVER and CERT_HASH set for web
get_profile() {
	local web=$1

	# read profile
	local profile
	profile=$(<"$PWD/test/ocserv/oc-daemon/profile.xml")

	# read fingerprint
	local sum
	sum=$($PODMAN exec "$web" cat "/web-cert.sum")

	# set finterprint in profile
	profile=${profile/CERT_HASH/"$sum"}

	# set https server in profile
	profile=${profile/WEB_SERVER/"$web"}

	echo "$profile"
}

# set xml profile on oc-daemon
set_profile_oc_daemon() {
	out "Setting XML profile on oc-daemon..."
	local web=$1

	# get profile
	local profile
	profile=$(get_profile "$web")

	# set profile
	$PODMAN exec "$OC_DAEMON_NAME" sh -c "echo '$profile' > /var/lib/oc-daemon/profile.xml"

	# check
	$PODMAN exec "$OC_DAEMON_NAME" cat /var/lib/oc-daemon/profile.xml
}

# set xml profile on ocserv
set_profile_ocserv() {
	out "Setting XML profile on ocserv..."
	local profile=$1

	# set profile
	$PODMAN exec "$OCSERV_NAME" sh -c "echo '$profile' > /var/lib/ocserv/profile.xml"

	# check
	$PODMAN exec "$OCSERV_NAME" cat /var/lib/ocserv/profile.xml
}

# compare profile with profile.xml on oc-daemon.
# returns error if profiles are not equal
is_equal_profile_oc_daemon() {
	local profile=$1

	local profile_ocd
	profile_ocd=$($PODMAN exec "$OC_DAEMON_NAME" cat /var/lib/oc-daemon/profile.xml)
	if [ "$profile" != "$profile_ocd" ]; then
		out "profile:"
		out "$profile"
		out "profile on oc-daemon:"
		out "$profile_ocd"
		return 1
	fi
}

# set oc-daemon config, with cpd.
set_oc_daemon_config_cpd() {
	local config="{
	\"CPD\": {
		\"Host\": \"$CONNCHECK_NAME\",
		\"ProbeTimer\": 5000000000,
		\"ProbeTimerDetected\": 2000000000
	}
}"
	out "Setting oc-daemon.json on oc-daemon..."
	$PODMAN exec "$OC_DAEMON_NAME" sh -c "echo '$config' > /var/lib/oc-daemon/oc-daemon.json"
	$PODMAN exec "$OC_DAEMON_NAME" cat /var/lib/oc-daemon/oc-daemon.json
}

# enable captive portal, emulate logged out state
enable_captive_portal() {
	local ruleset="table inet portal {
	chain prerouting {
		type nat hook prerouting priority dstnat; policy accept;
		tcp dport 80 redirect to :80
	}
	chain forward {
		type filter hook forward priority filter; policy drop;
	}
}"
	out "Enabling captive portal..."
	$PODMAN exec "$PORTAL_NAME" nft -f - "$ruleset"
	$PODMAN exec "$PORTAL_NAME" nft list ruleset
}

# disable captive portal, emulate logged in state
disable_captive_portal() {
	out "Disabling captive portal..."
	$PODMAN exec "$PORTAL_NAME" nft flush ruleset
	$PODMAN exec "$PORTAL_NAME" nft list ruleset
}

# returns whether captive portal is detected on oc-daemon
is_detected_captive_portal() {
	local status
	status=$($PODMAN exec "$OC_DAEMON_NAME" oc-client status -verbose)
	echo "$status"
	echo "$status" | $GREP "Captive Portal:   detected"
}

# ping external web server
ping_ext() {
	out "Pinging external web server"
	$PODMAN exec "$OC_DAEMON_NAME" ping -c 3 "$WEB_EXT_IPv4_EXT"
}

# ping internal web server
ping_int() {
	out "Pinging internal web server"
	$PODMAN exec "$OC_DAEMON_NAME" ping -c 3 "$WEB_INT_IPv4_INT"
}

# curl external web server
curl_ext() {
	out "HTTP GET external web server"
	$PODMAN exec "$OC_DAEMON_NAME" curl -v \
		--silent \
		--connect-timeout 3 \
		"https://$WEB_EXT_NAME" \
		--resolve "$WEB_EXT_NAME:443:$WEB_EXT_IPv4_EXT" \
		--cacert /ca-cert.pem
}

# curl internal web server
curl_int() {
	out "HTTP GET internal web server"
	$PODMAN exec "$OC_DAEMON_NAME" curl -v \
		--silent \
		--connect-timeout 3 \
		"https://$WEB_INT_NAME" \
		--resolve "$WEB_INT_NAME:443:$WEB_INT_IPv4_INT" \
		--cacert /ca-cert.pem
}

# run command in first argument and check whether return code is an error
expect_err() {
	if "$@"; then
		out "FAIL: Line ${LINENO}/${BASH_LINENO[*]}: $1 should return error"
		((FAILS++))
	else
		out "OK: Line ${LINENO}/${BASH_LINENO[*]}: $1 returned error as expected"
		((OKS++))
	fi
}

# run command in first argument and check whether return code is OK/no error
expect_ok() {
	if ! "$@"; then
		out "FAIL: Line ${LINENO}/${BASH_LINENO[*]}: $1 should not return error"
		((FAILS++))
	else
		out "OK: Line ${LINENO}/${BASH_LINENO[*]}: $1 returned no error as expected"
		((OKS++))
	fi
}

# set ocserv config
set_ocserv_config() {
	out "Setting new ocserv config..."
	local config=$1

	# write it to ocserv
	$PODMAN exec "$OCSERV_NAME" sh -c "echo '$config' > /etc/ocserv/ocserv.conf"

	# reload ocserv
	$PODMAN exec "$OCSERV_NAME" occtl reload

	# wait
	sleep 1

	# check
	$PODMAN exec "$OCSERV_NAME" cat /etc/ocserv/ocserv.conf
}

# show oc-client status on oc-daemon
show_oc_client_status() {
	$PODMAN exec "$OC_DAEMON_NAME" oc-client \
		-ca ca-cert.pem \
		-key client-key.pem \
		-cert client-cert.pem \
		-server "$OCSERV_NAME" \
		status \
		-verbose
}

# show routes on oc-daemon
show_routes() {
	$PODMAN exec "$OC_DAEMON_NAME" ip -4 route show table all
	$PODMAN exec "$OC_DAEMON_NAME" ip -6 route show table all
}

# show nftables ruleset on oc-daemon
show_nft_ruleset() {
	$PODMAN exec "$OC_DAEMON_NAME" nft list ruleset
}

# restart oc-daemon on oc-daemon
restart_oc_daemon() {
	$PODMAN exec "$OC_DAEMON_NAME" systemctl restart oc-daemon.service
}

# stop oc-daemon on oc-daemon
stop_oc_daemon() {
	$PODMAN exec "$OC_DAEMON_NAME" systemctl stop oc-daemon.service
}

# get oc-daemon log on oc-daemon
get_oc_daemon_log() {
	$PODMAN exec "$OC_DAEMON_NAME" journalctl -u oc-daemon.service
}

# get errors in oc-daemon log on oc-daemon.
# returns error if an error is found in log.
# ignores some pre-defined errors, see ignore_errors
get_log_errors() {
	local ignore_errors=(
		'msg="Could not read XML profile" error="open /var/lib/oc-daemon/profile.xml: no such file or directory"'
		'stderr="sysctl: permission denied on key \\"net.ipv4.conf.all.src_valid_mark\\"'
	)

	local log
	log=$(get_oc_daemon_log)

	local errors
	errors=$($GREP "level=error" <<< "$log")

	for i in "${ignore_errors[@]}"; do
		errors=$($GREP -v "$i" <<< "$errors")
	done

	if [ -n "${errors}" ];then
		out "$errors"
		return 1
	fi
}

# directories for GOCOVER files in container and on host
GOCOVERDIR="/gocover"
HOST_GOCOVERDIR="$PWD/test/ocserv/gocover/$($TIMESTAMP)"

# save GOCOVER directory
save_gocover_dir() {
	local dir="${HOST_GOCOVERDIR}/${TESTS}"
	mkdir -p "$dir"
	$PODMAN cp "${OC_DAEMON_NAME}:${GOCOVERDIR}/." "$dir"
}

# show GOCOVER percentage
show_gocover_percent() {
	local covdirs=""
	for i in $(seq 1 "$NUM_TESTS"); do
		if [ "$i" -eq 1 ]; then
			covdirs=$HOST_GOCOVERDIR/$i
		else
			covdirs=$covdirs,$HOST_GOCOVERDIR/$i
		fi
	done
	out "go tool covdata percent -i $covdirs"
	while read -r line
	do
		    out "$line"
	done < <(go tool covdata percent -i "$covdirs")
}

# show test summary
show_summary() {
	out "==============================="
	out "=== Cumulative Test Summary ==="
	out "==============================="
	out "Tests: $TESTS"
	out "OKs: $OKS"
	out "FAILs: $FAILS"
	out "==============================="
}

###############################################################################
###                               Test Cases                                ###
###############################################################################

# run tests and expect external server OK and internal ERR
test_expect_ok_err() {
	show_oc_client_status
	show_routes
	show_nft_ruleset
	expect_ok ping_ext
	expect_err ping_int
	expect_ok curl_ext
	expect_err curl_int
	expect_ok get_log_errors
}

# run tests and expect external server ERR and internal OK
test_expect_err_ok() {
	show_oc_client_status
	show_routes
	show_nft_ruleset
	expect_err ping_ext
	expect_ok ping_int
	expect_err curl_ext
	expect_ok curl_int
	expect_ok get_log_errors
}

# run tests and expect external server OK and internal OK
test_expect_ok_ok() {
	show_oc_client_status
	show_routes
	show_nft_ruleset
	expect_ok ping_ext
	expect_ok ping_int
	expect_ok curl_ext
	expect_ok curl_int
	expect_ok get_log_errors
}

# run tests and expect external server ERR and internal ERR
test_expect_err_err() {
	show_oc_client_status
	show_routes
	show_nft_ruleset
	expect_err ping_ext
	expect_err ping_int
	expect_err curl_ext
	expect_err curl_int
	expect_ok get_log_errors
}

# common parts of tests with default settings in ocserv.conf.
test_default_common() {
	out "Testing before VPN connection..."
	test_expect_ok_err

	# connect vpn
	connect_vpn_cmdline

	out "Testing after VPN connection..."
	test_expect_err_ok

	# reconnect vpn
	reconnect_vpn_cmdline

	out "Testing after reconnecting VPN..."
	test_expect_err_ok

	# disconnect vpn
	disconnect_vpn_cmdline

	out "Testing after disconnecting VPN..."
	test_expect_ok_err
}

# run test with default settings in ocserv.conf.
test_default() {
	out "Setting up test..."
	start_containers

	test_default_common

	out "Shutting down test..."
	stop_oc_daemon
	save_gocover_dir
	stop_containers
}

# run test with default settings in ocserv.conf, ipv6 version.
test_default_ipv6() {
	out "Setting up test..."
	start_containers_ipv6

	test_default_common

	out "Shutting down test..."
	stop_oc_daemon
	save_gocover_dir
	stop_containers_ipv6
}

# common parts of tests with split routing for ext-web.
test_splitrt_common() {
	local config="# splitrt config
auth = \"certificate\"
tcp-port = 443
udp-port = 443
run-as-user = ocserv
run-as-group = ocserv
socket-file = /run/ocserv-socket
chroot-dir = /var/lib/ocserv
server-cert = /etc/ocserv/server-cert.pem
server-key = /etc/ocserv/server-key.pem
ca-cert = /etc/ocserv/ca-cert.pem
isolate-workers = true
max-clients = 16
max-same-clients = 2
rate-limit-ms = 100
server-stats-reset-time = 604800
keepalive = 32400
dpd = 90
mobile-dpd = 1800
switch-to-tcp-timeout = 25
try-mtu-discovery = false
cert-user-oid = 0.9.2342.19200300.100.1.1
tls-priorities = \"NORMAL:%SERVER_PRECEDENCE:%COMPAT:-VERS-SSL3.0:-VERS-TLS1.0:-VERS-TLS1.1:-VERS-TLS1.3\"
auth-timeout = 240
min-reauth-time = 300
max-ban-score = 80
ban-reset-time = 1200
cookie-timeout = 300
deny-roaming = false
rekey-time = 172800
rekey-method = ssl
use-occtl = true
pid-file = /run/ocserv.pid
log-level = 1
device = vpns
predictable-ips = true
default-domain = example.com
ipv4-network = 192.168.1.0
ipv4-netmask = 255.255.255.0
dns = 192.168.1.1
ping-leases = false

# configure routing
route = default
no-route = $WEB_EXT_IPv4_EXT/32

cisco-client-compat = true
dtls-legacy = true
client-bypass-protocol = false
"
	set_ocserv_config "$config"

	out "Testing before VPN connection..."
	test_expect_ok_err

	# connect vpn
	connect_vpn_cmdline

	out "Testing after VPN connection..."
	test_expect_ok_ok

	# reconnect vpn
	reconnect_vpn_cmdline

	out "Testing after reconnecting VPN..."
	test_expect_ok_ok

	# disconnect vpn
	disconnect_vpn_cmdline

	out "Testing after disconnecting VPN..."
	test_expect_ok_err
}

# run test with split routing for ext-web.
test_splitrt() {
	out "Setting up test..."
	start_containers

	test_splitrt_common

	out "Shutting down test..."
	stop_oc_daemon
	save_gocover_dir
	stop_containers
}
# run test with split routing for ext-web, ipv6 version
test_splitrt_ipv6() {
	out "Setting up test..."
	start_containers_ipv6

	test_splitrt_common

	out "Shutting down test..."
	stop_oc_daemon
	save_gocover_dir
	stop_containers_ipv6
}

# run test with with restart
test_restart() {
	out "Setting up test..."
	start_containers

	# check errors in log before doing anything
	sleep 3
	expect_ok get_log_errors

	# check errors in log after restart without vpn connection
	restart_oc_daemon
	expect_ok get_log_errors

	# check errors in log after connecting vpn
	connect_vpn_cmdline
	sleep 3
	expect_ok get_log_errors

	# check errors in log after restart during connected vpn
	restart_oc_daemon
	expect_ok get_log_errors

	out "Shutting down test..."
	stop_oc_daemon
	save_gocover_dir
	stop_containers
}

# run test with reconnect
test_reconnect() {
	out "Setting up test..."
	start_containers

	# check errors in log before doing anything
	expect_ok get_log_errors

	# check errors in log after reconnect without vpn connection
	reconnect_vpn_cmdline
	sleep 3
	test_expect_err_ok

	# check errors in log after reconnect with vpn connection
	reconnect_vpn_cmdline
	sleep 3
	test_expect_err_ok

	out "Shutting down test..."
	stop_oc_daemon
	save_gocover_dir
	stop_containers
}

# run test with disconnect
test_disconnect() {
	out "Setting up test..."
	start_containers

	# check errors in log before doing anything
	expect_ok get_log_errors

	# check errors in log after disconnect without vpn connection
	disconnect_vpn_cmdline
	sleep 3
	test_expect_ok_err

	# check errors in log after disconnect with vpn connection
	connect_vpn_cmdline
	disconnect_vpn_cmdline
	sleep 3
	test_expect_ok_err

	out "Shutting down test..."
	stop_oc_daemon
	save_gocover_dir
	stop_containers
}

# run test with oc-client config
test_occlient_config() {
	out "Setting up test..."
	start_containers

	# test with system settings
	out "Testing with system settings..."
	save_oc_client_system_settings

	# connect vpn
	connect_vpn_default
	out "Testing with system settings, after VPN connection..."
	test_expect_err_ok

	# reconnect vpn
	reconnect_vpn_default
	out "Testing with system settings, after reconnecting VPN..."
	test_expect_err_ok

	# disconnect vpn
	disconnect_vpn_default
	out "Testing with system settings, after disconnecting VPN..."
	test_expect_ok_err

	# test with user settings
	out "Testing with user settings..."
	save_oc_client_user_settings

	# connect vpn
	connect_vpn_default
	out "Testing with user settings, after VPN connection..."
	test_expect_err_ok

	# reconnect vpn
	reconnect_vpn_default
	out "Testing with user settings, after reconnecting VPN..."
	test_expect_err_ok

	# disconnect vpn
	disconnect_vpn_default
	out "Testing with user settings, after disconnecting VPN..."
	test_expect_ok_err

	out "Shutting down test..."
	stop_oc_daemon
	save_gocover_dir
	stop_containers
}

# run test with xml profile, always on enabled
test_profile_alwayson() {
	out "Setting up test..."
	start_containers

	# set xml profile
	set_profile_oc_daemon $WEB_INT_NAME
	sleep 1
	test_expect_ok_err

	# connect vpn
	connect_vpn_cmdline
	out "Testing after VPN connection before restart..."
	test_expect_err_ok

	# restart, load xml profile on startup
	restart_oc_daemon
	sleep 1
	test_expect_ok_err

	# connect vpn
	connect_vpn_cmdline
	out "Testing after VPN connection after restart..."
	test_expect_err_ok

	# set xml profile again, should not change anything
	set_profile_oc_daemon $WEB_INT_NAME
	sleep 1
	test_expect_err_ok

	out "Shutting down test..."
	stop_oc_daemon
	save_gocover_dir
	stop_containers
}

# run test with xml profile, tnd enabled
test_profile_tnd() {
	out "Setting up test..."
	start_containers

	# set xml profile, when oc-daemon is already running
	# set external web server in profile, pretend to be in trusted network
	set_profile_oc_daemon $WEB_EXT_NAME
	sleep 1
	test_expect_ok_err

	# connect vpn
	connect_vpn_cmdline
	out "Testing after VPN connection attempt before restart..."
	test_expect_ok_err

	# restart, load xml profile on startup
	restart_oc_daemon
	sleep 1
	test_expect_ok_err

	# connect vpn
	connect_vpn_cmdline
	out "Testing after VPN connection attempt after restart..."
	test_expect_ok_err

	# set internal web server in profile, not in a trusted network
	set_profile_oc_daemon $WEB_INT_NAME
	sleep 1
	test_expect_ok_err

	# connect vpn
	connect_vpn_cmdline
	out "Testing after VPN connection after switching to internal server..."
	test_expect_err_ok

	out "Shutting down test..."
	stop_oc_daemon
	save_gocover_dir
	stop_containers
}

# run test with xml profile, profile from server
test_profile_server() {
	out "Setting up test..."
	start_containers

	# write profile.xml to ocserv
	local profile
	profile=$(get_profile $WEB_EXT_NAME)
	set_profile_ocserv "$profile"

	# set ocserv.conf with user-profile=profile.xml on ocserv
	local config="# profile.xml config
auth = \"certificate\"
tcp-port = 443
udp-port = 443
run-as-user = ocserv
run-as-group = ocserv
socket-file = /run/ocserv-socket
chroot-dir = /var/lib/ocserv
server-cert = /etc/ocserv/server-cert.pem
server-key = /etc/ocserv/server-key.pem
ca-cert = /etc/ocserv/ca-cert.pem
isolate-workers = true
max-clients = 16
max-same-clients = 2
rate-limit-ms = 100
server-stats-reset-time = 604800
keepalive = 32400
dpd = 90
mobile-dpd = 1800
switch-to-tcp-timeout = 25
try-mtu-discovery = false
cert-user-oid = 0.9.2342.19200300.100.1.1
tls-priorities = \"NORMAL:%SERVER_PRECEDENCE:%COMPAT:-VERS-SSL3.0:-VERS-TLS1.0:-VERS-TLS1.1:-VERS-TLS1.3\"
auth-timeout = 240
min-reauth-time = 300
max-ban-score = 80
ban-reset-time = 1200
cookie-timeout = 300
deny-roaming = false
rekey-time = 172800
rekey-method = ssl
use-occtl = true
pid-file = /run/ocserv.pid
log-level = 1
device = vpns
predictable-ips = true
default-domain = example.com
ipv4-network = 192.168.1.0
ipv4-netmask = 255.255.255.0
dns = 192.168.1.1
ping-leases = false
route = default

# configure profile.xml
user-profile = profile.xml

cisco-client-compat = true
dtls-legacy = true
client-bypass-protocol = false
"
	set_ocserv_config "$config"

	out "Testing before VPN connection..."
	test_expect_ok_err

	# connect vpn
	connect_vpn_cmdline
	out "Testing after VPN connection..."
	test_expect_ok_err

	# check profile on oc-daemon
	out "Checking XML Profile on oc-daemon after connect..."
	expect_ok is_equal_profile_oc_daemon "$profile"

	# reconnect vpn
	reconnect_vpn_cmdline
	out "Testing after VPN connection..."
	test_expect_ok_err

	# check profile on oc-daemon
	out "Checking XML Profile on oc-daemon after reconnect..."
	expect_ok is_equal_profile_oc_daemon "$profile"

	out "Shutting down test..."
	stop_oc_daemon
	save_gocover_dir
	stop_containers
}

# run test with CPD
test_cpd() {
	out "Setting up test..."
	start_containers

	# enable captive portal
	# pretend we are logged out and do not have network access
	enable_captive_portal

	# configure oc-daemon incl. cpd
	save_oc_client_system_settings
	set_profile_oc_daemon $WEB_INT_NAME
	set_oc_daemon_config_cpd
	restart_oc_daemon

	sleep 10

	# check if portal is detected
	expect_ok is_detected_captive_portal
	# make sure we cannot reach web-ext and web-int
	test_expect_err_err
	# make sure we cannot connect and
	# cannot reach web-ext and web-int
	connect_vpn_default
	test_expect_err_err

	# disable captive portal
	# pretend we are logged in and have network access now
	disable_captive_portal

	sleep 10

	# make sure portal is not detected any more
	expect_err is_detected_captive_portal

	# make sure we can reach web-ext now
	test_expect_ok_err
	# make sure we can connect now and
	# can reach web-int now
	connect_vpn_default
	test_expect_err_ok

	out "Shutting down test..."
	stop_oc_daemon
	save_gocover_dir
	stop_containers
}

# define test cases/runs
TEST_RUNS=(
	test_default
	test_default_ipv6
	test_splitrt
	test_splitrt_ipv6
	test_restart
	test_reconnect
	test_disconnect
	test_occlient_config
	test_profile_alwayson
	test_profile_tnd
	test_profile_server
	test_cpd
)

###############################################################################
###                                Startup                                  ###
###############################################################################

# helper to run a single test
run_test() {
	((TESTS++))
	out "==============================="
	out "Test $TESTS/$NUM_TESTS: $1"
	out "==============================="
	out "Starting test"
	local start_time=$SECONDS
	$1
	show_summary
	out "Test done, $(( SECONDS - start_time))s, total ${SECONDS}s"
}

# run all tests
run_all_tests() {
	NUM_TESTS=${#TEST_RUNS[@]}
	for i in "${TEST_RUNS[@]}"; do
		run_test "$i"
	done
	show_gocover_percent
}

# run specific test
run_specific_test() {
	NUM_TESTS=1
	run_test "$1"
	show_gocover_percent
}

# show usage
show_usage() {
	echo "Usage of $0:"
	echo "  help"
	echo "        show this help"
	echo "  list"
	echo "        list all available tests"
	echo "  up <ipv4|ipv6>"
	echo "        start test setup without running any tests,"
	echo "        remember to stop before running any tests"
	echo "  down <ipv4|ipv6>"
	echo "        stop previously started test setup"
	echo "  all"
	echo "        run all tests"
	echo "  <test>"
	echo "        run specific test, one of:"
	echo "       " "${TEST_RUNS[@]}"
}

# check command line arguments
if [ "$#" -lt 1 ]; then
	show_usage
	exit 2
fi

# handle command "up"
command_up() {
	case "$1" in
		ipv4)
			start_containers
			;;
		ipv6)
			start_containers_ipv6
			;;
		*)
			show_usage
			exit 2
			;;
	esac
}

# handle command "down"
command_down() {
	case "$1" in
		ipv4)
			stop_containers
			;;
		ipv6)
			stop_containers_ipv6
			;;
		*)
			show_usage
			exit 2
			;;
	esac
}

# parse command
case "$1" in
	help)
		# handle command "help"
		show_usage
		exit 0
		;;
	list)
		# handle command "list"
		echo "${TEST_RUNS[@]}"
		exit 0
		;;
	up)
		# handle command "up"
		command_up "$2"
		;;
	down)
		# handle command "down"
		command_down "$2"
		;;
	all)
		# handle command "all"
		run_all_tests 2>&1 | tee $LOG_FILE | $GREP "^==="
		;;
	*)
		# handle specific test
		if [[ ! " ${TEST_RUNS[*]} " =~ [[:space:]]$1[[:space:]] ]]; then
			    echo "unknown test"
			    exit 1
		fi
		run_specific_test "$1" 2>&1 | tee $LOG_FILE | $GREP "^==="
		;;
esac

# return error if a test failed
if [ $FAILS -ne 0 ]; then
	exit 1
fi
