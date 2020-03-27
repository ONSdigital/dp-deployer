#!/bin/bash -eux

pushd dp-deployer
  make build 
  cp build/$(go env GOOS)-$(go env GOARCH)/* Dockerfile.concourse ../build
popd
