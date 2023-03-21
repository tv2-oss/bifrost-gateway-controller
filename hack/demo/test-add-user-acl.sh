#! /bin/bash

MYIP=`curl -s ifconfig.me`

echo "************************"
echo "Using local IP: $MYIP"
echo "************************"
echo ""

cat hack/demo/user-gateway-acl.yaml | sed -e "s/1.2.3.4/$MYIP/"

echo ""
read -p "Press enter to deploy GatewayConfig"

cat hack/demo/user-gateway-acl.yaml | sed -e "s/1.2.3.4/$MYIP/" | kubectl apply -f -
