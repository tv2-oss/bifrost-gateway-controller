#! /bin/bash

set -x

#kubectl delete -n foo-infra gateway foo-gateway
#kubectl delete -n foo-site httproute foo-site
#kubectl delete -n foo-store httproute foo-store
kubectl delete -f test-data/getting-started/foo-namespaces.yaml

kubectl delete securitygrouprule.ec2.aws.upbound.io/gw-foo-infra-foo-gateway-upstream15021
kubectl delete securitygrouprule.ec2.aws.upbound.io/gw-foo-infra-foo-gateway-upstream80
kubectl delete securitygrouprule.ec2.aws.upbound.io/gw-foo-infra-foo-gateway-egress15021
kubectl delete securitygrouprule.ec2.aws.upbound.io/gw-foo-infra-foo-gateway-egress80
kubectl delete securitygrouprule.ec2.aws.upbound.io/gw-foo-infra-foo-gateway-ingress
kubectl delete lblistener.elbv2.aws.upbound.io/gw-foo-infra-foo-gateway
kubectl delete lbtargetgroup.elbv2.aws.upbound.io/gw-foo-infra-foo-gateway
kubectl delete lb.elbv2.aws.upbound.io/gw-foo-infra-foo-gateway
kubectl delete securitygroup.ec2.aws.upbound.io/gw-foo-infra-foo-gateway
kubectl delete -n foo-infra targetgroupbinding.elbv2.k8s.aws/gw-foo-infra-foo-gateway

kubectl delete -f test-data/gatewayclassconfig-aws-alb-crossplane-dev-env.yaml
kubectl delete -f test-data/gatewayclassblueprint-aws-alb-crossplane.yaml
kubectl delete -f test-data/gatewayclass-aws-alb-crossplane.yaml
