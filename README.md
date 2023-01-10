# Cloud Gateway Controller

The cloud-gateway-controller is an augmented Kubernetes network
gateway-controller -- think of it as a Kubernetes ingress-controller
that not only provides a data-path inside Kubernetes, but extends the
data-path outside Kubernetes into the surrounding cloud
infrastructure.

## End-to-end Network Path

xxx

When building a platform, it is essential to provide a well-designed
API to abstractions that are useful and manageable to users.

xxx

The [Kubernetes Gateway API](https://gateway-api.sigs.k8s.io/) is an
API for describing network gateways and configure routing from
gateways to Kubernetes services. This API is fast becoming the
standard API and is [widely
supported](https://gateway-api.sigs.k8s.io/implementations/).

## User Journeys

One of the principles driving the Gateway API was to support multiple
personas, i.e. design an API that has Kubernetes resources for each
persona. See e.g. the following example:

> ![Gateway-API personas](doc/images/gateway-api-personas.png)
(source: https://gateway-api.sigs.k8s.io/)

The following presents users journeys (user actions and corresponding
cloud-gateway-controller responses) and will use the personas
illustrated above.

- [Basic Network Datapath](doc/basic-datapath.md)
