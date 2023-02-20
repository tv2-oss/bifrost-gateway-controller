#! /usr/bin/bash

docker run --rm -u `id -u`:`id -g` -v $PWD:/apps -w /apps nginx:1.23.3 openssl "$@"
