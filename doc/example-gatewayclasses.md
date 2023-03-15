# Example GatewayClass Definitions

This document describe the example
`GatewayClass`/`GatewayClassBlueprint` definitions that are provided
as part of the *gateway-controller*.

## Contour and Istio

TODO: Text

This definition is provided in
[`gatewayclass-contour-istio.yaml`](../test-data/gatewayclass-contour-istio.yaml)
(without attached TLS certificate) and in
[`gatewayclass-contour-istio-cert.yaml`](../test-data/gatewayclass-contour-istio-cert.yaml)
(with attached TLS certificate).

TODO: Drawing, and description of template

## AWS ALB and Istio Using Crossplane

This definitions builds a data-path that consists of the following AWS
infrastructure:

- Application load balancer (ALB).
- Security group for ALB, together with ingress and egress rules (for
  both data and healthchecks).
- ALB target group and listener definitions.

This definition also includes the following Kubernetes infrastructure:

- A 'child' `Gateway` using the *istio* `GatewayClass`. This creates
  an Istio ingress gateway.
- `TargetGroupBinding` (an [AWS load balancer controller
  CRD](https://github.com/kubernetes-sigs/aws-load-balancer-controller/)
  for propagating Kubernetes endpoints for the Istio ingress gateway
  to the AWS ALB target group. This links the Kubernetes internal and
  external infrastructure.

This definition is provided in the following files:

- [`gatewayclassblueprint-aws-alb-crossplane.yaml`](../test-data/gatewayclassblueprint-aws-alb-crossplane.yaml) blueprint for infrastructure implementation
- [`gatewayclass-aws-alb-crossplane.yaml`](../test-data/gatewayclass-aws-alb-crossplane.yaml) definitions of `GatewayClass`es referencing the above `GatewayClassBlueprint`. Two `GatewayClass`es are created, one that is intended for internet exposed gateways, and one for non internet exposed gateways.
- [`gatewayclassconfig-aws-alb-crossplane-dev-env.yaml`](../test-data/gatewayclassconfig-aws-alb-crossplane-dev-env.yaml) settings for the two `Gateway`, i.e. with different subnet settings for the internet-exposed and non internet-exposed `GatewayClass'es.

TODO: Drawing
