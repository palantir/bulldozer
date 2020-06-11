FROM scratch

STOPSIGNAL SIGINT

# add the default configuration file
COPY config/bulldozer.example.yml /secrets/bulldozer.yml

# add static files
COPY ca-certificates.crt /etc/ssl/certs/

# add application files
COPY bulldozer /

ENTRYPOINT ["build/linux-amd64/bulldozer"]
CMD ["server", "--config", "/secrets/bulldozer.yml"]
