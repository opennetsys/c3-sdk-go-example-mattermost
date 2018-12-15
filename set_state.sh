#!/bin/sh

echo "untar-data"
rm -rf ./data
tar -xf ./state.tar

echo "pg-restore"
cat ./data/pgdump/*.sql | PGPASSWORD="docker" psql -d mattermost_db -h localhost -p 5432 -U docker

echo "done setting state"
