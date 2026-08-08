FROM alpine:3.20

# postgresql16-client: pg_dump/pg_restore/psql matching the postgres:16-alpine
#   service in docker-compose.yml
# mc (MinIO client): downloaded binary, not an Alpine package (apk's "mc" is
#   Midnight Commander)
RUN apk add --no-cache \
      postgresql16-client \
      bash \
      coreutils \
      findutils \
      gzip \
      openssl \
      curl \
      ca-certificates \
    && curl -fsSL -o /usr/local/bin/mc https://dl.min.io/client/mc/release/linux-amd64/mc \
    && chmod +x /usr/local/bin/mc

COPY scripts/ /scripts/
RUN chmod +x /scripts/*.sh

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENV BACKUP_DIR=/backups
VOLUME ["/backups"]

ENTRYPOINT ["/entrypoint.sh"]
