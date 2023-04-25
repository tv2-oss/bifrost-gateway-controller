# Example GatewayClassBlueprints

This document describe the example `GatewayClassBlueprint` definitions
that are provided as part of the *bifrost-gateway-controller*. Sample
`GatewayClass` definitions are also provided that reference the
blueprints.

## Contour and Istio

This blueprint builds a data-path that consists of the following
Kubernetes resources:

- A 'child' `Gateway` using the *istio* `GatewayClass`. This creates
  an Istio ingress gateway.
- An `Ingress` resource, which serves to 'simulate' a
  load-balancer. The `Ingress` resource use the ingress-class
  `contour` and forwards traffic to the Istio ingress gateway.
- A `Certificate` resource (a [cert-manager](https://cert-manager.io/)
  CRD) to allow termination of HTTPS through the ingress.

This definition is provided in the following files:

[`gatewayclassblueprint-contour-istio-cert.yaml`](gatewayclassblueprint-contour-istio-cert.yaml)
(with attached TLS certificate).
[`gatewayclassblueprint-contour-istio.yaml`](gatewayclassblueprint-contour-istio.yaml)
(without attached TLS certificate) and in
[`gatewayclassblueprint-contour-istio-values.yaml`](../charts/bifrost-gateway-controller/ci/gatewayclassblueprint-contour-istio-values.yaml)
(RBAC for *bifrost-gateway-controller* Helm deployment suited for the `contour-istio` blueprint).

## AWS ALB and Istio Using Crossplane

This blueprint builds a data-path that consists of the following AWS
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
  AWS infrastructure.

**Note** the ALB terminates TLS and forwards traffic un-encrypted to
the Istio ingress gateway.

This definition is provided in the following files:

- [`gatewayclassblueprint-aws-alb-crossplane.yaml`](gatewayclassblueprint-aws-alb-crossplane.yaml) blueprint for infrastructure implementation
- [`gatewayclass-aws-alb-crossplane.yaml`](gatewayclass-aws-alb-crossplane.yaml) definitions of `GatewayClass`es referencing the above `GatewayClassBlueprint`. Two `GatewayClass`es are created, one that is intended for internet exposed gateways, and one for non internet exposed gateways.
- [`gatewayclassconfig-aws-alb-crossplane-dev-env.yaml`](../test-data/gatewayclassconfig-aws-alb-crossplane-dev-env.yaml) example settings for the two `GatewayClass`es defined in `gatewayclass-aws-alb-crossplane.yaml`, i.e. with different subnet settings for the internet-exposed and non internet-exposed `GatewayClass'es.
- [`gatewayclassblueprint-crossplane-aws-alb-values.yaml`](../charts/bifrost-gateway-controller/ci/gatewayclassblueprint-crossplane-aws-alb-values.yaml)
RBAC for bifrost-gateway-controller Helm deployment suited for the `aws-alb-crossplane` blueprint.
