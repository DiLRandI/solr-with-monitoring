
FROM solr:9.9.0

# Install tini for process management
USER root
RUN apt-get update && apt-get install -y wget && rm -rf /var/lib/apt/lists/*
RUN wget -O /usr/local/bin/tini https://github.com/krallin/tini/releases/download/v0.19.0/tini && \
    chmod +x /usr/local/bin/tini




COPY start-solr-and-exporter.sh /start-solr-and-exporter.sh
RUN chmod +x /start-solr-and-exporter.sh
# Ensure Solr data directory exists and is owned by solr
RUN mkdir -p /var/solr/data && chown -R solr:solr /var/solr

USER solr


# Provide configsets for SolrCloud collection creation (do NOT precreate standalone cores)
# This makes configsets named "movies" and "users" available under /opt/solr/server/solr/configsets/
COPY _template /opt/solr/server/solr/configsets/movies
COPY _template /opt/solr/server/solr/configsets/users

# Copy custom solr-exporter config for Prometheus metrics
COPY solr-exporter-config.xml /opt/solr/contrib/prometheus-exporter/conf/solr-exporter-config.xml

# Expose Solr HTTP port and exporter port
EXPOSE 8983 9983

# Entrypoint: run both Solr and the exporter using tini and a startup script
ENTRYPOINT ["/usr/local/bin/tini", "--"]
CMD ["/start-solr-and-exporter.sh"]
