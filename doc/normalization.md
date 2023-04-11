# Normalization of Gateway Resources

This document describes how Gateway API resource specifications are
normalized before any resources are created.

## Motivation

The *bifrost-gateway-controller* cannot simply create resources as plain
copies of the Gateway API resources defining a data path because there
is not necessary a 1:1 correspondence between these 'specifying'
resources and the resources that are needed to define the data path
such as TLS certificates.

In the figure below, an example is shown, where the three `HTTPRoute`s
that reference the `Gateway` specified hostnames. Depending on whether
the `Gateway` specify a hostname on listeners (indicated by the dashed
box), we have the following (non exhaustive) options for e.g. how to
configure DNS and certificate:

1. The `Gateway` defines a wildcard hostname, which leaves e.g. the
   following options:

  a. Use the wildcard domain in both DNS and certificate. This has the
     drawback, that filtering of e.g. invalid domains will not be handled
     at the load balancer and the cloud edge, but inside Kubernetes.

  b. Ignore the wildcard, only use the intersection of domain names
     from `Gateway` and `HTTPRoutes`. This has the benefit that
     illegal domain filtering will happen at the load balancer with
     the drawback that e.g. the certificate will be updated when new
     domains are added due to changes in `HTTPRoute`s.

2. The `Gateway` does not specify a domain name, hence we should
   compute the union of domain names from attached `HTTPRoute`s and use
   these for DNS and certificate.

In the example the `Gateway` does not allow `HTTPRoute`s from
namespace `ns3` hence this `HTTPRoute` must be ignored irrespective of
which solution above is chosen.

The `HTTPRoute`s from `ns1` and `ns2` are allowed by the `Gateway` so
collectively the data path will allow only the two domains associated
with those routes and only where the hostname on the `Gateway`
listeners [intersect (see gateway API
spec)](https://gateway-api.sigs.k8s.io/references/spec/#gateway.networking.k8s.io%2fv1beta1.Listener)
with the `HTTPRoute` hostname(s).

The *bifrost-gateway-controller* will normalize the 'defining resources'
such that it can (depending on the actual data path definition
specified in the applied `GatewayClass`) create Cloud resources like
TLS certificate and DNS entry as well as a Kubernetes-internal
`Gateway` using both alternative 1a) and 1b) shown above. Option 1b)
is illustrated below.

![Normalization of TLDs](images/normalization-tld.png)
