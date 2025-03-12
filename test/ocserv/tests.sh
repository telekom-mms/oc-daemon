#!/bin/bash

# executables
PODMAN=podman
PODMAN_COMPOSE=podman-compose
#NSENTER=nsenter

# networks
NETWORK_EXT_NAME="oc-daemon-test_ext"
NETWORK_INT_NAME="oc-daemon-test_int"

# containers
OCSERV_NAME="ocserv"
DEB12_NAME="deb12"
WEB_EXT_NAME="ext-web"
WEB_INT_NAME="int-web"

# number of tests, OKs and fails
TESTS=0
OKS=0
FAILS=0

# start networks and containers
start_containers() {
	echo "Starting networks and containers..."
	COMPOSE_PARALLEL_LIMIT=1 $PODMAN_COMPOSE \
		--file "$PWD/test/ocserv/compose.yml" \
		up \
		--build \
		--detach
}

# shut down networks and containers
stop_containers() {
	echo "Stopping networks and containers..."
	$PODMAN_COMPOSE --file "$PWD/test/ocserv/compose.yml" down
}

# get container settings
get_settings() {
	# ocserv
	OCSERV_PID=$($PODMAN inspect --format "{{.State.Pid}}" $OCSERV_NAME)
	OCSERV_IP_EXT=$($PODMAN inspect \
		--format "{{range \$k,\$v := .NetworkSettings.Networks}}{{if eq \$k \"$NETWORK_EXT_NAME\"}}{{\$v.IPAddress}}{{end}}{{end}}" \
		$OCSERV_NAME)
	OCSERV_IP_INT=$($PODMAN inspect \
		--format "{{range \$k,\$v := .NetworkSettings.Networks}}{{if eq \$k \"$NETWORK_INT_NAME\"}}{{\$v.IPAddress}}{{end}}{{end}}" \
		$OCSERV_NAME)

	# deb12
	DEB12_PID=$($PODMAN inspect --format "{{.State.Pid}}" $DEB12_NAME)
	DEB12_IP_EXT=$($PODMAN inspect \
		--format "{{range \$k,\$v := .NetworkSettings.Networks}}{{if eq \$k \"$NETWORK_EXT_NAME\"}}{{\$v.IPAddress}}{{end}}{{end}}" \
		$DEB12_NAME)

	# web-ext
	WEB_EXT_PID=$($PODMAN inspect --format "{{.State.Pid}}" $WEB_EXT_NAME)
	WEB_EXT_IP_EXT=$($PODMAN inspect \
		--format "{{range \$k,\$v := .NetworkSettings.Networks}}{{if eq \$k \"$NETWORK_EXT_NAME\"}}{{\$v.IPAddress}}{{end}}{{end}}" \
		$WEB_EXT_NAME)

	# web-int
	WEB_INT_PID=$($PODMAN inspect --format "{{.State.Pid}}" $WEB_INT_NAME)
	WEB_INT_IP_INT=$($PODMAN inspect \
		--format "{{range \$k,\$v := .NetworkSettings.Networks}}{{if eq \$k \"$NETWORK_INT_NAME\"}}{{\$v.IPAddress}}{{end}}{{end}}" \
		$WEB_INT_NAME)

	# print infos about containers
	echo "Networks:"
	echo "- $NETWORK_EXT_NAME"
	echo "- $NETWORK_INT_NAME"
	echo "Containers:"
	echo "- $OCSERV_NAME:"
	echo "    PID: $OCSERV_PID"
	echo "    IP_EXT: $OCSERV_IP_EXT"
	echo "    IP_INT: $OCSERV_IP_INT"
	echo "- $DEB12_NAME:"
	echo "    PID: $DEB12_PID"
	echo "    IP_EXT: $DEB12_IP_EXT"
	echo "- $WEB_EXT_NAME:"
	echo "    PID: $WEB_EXT_PID"
	echo "    IP_EXT: $WEB_EXT_IP_EXT"
	echo "- $WEB_INT_NAME:"
	echo "    PID: $WEB_INT_PID"
	echo "    IP_INT: $WEB_INT_IP_INT"
}

# configure routing
configure_routing() {
	echo "Configuring routing in containers..."
	#sudo $NSENTER -n -t "$WEB_INT_PID" ip route add default via "$OCSERV_IP_INT"
	#sudo $NSENTER -n -t "$WEB_INT_PID" ip route show
	$PODMAN exec "$WEB_INT_NAME" ip route add default via "$OCSERV_IP_INT"
	$PODMAN exec "$WEB_INT_NAME" ip route show
}

# connect vpn
connect_vpn() {
	echo "Connecting to VPN..."
	$PODMAN exec "$DEB12_NAME" oc-client \
		-ca ca-cert.pem \
		-key client-key.pem \
		-cert client-cert.pem \
		-server "$OCSERV_NAME"
}

