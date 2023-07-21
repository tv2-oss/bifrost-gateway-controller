#! /bin/bash

NS=$1
GWNAME=$2

NAME=gw-${NS}-${GWNAME}

kubectl delete securitygrouprule.ec2.aws.upbound.io/${NAME}-upstream15021
kubectl delete securitygrouprule.ec2.aws.upbound.io/${NAME}-upstream80
kubectl delete securitygrouprule.ec2.aws.upbound.io/${NAME}-egress15021
kubectl delete securitygrouprule.ec2.aws.upbound.io/${NAME}-egress80
kubectl delete securitygrouprule.ec2.aws.upbound.io/${NAME}-ingress80
kubectl delete securitygrouprule.ec2.aws.upbound.io/${NAME}-ingress443
kubectl delete lblistener.elbv2.aws.upbound.io/${NAME}
kubectl delete lblistener.elbv2.aws.upbound.io/${NAME}-redir
kubectl delete lbtargetgroup.elbv2.aws.upbound.io/${NAME}
kubectl delete lb.elbv2.aws.upbound.io/${NAME}
kubectl delete securitygroup.ec2.aws.upbound.io/${NAME}
