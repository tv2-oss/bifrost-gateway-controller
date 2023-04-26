# Installing *bifrost-gateway-controller*

## Validating Artifact Signatures

Container and Helm chart artifacts are signed using [cosign](https://github.com/sigstore/cosign).

Signature validation should always be done using artifact digest. You will find digests on the [container package page](https://github.com/tv2-oss/bifrost-gateway-controller/pkgs/container/bifrost-gateway-controller).

Container signature validation:

```bash
export IMAGE_DIGEST=sha256:406122a8226ded74417856178d58e63a8265dc64f74ee6cedd88d2a44bbf41c2
cosign verify --certificate-identity-regexp https://github.com/tv2-oss/bifrost-gateway-controller/.github/workflows/build-release.yaml@refs/.* \
              --certificate-oidc-issuer https://token.actions.githubusercontent.com \
			  ghcr.io/tv2-oss/bifrost-gateway-controller@$IMAGE_DIGEST | jq .
```

Helm chart signature validation:

```bash
export CHART_DIGEST=sha256:cf638567ae1954b6c345a911768dbf17d002f24487ca128332830a5640f9b72d
cosign verify --certificate-identity-regexp https://github.com/tv2-oss/bifrost-gateway-controller/.github/workflows/chart-publish.yaml@refs/.* \
              --certificate-oidc-issuer https://token.actions.githubusercontent.com \
			  ghcr.io/tv2-oss/bifrost-gateway-controller-helm@$CHART_DIGEST | jq .
```

## Installing the Controller Using Helm

Deploying the controller using Helm as follows. Note, that the Helm
chart only contain RBAC settings for the core Gateway API resources,
and any additional resources included in blueprints should be added as
shown below with `gatewayclassblueprint-contour-istio-values.yaml`.

```
helm upgrade -i bifrost-gateway-controller-helm oci://ghcr.io/tv2-oss/bifrost-gateway-controller-helm --version 0.1.6 --values charts/bifrost-gateway-controller/ci/gatewayclassblueprint-contour-istio-values.yaml -n bifrost-gateway-controller-system --create-namespace
```

> Note: Helm currently does not support specifying a digest as chart version.

In addition to the *bifrost-gateway-controller*, you will need
blueprints defining datapath implementations. See [Example
GatewayClassBlueprints](../blueprints/README.md).
