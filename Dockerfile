FROM scratch

ARG VERSION

ADD ca-certificates.crt /etc/ssl/certs/
ADD build/${VERSION}/linux-amd64/bulldozer /
ADD client/build/ /assets/

EXPOSE 8080

CMD ["/bulldozer", "server", "--config", "/secrets/bulldozer.yml"]