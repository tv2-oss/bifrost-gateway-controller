# Basic Network Datapath

This document describe user actions necessary to create a basic
datapath. The datapath will consist of an application hosted on the
domain `foo.example.com` and traffic is routed to the `foo-site`
Kubernetes Service as shown below. This example will be single
cluster, single namespace, ignore path-based routing. Also, the
`foo-store` service is not included in this example.

![Gateway API example](images/gateway-api-multi-namespace.png)
(source: https://gateway-api.sigs.k8s.io/)

The `foo` team SRE persona, or possibly a platform/cluster operator,
defines the network path using a `Gateway` resource and chooses the
`public-gw-gateway-class` class as the 'implementation':

```yaml
apiVersion: gateway.networking.k8s.io/v1beta1
kind: Gateway
metadata:
  name: foo-gateway
spec:
  gatewayClassName: public-gw-gateway-class
  listeners:
  - name: web
    port: 80
    protocol: HTTP
    hostname: "foo.example.com"
```

Additionally, developers define routing from the gateway to the
`foo-site` Kubernetes service with the following `HTTPRoute` resource:

```yaml
apiVersion: gateway.networking.k8s.io/v1beta1
kind: HTTPRoute
metadata:
  name: foo-site
spec:
  parentRefs:
  - kind: Gateway
    name: foo-gateway
  rules:
  - backendRefs:
    - name: foo-site
      port: 80
```

## Implementation by *cloud-gateway-controller*

The definitions above are generic Gateway API definitions. To
illustrate how the *cloud-gateway-controller* implements this datapath
we will assume the `GatewayClass` specified (`public-gw-gateway-class`)
defines an implementation with an Istio service-mesh inside Kubernetes
and ingress through an Istio ingress-gateway. Additionally we assume
an AWS cloud infrastructure managed through Crossplane. To simplify
the illustration some presented resources will have non-essential
information left out.

Note, that using Crossplane to provision the AWS load balancer may be
over-kill for this simple example. However, examples that use more
advanced cloud features will require an implementation that provide
the level of control that e.g. Crossplane allows. Multi-cluster load
distribution being one example of this.

### Creating Istio Ingress Gateway

To create the Istio ingress-gateway, the *cloud-gateway-controller*
create a copy of the `foo-gateway` `Gateway` resource specifying an
`istio` class instead of `public-gw-gateway-class`:

```yaml
apiVersion: gateway.networking.k8s.io/v1beta1
kind: Gateway
metadata:
  name: foo-gateway-istio       # Suffixed to avoid name conflict
spec:
  gatewayClassName: istio       # Note, different class name
  listeners:
  - name: web
    port: 80
    protocol: HTTP
    hostname: "foo.example.com"
```

Since Istio [also implements the Gateway
API](https://istio.io/latest/docs/tasks/traffic-management/ingress/gateway-api)
(albeit with a scope limited to the Kubernetes cluster) the result
will be a Kubernetes `Deployment` and `Service` named
`foo-gateway-istio`. This implements the Kubernetes-internal part of
the datapath.

Similarly, a copy is created of the `HTTPRoute` to configure the Istio
gateway. Note, that the gateway reference in `parentRefs` is changed
to the Istio gateway:

```yaml
apiVersion: gateway.networking.k8s.io/v1beta1
kind: HTTPRoute
metadata:
  name: foo-site-istio       # Suffixed to avoid name conflict
spec:
  parentRefs:
  - kind: Gateway
    name: foo-gateway-istio  # Configure routing in the Istio gateway
  rules:
  - backendRefs:
    - name: foo-site
      port: 80
```

### Creating Cloud Load Balancer

The cloud infrastructure resources are provisioned through Crossplane
using basic AWS resources. For this example, the `GatewayClass`
`public-gw-gateway-class` define an AWS ALB based network path exposed
to the Internet through public subnets. Thus, the
*cloud-gateway-controller* creates the AWS ALB using the following
resources. Note how the port and protocol are propagated from the
`Gateway` resource to the `Listener` resource:

```yaml
apiVersion: elbv2.aws.crossplane.io/v1alpha1
kind: LoadBalancer
metadata:
  name: foo-gateway
spec:
  forProvider:
    subnetMappings:
    - subnetID: public-subnet-1
    - subnetID: public-subnet-2
    - subnetID: public-subnet-3
---
apiVersion: elbv2.aws.crossplane.io/v1alpha1
kind: TargetGroup
metadata:
  name: foo-gateway
spec:
  forProvider:
    protocol: HTTP
---
apiVersion: elbv2.aws.crossplane.io/v1alpha1
kind: Listener
metadata:
  name: foo-gateway
spec:
  forProvider:
    port: 80
    protocol: HTTP
    loadBalancerArn: <inserted by cloud-gateway-controller when known>
    defaultActions:
    - actionType: forward
      targetGroupArn: <inserted by cloud-gateway-controller when known>
```

Note, that in the above resource, there are inter-resource
references. These will be filled out by the controller as they become
known. Also, the `GatewayClass` may have been configured with some
static values, e.g. the subnet IDs above.

The final link between the `TargetGroup` created above and the Istio
ingress-gateway Pods is the following `TargetGroupBinding`. This
resource is managed by the [AWS load balancer
controller](https://kubernetes-sigs.github.io/aws-load-balancer-controller)
and updates the AWS `TargetGroup` with the addresses of the Istio
ingress-gateway Pods.

```yaml
apiVersion: elbv2.k8s.aws/v1beta1
kind: TargetGroupBinding
metadata:
  name: foo-gateway
spec:
  targetGroupARN: <inserted by cloud-gateway-controller when known>
  targetType: ip
  serviceRef:
    name: foo-gateway-istio    # Load balancer targets are Istio ingress-gw Pods
```
