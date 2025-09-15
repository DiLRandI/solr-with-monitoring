FROM solr:9.9.0

# Provide configsets for SolrCloud collection creation (do NOT precreate standalone cores)
# This makes configsets named "movies" and "users" available under /opt/solr/server/solr/configsets/
COPY _template /opt/solr/server/solr/configsets/movies
COPY _template /opt/solr/server/solr/configsets/users

# Expose Solr HTTP port
EXPOSE 8983

# Default command; SolrCloud will be enabled via docker-compose command/args
CMD ["solr-foreground"]
