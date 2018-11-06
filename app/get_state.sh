#!/bin/sh

echo "dump"
rm -rf ./data/pgdump
mkdir -p ./data/pgdump
PGPASSWORD="docker" pg_dump -d mattermost_db -h localhost -p 5432 -U docker > ./data/pgdump/mattermost_db.sql

echo "sort-dump"
python ./pg_dump_splitsort.py ./data/pgdump/mattermost_db.sql

echo "clean-dump"
rm ./data/pgdump/mattermost_db.sql ./data/pgdump/*status.sql

echo "tar-data"
rm ./state.tar
tar -cf ./state.tar ./data
