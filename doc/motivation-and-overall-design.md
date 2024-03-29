# Motivation

## Creating an End-to-end Network Datapath

A typical gateway/ingress controller for Kubernetes implements a
datapath inside Kubernetes, e.g. a Kubernetes `Deployment`,
`HorizontalPodAutoscaler` and `Service`. The `Service` may be of type
`LoadBalancer` which could result in a load-balancer service being
allocated from the surrounding cloud infrastructure.

However, there are several other concerns that impacts the network
datapath at the edge of the infrastructure:

- DNS. Ensuring that traffic are routed to our network datapath based
  on DNS lookups.

- TLS certificates bound to the DNS name(s) of the network datapath.

- Network connectivity, e.g. should the datapath be generally exposed
  to the internet or potentially internal subnets without public
  connectivity.

- ACLs limiting access to public datapaths based on IP addresses.

- Adaptive and 'deep inspection' type of filtering and protection -
  aka. 'web application firewall' solutions.

- Logging of network traffic.

- Load-balancer type, e.g. we may want to use an 'AWS application
  (ALB)' load balancer for some datapaths and an 'AWS API gateway' for
  others (an AWS API gateway is a server-less solution whereas an ALB
  is a 'running instance' type of solution with a non-zero idle-load
  cost).

- Multi-cluster and multi-region load balancing and traffic routing.

Similarly, inside the Kubernetes cluster, we may have services that
need a full service-mesh while others do not.

The following is two examples of network datapaths:

![Example network datapath](images/example-network-datapath.png)

## Using the Kubernetes Gateway API

When building a platform, it is essential to provide a well-designed
API to abstractions that are useful and manageable to users. The
*bifrost-gateway-controller* implements the [Kubernetes Gateway
API](https://gateway-api.sigs.k8s.io/) to achieve this objective.

The [Kubernetes Gateway API](https://gateway-api.sigs.k8s.io/) is an
API for describing network gateways and configure routing from
gateways to Kubernetes services. This API is fast becoming the
standard API and is [widely
supported](https://gateway-api.sigs.k8s.io/implementations/).

**The *bifrost-gateway-controller* presents the gateway API to users as
the sole interface for network datapath definition.** This is the only
interface users need to know and it fully supports a GitOps-based
workflow. Users do not need to work with Terraform or generally know
how the Gateway API is implemented by the platform. For features
beyond the core Gateway API, the *bifrost-gateway-controller* applies [GEP-713
policy attachments](https://gateway-api.sigs.k8s.io/geps/gep-713)

The Gateway API does not cover concerns such as DNS or web application
firewall (WAF) configuration. **The *bifrost-gateway-controller*
implements concerns beyond Gateway API scope using configuration in
`GatewayClass` resources.** E.g., a specific `GatewayClass` defines a
specific set of WAF rules.  This is very similar to how Kubernetes
[storage
classes](https://kubernetes.io/docs/concepts/storage/storage-classes)
map abstract storage claims to actual implementations.

A side-effect of using the Gateway API is that the
*bifrost-gateway-controller* interoperate well with other Kubernetes
solutions that automate networking, e.g. Canary deployments using
[Flagger](https://flagger.app).

## The *bifrost-gateway-controller* is a Controller-of-controllers

The *bifrost-gateway-controller* does not talk to any cloud-APIs to
implement the datapath. Instead it creates other Kubernetes resources
that implement the individual components - much like the Kubernetes
Deployment controller creates Pod resources and let another
controller implement Pod resources.

**One can think of the *bifrost-gateway-controller* as an advanced Helm
chart or a Crossplane composition.**

Because the *bifrost-gateway-controller* implements cloud resources
through other controllers and with resources configured through
`GatewayClass` resources, the *bifrost-gateway-controller* is
cloud-agnostic (but `GatewayClass` definitions are not).

Similarly, the *bifrost-gateway-controller* does not create datapaths
inside Kubernetes, e.g. the *service mesh gateway* shown in the image
above. These parts of the datapath are implemented by traditional
gateway controllers such as Istio, Contour etc.

The overarching purpose of the *bifrost-gateway-controller* is to
orchestrate the Kubernetes external and internal datapaths and this
complete datapath is configured using the Gateway API. This is
illustrated below using Crossplane for managing cloud resources and
Istio for managing the Kubernetes-internal datapath.

![Controller hierarchy](images/controller-hierarchy.png)

### Why Not Use e.g. Crossplane or Helm?

An important objective of the *bifrost-gateway-controller* is to maintain
a Gateway API compatible interface towards users. This would not be
possible with techniques such as Crossplane and Helm.  Also, the
mapping from the Gateway API to e.g. a Crossplane composition is
non-trivial, i.e. it is difficult to do with purely templating.

The *bifrost-gateway-controller* implements some of the same composition
logic as e.g. Crossplane. Why did we not use Crossplane compositions,
e.g. build a Gateway API implementation using the following hierarchy
of services?

- bifrost-gateway-controller implements facade gateway API, stamps out Crossplane claim.
- Crossplane implements claim towards a composition.
- Crossplane Composition defines how low-level cloud resources should be stamped out.
- Crossplane cloud-specific provider implements low-level resources towards cloud API.

While this would be feasible, there are several complicated
dependencies between each of the above components, which increase the
maintenance burden. The *bifrost-gateway-controller* design aims at
reducing the complexity by building on a more self-contained
monolithic design - or at least a design with less non-trivial
dependencies between components. For this reason, we use the basic,
low-level cloud resources of e.g. Crossplane and not the higher-level
composition functionality.

Another essential argument for keeping the composition logic internal
to the *bifrost-gateway-controller* is to support day-2 operational
concerns in the controller.  When operational concerns call for
updates that are non-trivial, e.g. where the order of operations
become important, a template-based solution is often not
sufficient. E.g. when we renew a TLS certificate, we want to create
the new certificate and associate it with our infrastructure before
discarding the old certificate.  Handling the composition internally
in the *bifrost-gateway-controller* allow us to implement such
operational concerns with dedicated code and thus without network
downtime.
