#!/bin/sh

version="$(/app/server --version | awk -F'-' '{print $1}')"

mkdir -p /app/binaries/old/v"$version"

cp -pu /app/build_binaries/* /app/binaries/old/v"$version"
cp -pu /app/build_binaries/* /app/binaries/

/app/server -c /app/server_config.yaml -vvv