#!/usr/bin/env zsh

base_directory=${0:h}

cd "${base_directory}" || exit 1
echo -e "\033[33m-->\033[0m Building \033[1;32mwin-loss-api\033[0m"
go build && \
    echo -e "\033[36m-->\033[0m Running \033[1;32mwin-loss-api\033[0m" && \
    "${base_directory}"/win-loss-rux
