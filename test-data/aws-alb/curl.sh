#! /bin/bash

export ALB_DNS=$(kubectl get loadbalancer.elbv2.aws.crossplane.io/aws-alb-test -o jsonpath='{.status.atProvider.dnsName}')

curl $ALB_DNS:8080
