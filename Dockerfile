FROM ubuntu:18.04
#FROM ubuntu

ENV DEBIAN_FRONTEND=noninteractive
ENV GOPATH /go
ENV PATH $GOPATH/bin:$PATH

RUN mkdir -p /go /go/bin /go/src /go/src/github.com/c3systems/c3-sdk-go-example-mattermost /go/pkg
RUN apt-get update -y && apt-get upgrade -y
RUN apt-get install -y --no-install-recommends --fix-missing make curl python gnupg2 dirmngr golang-go
RUN apt-get autoremove -y
RUN apt-get update -y --no-install-recommends

# Add the PostgreSQL PGP key to verify their Debian packages.
# It should be the same key as https://www.postgresql.org/media/keys/ACCC4CF8.asc
RUN apt-key adv --keyserver hkp://p80.pool.sks-keyservers.net:80 --recv-keys B97B0AFCAA1A47F044F244A07FCC7D46ACCC4CF8

# Add PostgreSQL's repository. It contains the most recent stable release
#     of PostgreSQL, ``9.3``.
RUN echo "deb http://apt.postgresql.org/pub/repos/apt/ precise-pgdg main" > /etc/apt/sources.list.d/pgdg.list

# Install ``python-software-properties``, ``software-properties-common`` and PostgreSQL 9.3
#  There are some warnings (in red) that show up during the build. You can hide
#  them by prefixing each apt-get statement with DEBIAN_FRONTEND=noninteractive
RUN apt-get update -y --no-install-recommends && apt-get install -y --no-install-recommends postgresql-9.3 postgresql-client-9.3 postgresql-contrib-9.3

# Note: The official Debian and Ubuntu images automatically ``apt-get clean``
# after each ``apt-get``

USER postgres

# Create a PostgreSQL role named `docker` with password `docker` and
# then create a database `mattermost-db` owned by the `docker` role.
RUN /etc/init.d/postgresql start &&\
        psql --command "CREATE DATABASE mattermost_db;" &&\
        psql --command "CREATE USER docker WITH SUPERUSER; ALTER USER docker VALID UNTIL 'infinity'; GRANT ALL PRIVILEGES ON DATABASE mattermost_db TO docker;"
# createdb -O docker mattermost_db

# Adjust PostgreSQL configuration so that remote connections to the
# database are possible.
# And add ``listen_addresses`` to ``/etc/postgresql/9.3/main/postgresql.conf``
RUN rm /etc/postgresql/9.3/main/pg_hba.conf &&\
    echo "local all all trust" >> /etc/postgresql/9.3/main/pg_hba.conf &&\
    echo "host all all 127.0.0.1/32 trust" >> /etc/postgresql/9.3/main/pg_hba.conf &&\
    echo "host all all ::1/128 trust" >> /etc/postgresql/9.3/main/pg_hba.conf &&\
    echo "listen_addresses='*'" >> /etc/postgresql/9.3/main/postgresql.conf &&\
    /etc/init.d/postgresql restart

# Expose the postgresql and mattermost ports
EXPOSE 5432
EXPOSE 8065

USER root

# Cd into the api code directory
WORKDIR /go/src/github.com/c3systems/c3-sdk-go-example-mattermost

# Copy the local package files to the container's workspace.
COPY . /go/src/github.com/c3systems/c3-sdk-go-example-mattermost

RUN ["chmod", "+x", "/go/src/github.com/c3systems/c3-sdk-go-example-mattermost/docker-entrypoint.sh"]
RUN ["chmod", "+x", "/go/src/github.com/c3systems/c3-sdk-go-example-mattermost/wait.sh"]
ENTRYPOINT ["/go/src/github.com/c3systems/c3-sdk-go-example-mattermost/docker-entrypoint.sh"]
