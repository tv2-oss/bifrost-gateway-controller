# Gateway-controller Helm Chart Changelog

## [UNRELEASED]

- Example text, add your PR info according to example below below this line. Do not bump chart version in Chart.yaml unless a chart release will be made following your PR.

## [0.1.9]

- Update securityContext for container to contain `readOnlyRootFilesystem: true` and `runAsNonRoot: true` ([#249](https://github.com/tv2-oss/bifrost-gateway-controller/pull/249)) [#kemv](https://github.com/kemv)

## [0.1.8]

- Add ServiceMonitor CRD to enable metrics endpoint discovery and configuration ([#202](https://github.com/tv2-oss/bifrost-gateway-controller/pull/202)) [@michaelvl](https://github.com/michaelvl)

## [0.1.7]

- Add HorizontalPodAutoscaler and PodDisruptionBudget resources to aws-crossplane blueprint and update Helm chart example values with RBAC for HPA and PDB. ([#186](https://github.com/tv2-oss/bifrost-gateway-controller/pull/186)) [@michaelvl](https://github.com/michaelvl)

## [0.1.8]

- Add json log support to helm chart ([#195](https://github.com/tv2-oss/bifrost-gateway-controller/pull/195)) [@michaelvl](https://github.com/michaelvl)
- Chart image tag default to Chart.appVersion instead of latest. ([#194](https://github.com/tv2-oss/bifrost-gateway-controller/pull/194)) [@michaelvl](https://github.com/michaelvl)

## [0.1.7]

- Added podAnnotations to chart, allowing users to set annotations for the controller pod. ([#189](https://github.com/tv2-oss/bifrost-gateway-controller/pull/189)) [@kemv](https://github.com/kemv)

## [0.1.6]

- Refactor chart release action. ([#143](https://github.com/tv2-oss/bifrost-gateway-controller/pull/143)) [@michaelvl](https://github.com/michaelvl)

## [0.1.5]

- Bump version as part of name change. ([#125](https://github.com/tv2-oss/gateway-controller/pull/125)) [@michaelvl](https://github.com/michaelvl)

## [0.1.4]

- Add issue templates and PR template. ([#121](https://github.com/tv2-oss/gateway-controller/pull/121)) [@michaelvl](https://github.com/michaelvl)

## [0.1.3]

- Helm chart corrections and test. ([#103](https://github.com/tv2-oss/gateway-controller/pull/103)) [@michaelvl](https://github.com/michaelvl)
