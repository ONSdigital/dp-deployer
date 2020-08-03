#!/bin/bash -eux

export cwd=$(pwd)

pushd $cwd/dp-deployer
  make audit
popd 