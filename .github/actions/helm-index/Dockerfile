ARG VERSION=3.6.3
FROM  alpine/helm:$VERSION

# # Create a non-root user matching GitHub Actions runner UID
# RUN addgroup -g 1001 runner && \
#     adduser -D -u 1001 -G runner runner

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# # Switch to non-root user
# USER runner

ENTRYPOINT ["/entrypoint.sh"]
