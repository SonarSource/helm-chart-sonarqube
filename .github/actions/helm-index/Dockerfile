ARG VERSION=3.6.3
FROM  alpine/helm:$VERSION

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
