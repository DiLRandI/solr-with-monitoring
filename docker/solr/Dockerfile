ARG SOLR_BASE_IMAGE=solr:10.0.0
FROM ${SOLR_BASE_IMAGE}

USER root

COPY --chown=solr:root solr/solr.xml /opt/solr/custom/solr.xml
COPY --chown=solr:root solr/configsets/movies-master /opt/solr/server/solr/configsets/movies-master
COPY --chown=solr:root solr/configsets/movies-follower /opt/solr/server/solr/configsets/movies-follower
COPY --chown=solr:root solr/configsets/books-master /opt/solr/server/solr/configsets/books-master
COPY --chown=solr:root solr/configsets/books-follower /opt/solr/server/solr/configsets/books-follower
COPY --chown=solr:root docker/solr/initdb/10-precreate-cores.sh /docker-entrypoint-initdb.d/10-precreate-cores.sh

RUN chmod +x /docker-entrypoint-initdb.d/10-precreate-cores.sh \
    && cp /opt/solr/custom/solr.xml /opt/solr/server/solr/solr.xml

USER solr

