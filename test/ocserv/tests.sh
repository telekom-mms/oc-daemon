#!/bin/bash

PODMAN=podman
PODMAN_COMPOSE=podman-compose
#NSENTER=nsenter

# start networks and containers
echo "Starting networks and containers..."
COMPOSE_PARALLEL_LIMIT=1 $PODMAN_COMPOSE --file "$PWD/test/ocserv/compose.yml" up --detach

# networks
NETWORK_EXT_NAME="oc-daemon-test_ext"
NETWORK_INT_NAME="oc-daemon-test_int"

# ocserv
OCSERV_NAME="ocserv"
OCSERV_PID=$($PODMAN inspect --format "{{.State.Pid}}" $OCSERV_NAME)
OCSERV_IP_EXT=$($PODMAN inspect --format "{{range \$k,\$v := .NetworkSettings.Networks}}{{if eq \$k \"$NETWORK_EXT_NAME\"}}{{\$v.IPAddress}}{{end}}{{end}}" $OCSERV_NAME)
OCSERV_IP_INT=$($PODMAN inspect --format "{{range \$k,\$v := .NetworkSettings.Networks}}{{if eq \$k \"$NETWORK_INT_NAME\"}}{{\$v.IPAddress}}{{end}}{{end}}" $OCSERV_NAME)

# deb12
DEB12_NAME="deb12"
DEB12_PID=$($PODMAN inspect --format "{{.State.Pid}}" $DEB12_NAME)
DEB12_IP_EXT=$($PODMAN inspect --format "{{range \$k,\$v := .NetworkSettings.Networks}}{{if eq \$k \"$NETWORK_EXT_NAME\"}}{{\$v.IPAddress}}{{end}}{{end}}" $DEB12_NAME)

# web-ext
WEB_EXT_NAME="ext-web"
WEB_EXT_PID=$($PODMAN inspect --format "{{.State.Pid}}" $WEB_EXT_NAME)
WEB_EXT_IP_EXT=$($PODMAN inspect --format "{{range \$k,\$v := .NetworkSettings.Networks}}{{if eq \$k \"$NETWORK_EXT_NAME\"}}{{\$v.IPAddress}}{{end}}{{end}}" $WEB_EXT_NAME)

# web-int
WEB_INT_NAME="int-web"
WEB_INT_PID=$($PODMAN inspect --format "{{.State.Pid}}" $WEB_INT_NAME)
WEB_INT_IP_INT=$($PODMAN inspect --format "{{range \$k,\$v := .NetworkSettings.Networks}}{{if eq \$k \"$NETWORK_INT_NAME\"}}{{\$v.IPAddress}}{{end}}{{end}}" $WEB_INT_NAME)

# print infos about containers
echo ""
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
echo ""

# configure routing
echo "Configuring routing in containers..."
#sudo $NSENTER -n -t "$WEB_INT_PID" ip route add default via "$OCSERV_IP_INT"
#sudo $NSENTER -n -t "$WEB_INT_PID" ip route show
$PODMAN exec "$WEB_INT_NAME" ip route add default via "$OCSERV_IP_INT"
$PODMAN exec "$WEB_INT_NAME" ip route show

# ping test before vpn connection
echo ""
echo "Ping testing before VPN connection..."
echo ""
echo "Pinging external web server"
$PODMAN exec "$DEB12_NAME" ping -c 3 "$WEB_EXT_IP_EXT"
echo ""
echo "Pinging internal web server"
$PODMAN exec "$DEB12_NAME" ping -c 3 "$WEB_INT_IP_INT"

# connect vpn
echo ""
echo "Connecting to VPN..."
$PODMAN exec "$DEB12_NAME" oc-client -ca ca-cert.pem -key client-key.pem -cert client-cert.pem -server "$OCSERV_NAME"

# ping test after vpn connection
echo ""
echo "Ping testing after VPN connection..."
echo ""
echo "Pinging external web server"
$PODMAN exec "$DEB12_NAME" ping -c 3 "$WEB_EXT_IP_EXT"
echo ""
echo "Pinging internal web server"
$PODMAN exec "$DEB12_NAME" ping -c 3 "$WEB_INT_IP_INT"

# shut down networks and containers
echo ""
echo "Stopping networks and containers..."
$PODMAN_COMPOSE --file "$PWD/test/ocserv/compose.yml" down
