# builder for certificates
FROM debian:12-slim AS certs

COPY \
test/ocserv/ca.tmpl \
test/ocserv/server.tmpl \
test/ocserv/client.tmpl \
test/ocserv/web-ext.tmpl \
test/ocserv/web-int.tmpl \
/

RUN \
apt-get update && \
apt-get install -y gnutls-bin && \
certtool --generate-privkey --outfile /ca-key.pem && \
certtool --generate-self-signed \
	--load-privkey /ca-key.pem \
	--template /ca.tmpl \
	--outfile /ca-cert.pem && \
certtool --generate-privkey --outfile /server-key.pem && \
certtool --generate-certificate \
	--load-privkey /server-key.pem \
	--load-ca-certificate /ca-cert.pem \
	--load-ca-privkey /ca-key.pem \
	--template /server.tmpl \
	--outfile /server-cert.pem && \
certtool --generate-privkey --outfile /client-key.pem && \
certtool --generate-certificate \
	--load-privkey /client-key.pem \
	--load-ca-certificate /ca-cert.pem \
	--load-ca-privkey /ca-key.pem \
	--template /client.tmpl \
	--outfile /client-cert.pem && \
certtool --generate-privkey --outfile /web-ext-key.pem && \
certtool --generate-certificate \
	--load-privkey /web-ext-key.pem \
	--load-ca-certificate /ca-cert.pem \
	--load-ca-privkey /ca-key.pem \
	--template /web-ext.tmpl \
	--outfile /web-ext-cert.pem && \
certtool --generate-privkey --outfile /web-int-key.pem && \
certtool --generate-certificate \
	--load-privkey /web-int-key.pem \
	--load-ca-certificate /ca-cert.pem \
	--load-ca-privkey /ca-key.pem \
	--template /web-int.tmpl \
	--outfile /web-int-cert.pem

## builder for oc-daemon debian package
#FROM goreleaser/goreleaser AS pkg
#
#RUN --mount=type=bind,source=.,target=/code,rw \
#cd /code && \
#goreleaser release --snapshot --clean && \
#cp dist/*.deb /

# ocserv
FROM debian:12-slim AS ocserv

EXPOSE 443/tcp
EXPOSE 443/udp

RUN \
apt-get update && \
apt-get install -y ocserv

COPY --from=certs /ca-cert.pem /server-key.pem /server-cert.pem /etc/ocserv/
COPY test/ocserv/ocserv.conf /etc/ocserv/

CMD ["ocserv", "-f"]

# oc-daemon
FROM debian:12-slim AS oc-daemon

COPY --from=certs /ca-cert.pem /client-key.pem /client-cert.pem /
COPY dist/oc-daemon*.deb /
#COPY --from=pkg /oc-daemon*.deb /

RUN \
apt-get update && \
apt-get install -y /*.deb && \
apt-get install -y procps systemd systemd-sysv systemd-resolved iputils-ping curl musl && \
systemctl enable oc-daemon.service && \
mkdir /gocover && \
mkdir /etc/systemd/system/oc-daemon.service.d && \
echo "[Service]\nEnvironment=\"GOCOVERDIR=/gocover\"" > /etc/systemd/system/oc-daemon.service.d/gocover.conf

CMD [ "/sbin/init" ]

# https server
FROM caddy:latest AS web

COPY test/ocserv/Caddyfile /etc/caddy/Caddyfile
COPY --from=certs /ca-cert.pem \
/web-ext-key.pem /web-ext-cert.pem \
/web-int-key.pem /web-int-cert.pem \
/
