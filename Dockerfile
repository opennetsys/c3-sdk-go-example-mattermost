FROM ubuntu:18.04

ENV DEBIAN_FRONTEND=noninteractive
ENV GOPATH /go
ENV PATH $GOPATH/bin:$PATH
ENV POSTGRES_URL postgres://docker:docker@localhost:5432/db?sslmode=disable

RUN mkdir -p /go /go/bin /go/src /go/src/github.com/c3systems/c3-sdk-go-example-mattermost /go/pkg &&\
  apt-get update -y && apt-get upgrade -y &&\
  apt-get install -y --no-install-recommends --fix-missing make curl python gnupg2 dirmngr golang-go build-essential ca-certificates git &&\
  apt-get autoremove -y &&\
  apt-get update -y --no-install-recommends &&\
  # Add the PostgreSQL PGP key to verify their Debian packages.
  # It should be the same key as https://www.postgresql.org/media/keys/ACCC4CF8.asc
  ( apt-key adv --keyserver ha.pool.sks-keyservers.net:80 --recv-keys B97B0AFCAA1A47F044F244A07FCC7D46ACCC4CF8 \
  || apt-key adv --keyserver pgp.mit.edu:80 --recv-keys B97B0AFCAA1A47F044F244A07FCC7D46ACCC4CF8 \
  || apt-key adv --keyserver keyserver.pgp.com:80 --recv-keys B97B0AFCAA1A47F044F244A07FCC7D46ACCC4CF8 ) &&\
  echo "deb http://apt.postgresql.org/pub/repos/apt/ precise-pgdg main" > /etc/apt/sources.list.d/pgdg.list &&\
  apt-get update -y --no-install-recommends && apt-get install -y --no-install-recommends --allow-unauthenticated postgresql-9.3 postgresql-client-9.3 postgresql-contrib-9.3 &&\
  # Adjust PostgreSQL configuration so that remote connections to the
  # database are possible.
  # And add ``listen_addresses`` to ``/etc/postgresql/9.3/main/postgresql.conf``
  rm /etc/postgresql/9.3/main/pg_hba.conf &&\
  echo "local all all trust" >> /etc/postgresql/9.3/main/pg_hba.conf &&\
  echo "host all all 127.0.0.1/32 trust" >> /etc/postgresql/9.3/main/pg_hba.conf &&\
  echo "host all all ::1/128 trust" >> /etc/postgresql/9.3/main/pg_hba.conf &&\
  echo "listen_addresses='*'" >> /etc/postgresql/9.3/main/postgresql.conf

# Expose the postgresql and mattermost ports
EXPOSE 5432
EXPOSE 8065

# Cd into the api code directory
WORKDIR /go/src/github.com/c3systems/c3-sdk-go-example-mattermost

# Copy the local package files to the container's workspace.
COPY . /go/src/github.com/c3systems/c3-sdk-go-example-mattermost

RUN chmod +x /go/src/github.com/c3systems/c3-sdk-go-example-mattermost/docker-entrypoint.sh &&\
  chmod +x /go/src/github.com/c3systems/c3-sdk-go-example-mattermost/wait.sh

CMD /go/src/github.com/c3systems/c3-sdk-go-example-mattermost/docker-entrypoint.sh
