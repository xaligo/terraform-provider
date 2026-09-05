ARG GO_VERSION=1.25.8

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS provider-source

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# This target cross-compiles the provider for the host running Terraform. The
# resulting binary is exported into Terraform's local filesystem mirror by the
# Makefile; it is not run as a Compose service.
FROM provider-source AS provider-host-build

ARG PROVIDER_OS
ARG PROVIDER_ARCH
ARG PROVIDER_VERSION=0.1.0

RUN test -n "${PROVIDER_OS}" \
    && test -n "${PROVIDER_ARCH}" \
    && CGO_ENABLED=0 GOOS="${PROVIDER_OS}" GOARCH="${PROVIDER_ARCH}" \
      go build -trimpath \
      -ldflags="-s -w -X main.version=${PROVIDER_VERSION}" \
      -o /out/terraform-provider-xaligo .

FROM scratch AS provider-artifact

ARG PROVIDER_VERSION=0.1.0

COPY --from=provider-host-build /out/terraform-provider-xaligo /terraform-provider-xaligo_v${PROVIDER_VERSION}

# Keep an OCI image available for the standalone conversion CLI. Host-side
# Terraform still uses the native artifact above because provider plugins are
# launched as child processes by Terraform Core.
FROM provider-source AS provider-image-build

ARG TARGETOS
ARG TARGETARCH
ARG PROVIDER_VERSION=0.1.0

RUN CGO_ENABLED=0 GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" \
      go build -trimpath \
      -ldflags="-s -w -X main.version=${PROVIDER_VERSION}" \
      -o /out/terraform-provider-xaligo .

FROM scratch AS provider-image

ARG PROVIDER_VERSION=0.1.0

LABEL org.opencontainers.image.title="terraform-provider-xaligo" \
      org.opencontainers.image.description="Terraform-to-XAL provider and conversion CLI" \
      org.opencontainers.image.source="https://github.com/xaligo/terraform-provider" \
      org.opencontainers.image.version="${PROVIDER_VERSION}"

COPY --from=provider-image-build /out/terraform-provider-xaligo /terraform-provider-xaligo

ENTRYPOINT ["/terraform-provider-xaligo"]
CMD ["--help"]
