#!/bin/bash
## Reference: https://github.com/norwoodj/helm-docs
set -eux

echo "Running Helm-Docs"
docker run \
    -v "$(pwd):/helm-docs" \
    -u $(id -u) \
    jnorwood/helm-docs:v1.11.0
