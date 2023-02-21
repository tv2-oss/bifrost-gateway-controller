# Creating GatewayClass Definitions

This document describes how to create new `GatewayClass`
definitions. See also [Example GatewayClass
Definitions](example-gatewayclasses.md) for the definitions provided
with the *cloud-gateway-controller*.

Before preparing new `GatewayClass` definitions, it is important to
understand the normalization implemented by the controller, since
`GatewayClass` definitions typically will use the normalized
specifications to define actual data path resources. See
[Normalization of Gateway Resources](normalization.md).

## `GatewayClass` and `GatewayClassParameter` Resources

The actual implementation of data-paths are defined by
`GatewayClassParameter` resources and the purpose of `GatewayClass`
resources is to name a given class, reference a
`GatewayClassParameter` resource and map the class to a specific
controller. See also [Gateway API on GatewayClass
documentation](https://gateway-api.sigs.k8s.io/references/spec/#gateway.networking.k8s.io/v1beta1.GatewayClass)

The `GatewayClassParameter` is a specific extension of this
controller.

A `GatewayClass` resource may look like the following. Note how we
specify our own controller name and a `default-gateway-class` resource
of kind `GatewayClassParameters` for parameters associated with the
`GatewayClass`:

```yaml
apiVersion: gateway.networking.k8s.io/v1beta1
kind: GatewayClass
metadata:
  name: kind-internal
spec:
  controllerName: "github.com/tv2/cloud-gateway-controller"
  parametersRef:
    group: v1alpha1
    kind: GatewayClassParameters
    name: default-gateway-class
```

A `GatewayClassParameter` contains templates for the sub-resource(s)
that implement the data-path. There are template(s) related to both
`Gateway` and `HTTPRoute` resources, and the general format is shown
below (with template details left out):

```yaml
apiVersion: cgc.tv2.dk/v1alpha1
kind: GatewayClassParameters
metadata:
  name: default-gateway-class
spec:

  # The following are templates used to 'implement' a 'parent' Gateway
  gatewayTemplate:
    resourceTemplates:
      # ... actual templates go here

  # The following are templates used to 'implement' a 'parent' HTTPRoute
  httpRouteTemplate:
    resourceTemplates:
      # ... actual templates go here
```

`Gateway` and `HTTPRoute` resources are handled independently.
I.e. when a `Gateway` resource is defined using a `GatewayClass` for
our controller, 'shadow' resources will be created using the
template(s) defined under `gatewayTemplate.resourceTemplates` in the
`GatewayClassParameters` associated with the given
`GatewayClass`. Similarly, 'shadow' resources will be created for
`HTTPRoute` resources using the templates under
`httpRouteTemplate.resourceTemplates`

Templates are Golang YAML templates (similar to e.g. Helm), and
includes support for the 100+ functions from the [Sprig
library](http://masterminds.github.io/sprig) as well as a `toYaml`
function.

TBD. More details on the templating format.


## Available Templating Variables

This section documents the variables that are available for templates
in `GatewayClassParameters`.

### Variables Available to `Gateway` Templates

The following structure is passed when rendering `HTTPRoute` templates:

```go
// Parameters used to render Gateway templates
type gatewayTemplateValues struct {
	// Parent Gateway
	Gateway *gatewayapi.Gateway

	// Union and intersection of all hostnames across all
	// listeners and attached HTTPRoutes. Particularly useful for
	// certificates since these are not port specific.
	HostnamesUnion, HostnamesIntersection []string
}
```

The `Gateway` field of the structure above holds the parent
`Gateway` and fields can be referenced in the template using
Go-struct field names as shown in the excerpt below:

```yaml
  metadata:
    name: {{ .Gateway.ObjectMeta.Name }}-child
    namespace: {{ .Gateway.ObjectMeta.Namespace }}
```

### Variables Available to `HTTPRoute` Templates

The following structure is passed when rendering `HTTPRoute` templates:

```go
type httprouteTemplateValues struct {
	// Parent HTTPRoute
	HTTPRoute *gatewayapi.HTTPRoute

	// Parent Gateway references. Only Gateways implemented by will be included
	ParentRef *gatewayapi.Gateway
}
```

The `HTTPRoute` field of the structure above holds the parent
`HTTPRoute` and fields can be referenced in the template using
Go-struct field names.

Note, that if the `HTTPRoute` is attached to multiple `Gateway`s
(which may be using different `GatewayClassParameters`), rendering of
the `HTTPRoute` will be done independently for each parent `Gateway`
the `HTTPRoute` is attached to. The `ParentRef` field will contain the
specific parent Gateway.
