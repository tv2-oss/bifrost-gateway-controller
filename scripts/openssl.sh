#! /usr/bin/bash

docker run --rm -u `shell id -u`:`id -u` -v $PWD:/apps -w /apps nginx:1.23.3 openssl "$@"
