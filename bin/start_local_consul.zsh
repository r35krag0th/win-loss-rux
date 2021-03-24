#!/usr/bin/env zsh

base_directory=$(realpath $(dirname $0)/..)

cd $base_directory
consul agent -server \
    -bootstrap-expect=1 \
    -data-dir=${base_directory}/consul-data \
    -node=agent-one \
    -bind=127.0.0.1 \
    -enable-script-checks=true \
    -config-dir=${base_directory}/consul.d
