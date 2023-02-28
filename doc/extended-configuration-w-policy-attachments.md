# Extending GatewayClass Definitions using Policy Attachments

This document describes how `GatewayClassBlueprint` associated with a
`GatewayClass` through the parameters reference, can be extended with
additional configuration while still supporting the [role-based model
of the gateway
API](https://gateway-api.sigs.k8s.io/#what-is-the-gateway-api).

Extending GatewayClass definitions follow the Policy Attachment
mechanism described in
[GEP-713](https://gateway-api.sigs.k8s.io/geps/gep-713). This
mechanism describes how CRDs containing extensions outside current
gateway API can be attached to standards-based gateway API and
Kubernetes resources such as namespaces. For our purpose, the
challenges are:

- Keep the role-based model intact while ensuring we cover usecases
  ranging from environment global settings under infrastructure
  provider control to Gateway specific settings under user/tenant
  control.

- Cover namespaced as well as cluster scoped resources
  (e.g. `GatewayClass`)

- Support precedence of settings as described in
  [GEP-713](https://gateway-api.sigs.k8s.io/geps/gep-713/#hierarchy).

- Cover situations when an infrastructure contain more than one
  `GatewayClass`.

The illustration below show the three categories of settings we
envision in the role-based model with yellow boxes being *policies*
attached to blue boxed with 'traditional' gateway API resources. The
three groups are:

- Environment global settings under infrastructure provider control.
- Tenant-specific parameters under infrastructure provider control.
- `Gateway` (and potentially `HTTPRoute`) specific settings under
  user/tenant control.

![Extension through policy attachment](images/policy-attachment.png)

## Policies are Generic Values for Templates in `GatewayClassBlueprint`s

The policy CRDs contain generic values that will be passed to the
templates stored in the `GatewayClassBlueprint` selected by a given
`Gateway` through a `GatewayClass`. Values will be subjected to the
precedence rules defined by GEP-713, but otherwise made directly
available to template rendering through a top-level `.Values` object
(much like Helm charts).

The keep the terminology from GEP-713, we will refer to these generic
values as *policies* even though the scope if broader than merely
*policies*.

## Using RBAC for Role-based Access

Two different CRDs will be used to effectively attach policies to
gateway resources, namely `GatewayClassConfig` and
`GatewayConfig`. Two CRDs will be used to easily separate access
rights for infrastructure providers and users/tenants.

## Namespaced vs Cluster Scoped Resources

Both `GatewayClassConfig` and `GatewayConfig` CRDs will be
namespaced. For `GatewayConfig` this is obviously because this is used
to configure namespaced `Gateway`s. However, `GatewayClassConfig`s are
namespaced because we need such resources to reference the specific
`GatewayClassBlueprint` which it configures and simultaneously contain
a namespace reference such that different users/tenants can have
configuration for the same `GatewayClassBlueprint`. An example
use-case for this is that infrastructure providers configure the
tagging rules for templated resources such as cloud load balancers and
the specified tagging will always be applied when a user/tenant
creates a `Gateway` using the specified `GatewayClassBlueprint`.

Hence, when the illustration above shows *tenant scope* it parallels a
Kubernetes namespace, i.e. infrastructure providers create
`GatewayClassConfig` definitions in user/tenant namespaces where
e.g. specific tagging rules are defined.

A special case is namespaced `GatewayClassConfig` definitions in the
*gateway-controller* namespace - these are considered as
infrastructure global and applies to `Gateway`s defined in any
namespace.

The *gateway-controller* merges values before rendering templates
using the following order of precedence (aka. as *hierarchy* in
GEP-713):

Increasing order of precedence:

- `GatewayConfig` in same namespace as `Gateway`

- `GatewayClassConfig` in same namespace as `Gateway` when `Gateway`
  reference `GatewayClassConfig` indirectly through `GatewayClass`.

- `GatewayClassConfig` in *gateway-controller* namespace when
  `Gateway` reference `GatewayClassConfig` indirectly through
  `GatewayClass`.
