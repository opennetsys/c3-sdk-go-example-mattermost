#!/bin/bash

service postgresql restart &&\
  make run-server -C /go/src/github.com/mattermost/mattermost-server &&\
  make wait -C /go/src/github.com/mattermost/mattermost-server &&\
  tail -f /dev/stdout
