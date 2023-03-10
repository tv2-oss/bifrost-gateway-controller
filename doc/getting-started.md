# Getting Started using a KIND Cluster

This guide describes how to exercise the *gateway-controller*
using a KIND cluster, i.e. this guide does not require a cloud
account. Instead, some resources that typically are allocated from a
cloud provider are instead simulated inside the KIND
cluster. Specifically:

- An `ingress` resource is used to simulate load-balancer allocation. [Contour](https://projectcontour.io) is used to implement the `ingress` resources.

- Cert-manager is used together with a cluster-internal CA to issue certificates.

- External-dns is used with a cluster-internal CoreDNS deployment to handle DNS.

The cluster-internal datapath is implemented using Istio.

## Prerequisites

The following is some of the prerequistites needed to build and run
the getting-started guide.

- Docker
- [KIND](https://kind.sigs.k8s.io)
- `kubectl`, `kustomize`, `make`, `helm`

For building the controller etc. (developers):

- Go (see [`go.mod`](go.mod) for version)
- [GoReleaser](https://github.com/goreleaser/goreleaser)
- Docker, with buildx
- KubeBuilder and associated tooling

## Deploy KIND Cluster and Dependencies

Deploy the KIND cluster and add gateway-API with:

```
make create-cluster deploy-gateway-api
```

Deploy Istio control-plane, Contour and external-dns+local CoreDNS and
cert-manager with:

```
make deploy-istio
make deploy-contour deploy-contour-provisioner
make setup-external-dns-test
make deploy-cert-manager
```

## Deploy gateway-controller

There are three alternative ways to deploy the controller as described
by the following sections.

### Deploy with Helm (recommended)

TODO: We do not yet have a Helm chart

### Deploy from Local-build and YAML Artifacts (recommended for end-to-end tests)

Deploying the controller using the generated manifests will
additionally test these, hence this is recommended for end-to-end
testing:

```
make build docker-build             # Build controller+container
make install                        # Install CRDs
make cluster-load-controller-image  # Load container into KIND
make deploy                         # Deploy from YAML manifests
```

### Run Controller Locally (recommended for development of the controller)

Running the controller locally is useful during development of the
controller:

```
make install   # Install CRDs
make run
```

### Deploy GatewayClass for KIND Datapath

```
kubectl apply -f test-data/gatewayclass-contour-istio-cert.yaml
```

## Create Datapath and Deploy Test Applications

As an example, we will implement the following usecase:

![Gateway-API example](images/gateway-api-multi-namespace.png)
(source: https://gateway-api.sigs.k8s.io/)

In this example, three different personas are used. A cluster
operator/SRE that manages a gateway in a `foo-infra` namespace and two
developer personas that manage two applications in `foo-store` and
`foo-site` namespaces.

To watch the progress and resources created, it can be convenient to watch for
resources with the following command:

```
watch kubectl get gateway,httproute,ingress,certificate,po,gatewayclass -A
```

### Cluster Operator/SRE

The cluster operator persona creates namespaces (and RBAC etc, but
that is out out-of-scope for this guide):

```
kubectl apply -f test-data/getting-started/foo-namespaces.yaml
```

The cluster-operator/SRE also creates the common `Gateway`:

```
kubectl apply -f test-data/getting-started/foo-gateway.yaml
```

### Developer of 'Site' Application

As a 'site' developer we deploy the 'foo-site' application and associated
routing into the `foo-site` namespace:

```
kubectl -n foo-site apply -f test-data/getting-started/app-foo-site.yaml
kubectl -n foo-site apply -f test-data/getting-started/foo-site-httproute.yaml
```

### Developer of 'Store' Application

As a 'store' developer we deploy the v1 and v2 versions of the 'foo-store'
application and associated routing into the `foo-store` namespace:

```
kubectl -n foo-store apply -f test-data/getting-started/app-foo-store-v1.yaml
kubectl -n foo-store apply -f test-data/getting-started/app-foo-store-v2.yaml
kubectl -n foo-store apply -f test-data/getting-started/foo-store-httproute.yaml
```

### Observations

In response to the `foo-gateway` `Gateway` resource, expect to see a
shadow `Gateway` called `foo-gateway-istio`. Also, expect to see Istio
respond to the `foo-gateway-istio` `Gateway` by creating an
ingress-gateway deployment. The PODs created for the Istio
ingress-gateway names will start with `foo-gateway-istio-`.

### Testing the Datapath

Test access to foo-site and foo-store applications (this uses ports
exported from KIND to your localhost):

```
curl --resolve foo.example.com:80:127.0.0.1 http://foo.example.com/site
curl --resolve foo.example.com:80:127.0.0.1 http://foo.example.com/store
```

Expect to see a `Welcome-to-foo-site`, `Welcome-to-foo-store-v1` and
`Welcome-to-foo-store-v2` being echoed.

Similarly, but using HTTPS through the cert-manager issued
certificate:

```
curl --cacert foo-example-com.crt --resolve foo.example.com:443:127.0.0.1 https://foo.example.com/site
curl --cacert foo-example-com.crt --resolve foo.example.com:443:127.0.0.1 https://foo.example.com/store
```

## Cleanup

```
make delete-cluster
```
