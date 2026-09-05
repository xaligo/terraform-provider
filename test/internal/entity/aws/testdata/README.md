# Terraform AWS Provider definition snapshot

`terraform-provider-aws.json` is a deterministic inventory derived
from HashiCorp's `terraform-provider-aws` release `v6.63.0`, commit
`07c0e849a5d45731848cc9b10eab557cbc141d76` (MPL-2.0).

The inventory reads `TypeName` values from generated
`internal/service/*/service_package_gen.go` methods for resources, data
sources, list resources, ephemeral resources, and actions. Provider function
names come from `internal/function/*` metadata. SDKv2 and Plugin Framework
entries are deduplicated by definition kind and type name.

The snapshot deliberately contains names and owning service only. It does not
copy provider implementation code, invoke the provider, require AWS
credentials, or perform network access during tests.

Run `make aws-definitions-update` to download the latest official release
archive and atomically regenerate this snapshot together with
the kind/service files below `internal/entity/aws`. Run
`make aws-definitions-check` to verify generated code and detect a newer
upstream release without modifying files. `GITHUB_TOKEN` is optional and can
be supplied to raise GitHub API rate limits. A new release intentionally fails
the pinned version, count, and SHA-256 assertions until its generated diff has
been reviewed.
