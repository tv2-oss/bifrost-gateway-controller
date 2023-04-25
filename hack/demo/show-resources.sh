#! /bin/bash

kubectl get gateway,lbs,lbtargetgroups,lblisteners,securitygroups,securitygrouprules -A | sed -E 's#(arn:aws:elasticloadbalancing:eu-central-1:)[0-9]+(:[-0-9a-z\/]+)#\11234567890\2#'
