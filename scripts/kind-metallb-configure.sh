#! /bin/bash

CIDR=`docker network inspect -f '{{.IPAM.Config}}' kind | cut -d' ' -f1 | cut -c3-`

echo "Docker CIDR: $CIDR"

# Assume a /16 CIDR

BYTE12=`echo $CIDR | cut -d'.' -f1-2`

CIDR_START="$BYTE12.255.200"
CIDR_END="$BYTE12.255.250"

cat | kubectl apply -f - <<EOF
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
