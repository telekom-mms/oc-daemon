FROM debian:12-slim

EXPOSE 443/tcp
EXPOSE 443/udp

RUN \
	apt update && \
	apt install -y ocserv

# COPY ca.tmpl server.tmpl ocserv.conf /etc/ocserv/
#COPY ocserv.sh /usr/bin/

#RUN \
#	certtool --generate-privkey --outfile /etc/ocserv/ca-key.pem && \
#	certtool --generate-self-signed \
#		--load-privkey /etc/ocserv/ca-key.pem \
#		--template /etc/ocserv/ca.tmpl \
#		--outfile /etc/ocserv/ca-cert.pem && \
#	certtool --generate-privkey --outfile /etc/ocserv/server-key.pem && \
#	certtool --generate-certificate \
#		--load-privkey /etc/ocserv/server-key.pem \
#		--load-ca-certificate /etc/ocserv/ca-cert.pem \
#		--load-ca-privkey /etc/ocserv/ca-key.pem \
#		--template /etc/ocserv/server.tmpl \
#		--outfile /etc/ocserv/server-cert.pem && \
#	echo test_password | ocpasswd -c /etc/ocserv/passwd test_user

#CMD ["ocserv", "-f"]
#ENTRYPOINT ["/usr/bin/ocserv.sh"]
ENTRYPOINT ["/ocserv/ocserv.sh"]
