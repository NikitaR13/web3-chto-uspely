# frostfs/Dockerfile
FROM ghcr.io/nspcc-dev/frostfs:v0.10.0

COPY config.yaml /etc/frostfs/config.yaml
RUN mkdir -p /data/frostfs
VOLUME ["/data/frostfs"]

CMD ["frostfs", "server", "--config", "/etc/frostfs/config.yaml"]
