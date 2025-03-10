# builder for certificates
FROM debian:12-slim AS certs

COPY test/ocserv/ca.tmpl test/ocserv/server.tmpl test/ocserv/client.tmpl /

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
	--outfile /client-cert.pem

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

RUN \
apt-get update && \
apt-get install -y /*.deb && \
apt-get install -y procps systemd systemd-sysv systemd-resolved iputils-ping curl && \
systemctl enable oc-daemon.service

CMD [ "/sbin/init" ]
