#! /bin/bash

ADDR=`kubectl -n foo-infra get gateway foo-gateway -o jsonpath='{.status.addresses[0].value}'`
IP=`dig "$ADDR" +short | head -n1`

echo "-------------------------------------------------------------------"
echo "Skipping DNS, using $ADDR = $IP"
echo "-------------------------------------------------------------------"
read -p "Press enter to run CURL commands"

echo "-------------------------------------------------------------------"
echo ""
echo "1x curl http://foo.example.com/site"
curl --resolve foo.example.com:80:$IP http://foo.example.com/site

echo "-------------------------------------------------------------------"
echo ""
echo "20x curl http://foo.example.com/store"
for i in {1..20}
do
    curl --resolve foo.example.com:80:$IP http://foo.example.com/store
done
