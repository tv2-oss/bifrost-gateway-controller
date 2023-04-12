![Bifrost logo](doc/images/bifrost-logo.png)

The *bifrost-gateway-controller* is an augmented Kubernetes network
gateway-controller -- think of it as a Kubernetes ingress-controller
that not only provides a data-path inside Kubernetes, but extends the
data-path outside Kubernetes into the surrounding cloud
infrastructure.

> **Status: Alpha**. The *bifrost-gateway-controller* is under active
> development and functionality and particularly CRDs may change.

- [Motivation and Overall Design](doc/motivation-and-overall-design.md)
- [Getting Started using a KIND Cluster](doc/getting-started.md)
- [Example GatewayClassBlueprints](blueprints/README.md)
- [Creating GatewayClass Definitions](doc/creating-gatewayclass-definitions.md)
- [Extending GatewayClass Definitions using Policy Attachments](doc/extended-configuration-w-policy-attachments.md)
- [User Journeys](doc/user-journeys.md)
- [About the Bifrost Name](doc/bifrost-name.md)

## TL;DR

The *bifrost-gateway-controller* is a cloud-agnostic solution for
orchestrating end-to-end datapaths through cloud and Kubernetes using
the [Kubernetes Gateway API](https://gateway-api.sigs.k8s.io/). An
example using AWS load-balancer and Istio is shown below. This is,
however, only one possible [GatewayClass
Definition](doc/creating-gatewayclass-definitions.md).

![Controller TL;DR](doc/images/controller-hierarchy.png)
