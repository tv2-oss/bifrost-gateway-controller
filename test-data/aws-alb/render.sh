#! /bin/bash

export VPC_ID=vpc-0e4501e0fbc404284
export SUBNET1=subnet-03b44b0d7d83febbc
export SUBNET2=subnet-0844755379de6d6f8
export SUBNET3=subnet-0e6f13b058d3c3729

#export SG_ID=$(kubectl get securitygroup.ec2.aws.crossplane.io/aws-alb-test-shared -o jsonpath='{.status.atProvider.securityGroupID}')
#export ALB_ARN=$(kubectl get loadbalancer.elbv2.aws.crossplane.io/aws-alb-test -o jsonpath='{.status.atProvider.loadBalancerARN}')
export TG_ARN=$(kubectl get lbtargetgroup.elbv2.aws.upbound.io aws-alb-test -o jsonpath='{.status.atProvider.arn}')

chmod a+w render.yaml
cat aws-alb-upbound-provider.yaml | envsubst > render.yaml
chmod a-w render.yaml
