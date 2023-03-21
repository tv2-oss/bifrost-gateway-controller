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
kubectl -n foo-infra apply -f hack/demo/foo-namespaces.yaml -f hack/demo/foo-gateway.yaml
kubectl -n foo-site apply -f test-data/getting-started/app-foo-site.yaml
kubectl -n foo-site  apply -f test-data/getting-started/foo-site-httproute.yaml
kubectl -n foo-store apply -f test-data/getting-started/app-foo-store-v1.yaml
kubectl -n foo-store apply -f test-data/getting-started/app-foo-store-v2.yaml
kubectl -n foo-store apply -f test-data/getting-started/foo-store-httproute.yaml

echo ""
echo "-------------------------------------------------------------------"
read -p "Press enter to show user GatewayConfig with ACL CIDR"
hack/demo/test-add-user-acl.sh
