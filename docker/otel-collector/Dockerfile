ARG OTEL_COLLECTOR_BASE_IMAGE=otel/opentelemetry-collector-contrib:0.148.0

FROM busybox:1.36.1-musl AS busybox

FROM ${OTEL_COLLECTOR_BASE_IMAGE}

# The upstream collector image is distroless, so copy one tiny static binary
# purely to probe the existing health_check endpoint from Docker healthchecks.
COPY --from=busybox /bin/busybox /bin/busybox
