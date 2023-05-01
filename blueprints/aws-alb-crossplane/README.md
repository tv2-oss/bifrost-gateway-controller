# AWS ALB and Istio Using Crossplane

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
- [`gatewayclassconfig-aws-alb-crossplane-dev-env.yaml`](../../test-data/gatewayclassconfig-aws-alb-crossplane-dev-env.yaml) example settings for the two `GatewayClass`es defined in `gatewayclass-aws-alb-crossplane.yaml`, i.e. with different subnet settings for the internet-exposed and non internet-exposed `GatewayClass'es.
- [`gatewayclassblueprint-crossplane-aws-alb-values.yaml`](../../charts/bifrost-gateway-controller/ci/gatewayclassblueprint-crossplane-aws-alb-values.yaml)
RBAC for bifrost-gateway-controller Helm deployment suited for the `aws-alb-crossplane` blueprint.

## Compatibility

This blueprint use AWS Crossplane resources through the [Upbound AWS
Provider](https://marketplace.upbound.io/providers/upbound/provider-aws). The
following compatibility between this blueprint, Crossplane, Crossplane
Upbound AWS provider and Istio versions has been verified:

| Blueprint | AWS Provider | Crossplane | Istio | Status |
| --------- | ------------ | ---------- | ----- | ------ |
| `0.0.18` | `v0.28.0` | `v1.11.0` | `1.16.1` | :heavy_check_mark: |
| `0.0.18` | `v0.32.1` | `v1.11.0` | `1.16.1` | :x: |
| `0.0.18` | `v0.33.0` | `v1.11.0` | `1.16.1` | :heavy_check_mark: |
| `0.0.19` | `v0.33.0` | `v1.11.0` | `1.16.1` | :heavy_check_mark: |

## Testing AWS/Crossplane/Istio Blueprint

This section describes how to test the blueprint using different
version of the dependencies.

### Prerequisite

- A Kubernetes cluster.
- IAM roles for Crossplane to interact with AWS (see make target `deploy-crossplane-aws-provider`).
- IAM role for AWS load balancer controller (see make target `deploy-aws-load-balancer-controller`)
- A TLS certificate and associated domain name (see below).

### Deploying Dependencies

Deploy dependencies with the make targets shown below. Version information can be left out to use default versions:

```bash
make deploy-gateway-api
make deploy-aws-load-balancer-controller-crds
AWS_LOAD_BALANCER_CONTROLLER_CHART_VERSION=v1.4.6  make deploy-aws-load-balancer-controller
CROSSPLANE_VERSION=v1.11.0                         make deploy-crossplane
CROSSPLANE_AWS_PROVIDER_VERSION=v0.28.0            make deploy-crossplane-aws-provider
ISTIO_VERSION=1.16.1                               make deploy-istio
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
CERTIFICATE_ARN=some-arn-for-foo.example.com make deploy-namespace-gatewayclassconfig
```

Deploy the getting-started use-case:

```bash
GATEWAY_CLASS_NAME=aws-alb-crossplane-public DOMAIN=foo.example.com make deploy-getting-started-usecase
```

Test the deployed data-path when resources are ready:

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
