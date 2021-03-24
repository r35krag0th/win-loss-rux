#!/usr/bin/env zsh

base_directory=$(realpath $(dirname $0)/..)

cd $base_directory
echo -e "\033[33m-->\033[0m Building \033[1;32mwin-loss-api\033[0m"
go build && \
    echo -e "\033[36m-->\033[0m Running \033[1;32mwin-loss-api\033[0m" && \
    ${base_directory}/win-loss-rux
