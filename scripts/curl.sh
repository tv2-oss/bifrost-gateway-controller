#! /bin/bash

docker run --rm --net host -v $PWD:/local:ro -w /local curlimages/curl:7.87.0 "$@"
