#!/bin/bash

service postgresql restart &&\
  make run-server -C /go/src/github.com/c3systems/c3-sdk-go-example-mattermost &&\
  make wait -C /go/src/github.com/c3systems/c3-sdk-go-example-mattermost &&\
  tail -f /dev/stdout
