#! /bin/bash

export ALB_DNS=$(kubectl get lb.elbv2.aws.upbound.io aws-alb-test -o jsonpath='{.status.atProvider.dnsName}')

curl $ALB_DNS:8080
