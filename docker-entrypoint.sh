#!/bin/bash
/etc/init.d/postgresql start &&\
  sh ./wait-for-postgres.sh &&\
  psql -U postgres --command "CREATE DATABASE mattermost_db;" &&\
  psql -U postgres --command "CREATE USER docker WITH SUPERUSER; ALTER USER docker VALID UNTIL 'infinity'; GRANT ALL PRIVILEGES ON DATABASE mattermost_db TO docker;" &&\
  make run-server -C /go/src/github.com/c3systems/c3-sdk-go-example-mattermost &&\
  make wait -C /go/src/github.com/c3systems/c3-sdk-go-example-mattermost &&\
  tail -f /dev/stdout