# ping external web server
ping_ext() {
	echo "Pinging external web server"
	$PODMAN exec "$DEB12_NAME" ping -c 3 "$WEB_EXT_IP_EXT"
}

# ping internal web server
ping_int() {
	echo "Pinging internal web server"
	$PODMAN exec "$DEB12_NAME" ping -c 3 "$WEB_INT_IP_INT"
}

# curl external web server
curl_ext() {
	echo "HTTP GET external web server"
	$PODMAN exec "$DEB12_NAME" curl -s --connect-timeout 3 "$WEB_EXT_IP_EXT" > /dev/null
}

# curl internal web server
curl_int() {
	echo "HTTP GET internal web server"
	$PODMAN exec "$DEB12_NAME" curl -s --connect-timeout 3 "$WEB_INT_IP_INT" > /dev/null
}

# run command in first argument and check whether return code is an error
expect_err() {
	if $1; then
		echo "FAIL: Line ${LINENO}/${BASH_LINENO[*]}: $1 should return error"
		((FAILS++))
	else
		echo "OK: Line ${LINENO}/${BASH_LINENO[*]}: $1 returned error as expected"
		((OKS++))
	fi
}

# run command in first argument and check whether return code is OK/no error
expect_ok() {
	if ! $1; then
		echo "FAIL: Line ${LINENO}/${BASH_LINENO[*]}: $1 should not return error"
		((FAILS++))
	else
		echo "OK: Line ${LINENO}/${BASH_LINENO[*]}: $1 returned no error as expected"
		((OKS++))
	fi
}

# TODO: test ipv6
# TODO: test ipv4 and ipv6
# TODO: test split routing with ipv4 and ipv6
# TODO: test always on/trafpol
# TODO: test profile update (from server?)?
# TODO: test TND?
# TODO: test Captive Portal Detection?

# set ocserv config
set_ocserv_config() {
	echo "Setting new ocserv config..."
	local config=$1

	# write it to ocserv
	$PODMAN exec "$OCSERV_NAME" sh -c "echo \"$config\" > /etc/ocserv/ocserv.conf"

	# reload ocserv
	$PODMAN exec "$OCSERV_NAME" occtl reload

	# wait
	sleep 1

	# check
	$PODMAN exec "$OCSERV_NAME" cat /etc/ocserv/ocserv.conf
}

# show routes on deb12
show_routes() {
	$PODMAN exec "$DEB12_NAME" ip -4 route show table all
	$PODMAN exec "$DEB12_NAME" ip -6 route show table all
}

# show nftables ruleset on deb12
show_nft_ruleset() {
	$PODMAN exec "$DEB12_NAME" nft list ruleset
}

# show test summary
show_summary() {
	echo "==============================="
	echo "=== Cumulative Test Summary ==="
	echo "==============================="
	echo "Tests: $TESTS"
	echo "OKs: $OKS"
	echo "FAILs: $FAILS"
	echo "==============================="
}

# run test with default settings in ocserv.conf
run_test_default() {
	echo "Setting up test..."
	start_containers
	get_settings
	configure_routing

	show_routes
	show_nft_ruleset
	echo "Ping testing before VPN connection..."
	expect_ok ping_ext
	expect_err ping_int
	echo "HTTP GET testing before VPN connection..."
	expect_ok curl_ext
	expect_err curl_int

	# connect vpn
	connect_vpn
	show_routes
	show_nft_ruleset

	echo "Ping testing after VPN connection..."
	expect_err ping_ext
	expect_ok ping_int
	echo "HTTP GET testing after VPN connection..."
	expect_err curl_ext
	expect_ok curl_int

	echo "Shutting down test..."
	stop_containers
}

# run test with split routing for ext-web
run_test_splitrt() {
	echo "Setting up test..."
	start_containers
	get_settings
	configure_routing

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
no-route = $WEB_EXT_IP_EXT/32

cisco-client-compat = true
dtls-legacy = true
client-bypass-protocol = false
"
	set_ocserv_config "$config"

	show_routes
	show_nft_ruleset
	echo "Ping testing before VPN connection..."
	expect_ok ping_ext
	expect_err ping_int
	echo "HTTP GET testing before VPN connection..."
	expect_ok curl_ext
	expect_err curl_int

	# connect vpn
	connect_vpn
	show_routes
	show_nft_ruleset

	echo "Ping testing after VPN connection..."
	expect_ok ping_ext
	expect_ok ping_int
	echo "HTTP GET testing after VPN connection..."
	expect_ok curl_ext
	expect_ok curl_int

	echo "Shutting down test..."
	stop_containers
}

# define test cases/runs
TEST_RUNS=(
	run_test_default
	run_test_splitrt
)

# run tests
for i in "${TEST_RUNS[@]}"; do
	((TESTS++))
	$i
	show_summary
done

# return error if a test failed
if [ $FAILS -ne 0 ]; then
	exit 1
fi
