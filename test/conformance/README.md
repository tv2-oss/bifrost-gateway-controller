# Gateway API Conformance Test

The conformance test can be executed against a local KIND cluster,
which can be created with:

```
make setup-e2e-test-cluster
```

Run the controller external to the cluster with:

```
make run
```

Conformance tests ('core' parts or 'full' - see [Gateway API -
Conformance](https://gateway-api.sigs.k8s.io/concepts/conformance/))
can be executed with:

```
make conformance-test
make conformance-test-full
```
