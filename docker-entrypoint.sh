#!/bin/bash

service postgresql restart &&\
  make run-server -C /go/src/github.com/c3systems/mattermost-server &&\
  make wait -C /go/src/github.com/c3systems/mattermost-server &&\
  tail -f /dev/stdout
