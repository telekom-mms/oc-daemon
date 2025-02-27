FROM debian:12-slim

COPY dist/oc-daemon*.deb /
COPY test/ocserv/client.tmpl /
#COPY ./dist/oc-daemon_1.3.2-n1740585083_amd64.deb /

RUN \
apt update && \
apt install -y /*.deb && \
apt install -y procps systemd systemd-sysv systemd-resolved gnutls-bin iputils-ping

RUN systemctl enable oc-daemon.service

RUN \
certtool --generate-privkey --outfile /key.pem && \
certtool --generate-self-signed \
	--load-privkey /key.pem \
	--template /client.tmpl \
	--outfile /cert.pem

#ENTRYPOINT ["/oc-daemon/test/ocserv/deb12.sh"]
CMD [ "/sbin/init" ]
