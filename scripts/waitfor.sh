#! /bin/bash

# Repeat running command until it succeeds, with timeout after 60 attempts spaced 1s (+time running command)

for i in {1..60}; do
  if [ "$(${@})" ]; then
    break
  fi
  sleep 1
done
if [ ! "$(${@})" ]; then
  echo "Failed ro run '${1}' after 60 attempts"
fi
