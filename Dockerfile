# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY cloud-gateway-controller /usr/bin/cloud-gateway-controller
USER 65532:65532

ENTRYPOINT ["/usr/bin/cloud-gateway-controller"]
