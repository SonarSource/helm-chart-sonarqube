ARG VERSION=3.6.3
FROM  alpine/helm:$VERSION

# Create a non-root user
RUN addgroup -g 1000 helm && \
    adduser -D -u 1000 -G helm helm

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Switch to non-root user
USER helm

ENTRYPOINT ["/entrypoint.sh"]
