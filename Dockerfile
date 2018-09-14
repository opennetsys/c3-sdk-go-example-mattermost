FROM ubuntu:18.04

ENV DEBIAN_FRONTEND=noninteractive
ENV GOPATH /go
ENV PATH $GOPATH/bin:$PATH

# TODO: change github.com/mattermost to github.com/c3systems
RUN mkdir -p /go /go/bin /go/src /go/src/github.com/mattermost/mattermost-server /go/pkg
RUN apt-get update -y --no-install-recommends
RUN apt-get install -y --no-install-recommends software-properties-common build-essential curl python3.6 bzr git ca-certificates
RUN apt-get update -y --no-install-recommends
RUN apt-get install -y --no-install-recommends postgresql-10 golang-go

USER postgres

# Create a PostgreSQL role named ``docker`` with no password and
# then create a database `mattermost-db` owned by the ``docker`` role.
RUN /etc/init.d/postgresql start &&\
 psql --command "CREATE USER docker WITH PASSWORD 'docker' SUPERUSER;" &&\
 createdb -O docker mattermost_db

# Expose the postgresql and mattermost ports
EXPOSE 5432
EXPOSE 8065

USER root

# Cd into the api code directory
WORKDIR /go/src/github.com/mattermost/mattermost-server

# Copy the local package files to the container's workspace.
COPY . /go/src/github.com/mattermost/mattermost-server

RUN ["chmod", "+x", "/go/src/github.com/mattermost/mattermost-server/docker-entrypoint.sh"]
RUN ["chmod", "+x", "/go/src/github.com/mattermost/mattermost-server/wait.sh"]
ENTRYPOINT ["/go/src/github.com/mattermost/mattermost-server/docker-entrypoint.sh"]
