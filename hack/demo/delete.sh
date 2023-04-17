#! /bin/bash

set -x

SCOPE=${1:-""}

if [ -z "$SCOPE" ] || [ "$SCOPE" == "bifrost" ]; then
    helm uninstall -n bifrost-gateway-controller-system bifrost-gateway-controller
fi

if [ -z "$SCOPE" ] || [ "$SCOPE" == "app" ]; then
    #kubectl delete -n foo-infra gateway foo-gateway
    #kubectl delete -n foo-site httproute foo-site
    #kubectl delete -n foo-store httproute foo-store
    kubectl delete -f test-data/getting-started/foo-namespaces.yaml
fi

if [ -z "$SCOPE" ] || [ "$SCOPE" == "tenantconfig" ]; then
    kubectl delete -f hack/demo/namespace-gatewayclassconfig.yaml
fi

if [ -z "$SCOPE" ] || [ "$SCOPE" == "acl" ]; then
    kubectl delete -n foo-infra GatewayConfig foo-gateway-custom-acl
fi

if [ -z "$SCOPE" ] || [ "$SCOPE" == "clusterresources" ]; then
    hack/demo/delete-gw-cluster-resources.sh foo-infra foo-gateway
fi

if [ -z "$SCOPE" ] || [ "$SCOPE" == "configs" ]; then
    kubectl delete -f hack/demo/gatewayclassconfig-public.yaml
    kubectl delete -f hack/demo/gatewayclassconfig-internal.yaml
fi

if [ -z "$SCOPE" ] || [ "$SCOPE" == "blueprints" ]; then
    kubectl delete -f blueprints/gatewayclassblueprint-aws-alb-crossplane.yaml
    kubectl delete -f blueprints/gatewayclass-aws-alb-crossplane.yaml
fi
