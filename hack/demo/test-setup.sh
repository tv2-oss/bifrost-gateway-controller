#! /bin/bash

echo ""
echo "-------------------------------------------------------------------"
read -p "Press enter to deploy GatewayClassBlueprint + GatewayClass'es"
kubectl apply -f test-data/gatewayclassblueprint-aws-alb-crossplane.yaml
kubectl apply -f test-data/gatewayclass-aws-alb-crossplane.yaml

echo ""
echo "-------------------------------------------------------------------"
read -p "Press enter to deploy GatewayClassConfig's"
kubectl apply -f test-data/gatewayclassconfig-aws-alb-crossplane-dev-env.yaml

echo ""
echo "-------------------------------------------------------------------"
read -p "Press enter to deploy getting-started usecase"
make setup-getting-started-usecase
