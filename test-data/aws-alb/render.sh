#! /bin/bash

export VPC_ID=vpc-031e552470914799f
export SUBNET1=subnet-01314e279c1540a54
export SUBNET2=subnet-0763327f6870ebc5c
export SUBNET3=subnet-0c22e47fd072c6093

export SG_ID=$(kubectl get securitygroup.ec2.aws.crossplane.io/aws-alb-test-shared -o jsonpath='{.status.atProvider.securityGroupID}')
export ALB_ARN=$(kubectl get loadbalancer.elbv2.aws.crossplane.io/aws-alb-test -o jsonpath='{.status.atProvider.loadBalancerARN}')
export TG_ARN=$(kubectl get targetgroup.elbv2.aws.crossplane.io/aws-alb-test -o jsonpath='{.metadata.annotations.crossplane\.io/external-name}')

cat aws-alb.yaml | envsubst > render.yaml
