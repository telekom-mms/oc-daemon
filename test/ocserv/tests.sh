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
	$PODMAN exec "$DEB12_NAME" curl -s --connect-timeout 3 "$WEB_EXT_IP_EXT"
}

# curl internal web server
curl_int() {
	echo "HTTP GET internal web server"
	$PODMAN exec "$DEB12_NAME" curl -s --connect-timeout 3 "$WEB_INT_IP_INT"
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

# run test
run_test() {
	echo "Setting up test..."
	start_containers
	get_settings
	configure_routing
	echo "Ping testing before VPN connection..."
	expect_ok ping_ext
	expect_err ping_int
	echo "HTTP GET testing before VPN connection..."
	expect_ok curl_ext
	expect_err curl_int

	connect_vpn
	echo "Ping testing after VPN connection..."
	expect_err ping_ext
	expect_ok ping_int
	echo "HTTP GET testing after VPN connection..."
	expect_err curl_ext
	expect_ok curl_int

	echo "Shutting down test..."
	stop_containers
}

run_test
((TESTS++))

echo "==============="
echo "=== Summary ==="
echo "==============="
echo "Tests: $TESTS"
echo "OKs: $OKS"
echo "FAILs: $FAILS"

if [ $FAILS -ne 0 ]; then
	exit 1
fi
