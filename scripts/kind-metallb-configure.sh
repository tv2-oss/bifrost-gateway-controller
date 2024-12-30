#! /bin/bash

for cidr in $(docker network inspect -f '{{json .IPAM.Config}}' kind | jq -r '.[].Subnet'); do
    echo ">> $cidr"
    if [[ "$cidr" =~ ([0-9]+)\.([0-9]+)\.([0-9]+)\.([0-9]+)\/([0-9]+) ]]; then
        b1=${BASH_REMATCH[1]}
        b2=${BASH_REMATCH[2]}
        b3=${BASH_REMATCH[3]}
        b4=${BASH_REMATCH[4]}
        nm=${BASH_REMATCH[5]}
        break
    fi
done

echo "Docker IPv4 CIDR: $b1.$b2.$b3.$b4/$nm"

CIDR_START="$b1.$b2.$b3.200"
CIDR_END="$b1.$b2.$b3.250"

cat <<EOF | kubectl apply -f -
apiVersion: metallb.io/v1beta1
kind: IPAddressPool
metadata:
  name: example
  namespace: metallb-system
spec:
  addresses:
  - $CIDR_START-$CIDR_END
---
apiVersion: metallb.io/v1beta1
kind: L2Advertisement
metadata:
  name: empty
  namespace: metallb-system
EOF
