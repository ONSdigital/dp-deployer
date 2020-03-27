#!/bin/bash -eux

pushd dp-deployer
  make build 
  cp build/dp-deployer Dockerfile.concourse ../build
popd
