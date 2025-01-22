# neo-node/Dockerfile
FROM ghcr.io/nspcc-dev/neo-go:v0.99.3

WORKDIR /app
COPY config /app/config

CMD ["node", "--config-path", "/app/config", "--privnet"]
