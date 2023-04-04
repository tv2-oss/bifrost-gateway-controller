#! /bin/bash

ADDR=`kubectl -n foo-infra get gateway foo-gateway -o jsonpath='{.status.addresses[0].value}'`
IP=`dig "$ADDR" +short | head -n1`
DOMAIN=foo.kubecon23.tv2dev.dk

echo "-------------------------------------------------------------------"
echo "Skipping DNS, using $DOMAIN = $IP"
echo "-------------------------------------------------------------------"
read -p "Press enter to run CURL commands"

echo "-------------------------------------------------------------------"
echo ""
echo "1x curl --resolve $DOMAIN:443:$IP https://$DOMAIN/site"
curl --resolve $DOMAIN:443:$IP https://$DOMAIN/site

echo "-------------------------------------------------------------------"
echo ""
echo "20x curl --resolve $DOMAIN:443:$IP https://$DOMAIN/store"
for i in {1..20}
do
    curl --resolve $DOMAIN:443:$IP https://$DOMAIN/store
done
