# Contour and Istio

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
[`gatewayclassblueprint-contour-istio-values.yaml`](../../charts/bifrost-gateway-controller/ci/gatewayclassblueprint-contour-istio-values.yaml)
(RBAC for *bifrost-gateway-controller* Helm deployment suited for the `contour-istio` blueprint).
