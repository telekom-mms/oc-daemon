# builder for certificates
FROM docker.io/library/debian:12-slim AS certs

COPY \
test/ocserv/certs/ca.tmpl \
test/ocserv/certs/ocserv.tmpl \
test/ocserv/certs/oc-daemon.tmpl \
test/ocserv/certs/web-ext.tmpl \
test/ocserv/certs/web-int.tmpl \
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
	--template /ocserv.tmpl \
	--outfile /server-cert.pem && \
certtool --generate-privkey --outfile /client-key.pem && \
certtool --generate-certificate \
	--load-privkey /client-key.pem \
	--load-ca-certificate /ca-cert.pem \
	--load-ca-privkey /ca-key.pem \
	--template /oc-daemon.tmpl \
	--outfile /client-cert.pem && \
certtool --generate-privkey --outfile /web-ext-key.pem && \
certtool --generate-certificate \
	--load-privkey /web-ext-key.pem \
	--load-ca-certificate /ca-cert.pem \
	--load-ca-privkey /ca-key.pem \
	--template /web-ext.tmpl \
	--outfile /web-ext-cert.pem && \
certtool --fingerprint \
	--infile /web-ext-cert.pem \
	--hash sha256 \
	--outfile=/web-ext-cert.sum && \
certtool --generate-privkey --outfile /web-int-key.pem && \
certtool --generate-certificate \
	--load-privkey /web-int-key.pem \
	--load-ca-certificate /ca-cert.pem \
	--load-ca-privkey /ca-key.pem \
	--template /web-int.tmpl \
	--outfile /web-int-cert.pem && \
certtool --fingerprint \
	--infile /web-int-cert.pem \
	--hash sha256 \
	--outfile=/web-int-cert.sum

# ocserv
FROM docker.io/library/debian:12-slim AS ocserv

EXPOSE 443/tcp
EXPOSE 443/udp

RUN \
apt-get update && \
apt-get install -y ocserv

COPY --from=certs /ca-cert.pem /server-key.pem /server-cert.pem /etc/ocserv/
COPY test/ocserv/ocserv/ocserv.conf /etc/ocserv/

CMD ["ocserv", "-f"]

# oc-daemon
FROM docker.io/library/debian:12-slim AS oc-daemon

COPY --from=certs /ca-cert.pem /client-key.pem /client-cert.pem /

COPY dist/oc-daemon*.deb /

RUN \
apt-get update && \
apt-get install -y /*.deb && \
apt-get install -y procps systemd systemd-sysv systemd-resolved iputils-ping curl musl && \
systemctl enable oc-daemon.service && \
mkdir /gocover && \
mkdir /etc/systemd/system/oc-daemon.service.d && \
echo "[Service]\nEnvironment=\"GOCOVERDIR=/gocover\"" > /etc/systemd/system/oc-daemon.service.d/gocover.conf

CMD [ "/sbin/init" ]

# web-ext
FROM docker.io/library/caddy:latest AS web-ext

COPY test/ocserv/web/Caddyfile /etc/caddy/Caddyfile
COPY --from=certs /ca-cert.pem /
COPY --from=certs /web-ext-key.pem /web-key.pem
COPY --from=certs /web-ext-cert.pem /web-cert.pem
COPY --from=certs /web-ext-cert.sum /web-cert.sum

# web-int
FROM docker.io/library/caddy:latest AS web-int

COPY test/ocserv/web/Caddyfile /etc/caddy/Caddyfile
COPY --from=certs /ca-cert.pem /
COPY --from=certs /web-int-key.pem /web-key.pem
COPY --from=certs /web-int-cert.pem /web-cert.pem
COPY --from=certs /web-int-cert.sum /web-cert.sum
