# AWS ALB and Istio Using Crossplane

This blueprint builds a data-path that consists of the following AWS
infrastructure:

- Application load balancer (ALB).
- Security group for ALB, together with ingress and egress rules (for
  both data and healthchecks).
- ALB target group
- ALB listener definitions for both terminating TLS (port 443) and redirecting HTTP (port 80) to HTTPS.

This definition also includes the following Kubernetes infrastructure:

- A 'child' `Gateway` using the *istio* `GatewayClass`. This creates
  an Istio ingress gateway.
- `TargetGroupBinding` (an [AWS load balancer controller
  CRD](https://github.com/kubernetes-sigs/aws-load-balancer-controller/)
  for propagating Kubernetes endpoints for the Istio ingress gateway
  to the AWS ALB target group. This links the Kubernetes internal and
  AWS infrastructure.
- Optional HorizontalPodAutoscaler
- Optional PodDisruptionBudget

**Note** the ALB terminates TLS and forwards traffic un-encrypted to
the Istio ingress gateway.

This definition is provided in the following files:

- [`gatewayclassblueprint-aws-alb-crossplane.yaml`](gatewayclassblueprint-aws-alb-crossplane.yaml) blueprint for infrastructure implementation
- [`gatewayclass-aws-alb-crossplane.yaml`](gatewayclass-aws-alb-crossplane.yaml) definitions of `GatewayClass`es referencing the above `GatewayClassBlueprint`. Three `GatewayClass`es are created, one that is intended for internet exposed gateways (`public`), one for internet exposed gateways but access limited by e.g. ACLs (`private`) and one for non internet exposed gateways (`internal`).
- [`gatewayclassconfig-aws-alb-crossplane-dev-env.yaml`](../../test-data/gatewayclassconfig-aws-alb-crossplane-dev-env.yaml) example settings for the three `GatewayClass`es defined in `gatewayclass-aws-alb-crossplane.yaml`, i.e. with different subnet settings for the internet-exposed and non internet-exposed `GatewayClass'es.
- [`gatewayclassblueprint-crossplane-aws-alb-values.yaml`](../../charts/bifrost-gateway-controller/ci/gatewayclassblueprint-crossplane-aws-alb-values.yaml)
RBAC for bifrost-gateway-controller Helm deployment suited for the `aws-alb-crossplane` blueprint.

## Compatibility

This blueprint use AWS Crossplane resources through the [Upbound AWS
Provider](https://marketplace.upbound.io/providers/upbound/provider-aws). The
following compatibility between this blueprint, Crossplane, Crossplane
Upbound AWS provider and Istio versions has been verified:

| Bifrost/Blueprint | AWS Provider | Crossplane | Istio | Status |
| ----------------- | ------------ | ---------- | ----- | ------ |
| `0.0.18` | `v0.28.0` | `v1.11.0` | `1.16.1` | :heavy_check_mark: |
| `0.0.18` | `v0.32.1` | `v1.11.0` | `1.16.1` | :x: |
| `0.0.18` | `v0.33.0` | `v1.11.0` | `1.16.1` | :heavy_check_mark: |
| `0.0.19` | `v0.33.0` | `v1.11.0` | `1.16.1` | :heavy_check_mark: |
| `0.0.20` | `v0.33.0` | `v1.11.0` | `1.17.2` | :x: (*) |
| `0.0.21` | `v0.33.0` | `v1.11.0` | `1.17.2` | :heavy_check_mark: |

(*) In Istio [1.17.0 Gateway naming convention was changed](https://istio.io/latest/news/releases/1.17.x/announcing-1.17/change-notes/) to be a concatenation of Gateway `Name` and `GatewayClass`.

## Testing AWS/Crossplane/Istio Blueprint

This section describes how to test the blueprint using different
version of the dependencies.

### Prerequisite

- A Kubernetes cluster.
- IAM roles for Crossplane to interact with AWS (see make target `deploy-crossplane-aws-provider`).
- IAM role for AWS load balancer controller (see make target `deploy-aws-load-balancer-controller`)
- A TLS certificate and associated domain name (see below).

Specifically these environment variables should be provided:

```
export CLUSTERNAME=
export AWS_LOAD_BALANCER_CONTROLLER_IAM_ROLE_ARN=
export CROSSPLANE_INITIAL_IAM_ROLE_ARN=
export CROSSPLANE_IAM_ROLE_ARN=
export DOMAIN=
export CERTIFICATE_ARN=
```

### Deploying Dependencies

Deploy dependencies with the make targets shown below. Version information can be left out to use default versions:

```bash
make deploy-gateway-api
make deploy-aws-load-balancer-controller-crds
AWS_LOAD_BALANCER_CONTROLLER_CHART_VERSION=v1.4.6  make deploy-aws-load-balancer-controller
CROSSPLANE_VERSION=v1.11.0                         make deploy-crossplane
CROSSPLANE_AWS_PROVIDER_VERSION=v0.33.0            make deploy-crossplane-aws-provider
ISTIO_VERSION=1.17.2                               make deploy-istio
```

Deploy controller and blueprint:

```
BIFROST_VERSION=0.1.6              make deploy-controller-aws-helm
BIFROST_BLUEPRINTS_VERSION=0.0.18  make deploy-aws-istio-blueprint
```

Note, there is also a `deploy-aws-istio-blueprint-local` make target to deploy
local repository blueprint version which is useful when developing
blueprints.

A `GatewayClassConfig` is also needed - because it is very environment
specific, this guide does not describe how to prepare it. Additionally,
a namespace-default `GatewayClassConfig` may be needed:

```bash
make deploy-namespace-gatewayclassconfig
```

Deploy the getting-started use-case:

```bash
GATEWAY_CLASS_NAME=aws-alb-crossplane-public make deploy-getting-started-usecase
```

Test the deployed data-path when resources are ready (use
e.g. `hack/demo/show-resources.sh` to observe status). Particularly
watch for an address on `foo-gateway`.

```bash
hack/demo/curl.sh $DOMAIN  # Where DOMAIN is as defined above
```

## Undeploying

```
make undeploy-getting-started-usecase
make undeploy-aws-istio-blueprint
make undeploy-controller
make undeploy-aws-load-balancer-controller
make undeploy-crossplane-aws-provider
make undeploy-crossplane
make undeploy-istio
```
