# terraform-provider-xaligo

`terraform-provider-xaligo` reads Terraform configuration and manages a
deterministic [xaligo](https://github.com/xaligo/xaligo) `.xal` diagram source
as a local Terraform resource. It does not run Terraform recursively, read
state, contact cloud APIs, or render SVG/PPTX.

## Architecture

The implementation follows a one-way Clean Architecture dependency flow:

```text
main.go / cmd/*
  -> internal/base.go (application lifecycle boundary)
  -> internal/router.go (composition and CLI/provider routing)
  -> controller (Terraform Framework and concept-specific Cobra commands)
  -> usecase (conversion and artifact lifecycle)
  -> repository
  -> entity models

repository concepts:
  terraform.go  HCL source loading
  aws.go        AWS-to-XAL mapping
  xaligo.go     XAL serialization
  path.go       path safety and stable IDs
  artifact.go   atomic artifact lifecycle

test/internal/* mirrors internal/* and tests packages through public APIs
```

`internal/entity` has no root-level Go package. Shared diagnostics, Terraform
values, infrastructure graphs, and use-case DTOs live in
`internal/entity/common`; XAL documents and Terraform Framework state models
live in `internal/entity/xaligo`. Only constructor-backed implementation
objects remain in controller, use-case, and repository packages. Framework
behavior, HCL parsing, filesystem operations, and XML serialization remain in
their concept-specific implementations. Each layer exposes interfaces, keeps
implementations private, and all project method receivers use `rcvr`. The root
executable remains backward compatible, while
`cmd/provider` and `cmd/cli` provide focused entry points. No legacy
`internal/old` or `internal/config` compatibility layer is retained. Layer
components never depend on peers in the same layer; all concrete construction
and cross-controller wiring is owned by `internal/router.go`. The Cobra root is
assembled there from separate common and conversion command controllers.

## Terraform provider

```hcl
terraform {
  required_providers {
    xaligo = {
      source = "xaligo/xaligo"
    }
  }
}

provider "xaligo" {
  export = "enable"
}

resource "xaligo_diagram" "architecture" {
  source_dir  = abspath(path.module)
  output_path = "architecture.xal"
  title       = "Architecture"

  paper_size   = "A3"
  orientation  = "landscape"
  grid_columns = 3
  grid_gap     = 20

  container {
    id       = "application-services"
    items    = ["items.aws_s3_bucket.logs", "items.aws_lambda_function.worker"]
    layout   = "horizontal"
    gap      = 0
    align    = "middle-spread"
    overflow = "visible"
  }

  row {
    gap      = 20
    overflow = "visible"

    col {
      span  = 8
      class = "pa-2"
      items = ["items.application-services"]
    }

    col {
      span  = 4
      items = ["items.aws_sqs_queue.jobs"]
    }
  }

  layout {
    item   = "items.xaligo-aws-cloud"
    layout = "staggered"
    row    = 2
  }
}
```

To build layouts with normal Terraform expressions, load every infrastructure
address into one map and derive subsets with variables or `for` expressions:

```hcl
data "xaligo_items" "all" {
  source_dir = abspath(path.module)
}

variable "application_addresses" {
  type    = list(string)
  default = ["aws_lambda_function.worker", "aws_s3_bucket.logs"]
}

locals {
  items = merge(data.xaligo_items.all.items, {
    application_services = "application-services"
  })
  application_items = [for address in var.application_addresses : local.items[address]]
}

resource "xaligo_diagram" "from_map" {
  source_dir  = abspath(path.module)
  output_path = "architecture.xal"

  container {
    id     = local.items.application_services
    items  = local.application_items
    layout = "horizontal"
  }
}
```

`export` accepts only `enable` and `disable`, defaulting to `disable`.
Planning computes deterministic source and output digests without writing a
file. Apply writes only in `enable` mode. Destroy removes only an unchanged
provider-owned output.

## CLI

The provider executable also exposes an explicit conversion interface modeled
after xaligo's subcommand-based CLI:

```sh
terraform-provider-xaligo convert ./terraform \
  --output architecture.xal \
  --title "Architecture"

terraform-provider-xaligo version
terraform-provider-xaligo --help
```

A relative output path is resolved from the source directory. Conversion reads
only direct regular `.tf` files and does not initialize Terraform providers or
modules. Existing output requires `--overwrite`.

Generated frames retain the 1280×720 default for compact diagrams. For deeper
or denser hierarchies, the converter deterministically expands the frame height
and emits XAL `row` ratios so nested VPC, availability-zone, subnet, and
fallback groups keep a positive render area.

AWS resources are emitted under xaligo's AWS-specific cloud, region, VPC,
availability-zone, and reviewed service-group tags whenever their Terraform
relationships prove that placement. Subnets with an associated static route to
an Internet gateway or same-VPC NAT gateway become `public-subnet` or
`private-subnet`; ambiguous resources remain visible in neutral groups with a
diagnostic instead of being assigned an invented AWS topology.

`paper_size` selects the frame's physical aspect ratio at 96 dpi. When a deep
diagram needs more vertical space, the converter expands both dimensions while
preserving that ratio, preventing container padding from producing a
non-positive render area. `grid_columns` divides XAL's native 12-column system;
for example, three columns emit `span="4"`. Grid rows are applied to sibling
leaf nodes while semantic AWS containers retain their full-width hierarchy.

Every generated element is registered in one logical `items` map. A `row`
block selects entries from that map by Terraform address or generated XAL ID;
the optional `items.` prefix is accepted. Because Terraform does not allow a
bare expression such as `items.aws-cloud.xxx` inside a resource block, row
members use strings: `items = ["items.aws_s3_bucket.logs"]`. Selected entries
must be unique siblings so an explicit row cannot break AWS containment.

The provider covers all XAL V1 layout tags: `container`, `row`, and `col`.
`container` and `col` accept `layout`, `gap`, `class`, `align`, `overflow`,
`content_width`, `content_height`, `width`, and `height`; `col` also accepts
`span`. A `row` accepts `gap`, `overflow`, shorthand `items`, or explicit nested
`col` blocks. The `layout` modifier applies `row`, `col`, fixed dimensions, and
shared container attributes to an existing generated item. Decimal spans and
dimensions are preserved in XAL. Explicit spans may total at most 12; omitted
spans share the remaining columns.

## Local development

Only [MiniStack](https://www.ministack.org/) runs in Docker Compose. Terraform
runs on the host and reaches the emulator at `http://localhost:4566`.

```sh
cp .env.example .env # optional overrides
make ministack-up
```

Terraform providers are native plugin executables that Terraform starts as
child processes. Therefore the provider is built inside Docker, but exported
for the host OS and architecture into the repository-local Terraform mirror at
`samples/simple-vpc/terraform.d/plugins/`:

```sh
make provider-install
make provider-version
```

This keeps the Go toolchain in Docker while allowing the host Terraform process
to launch the provider normally. Contributors with the pinned Go toolchain can
use `make provider-install-local` instead. `make provider-image` also builds a
standalone OCI image for the converter CLI, but that image is not a Compose
service and is not used by host Terraform.

Run the sample lifecycle with the host Terraform executable:

```sh
make tf-init
make tf-plan
make tf-apply
make tf-destroy
```

`make sample` initializes and applies `samples/simple-vpc`, provisions its AWS
resources in MiniStack, and compares `generated.xal` byte-for-byte with
`expected.xal`. `make convert` runs only the conversion CLI.

MiniStack state is persisted in the `ministack-data` Docker volume. Stop it
with `make ministack-down`; use `docker compose down -v` only when intentionally
discarding the emulated AWS state.

`PROVIDER_VERSION`, `GO_VERSION`, `PROVIDER_IMAGE`, `HOST_OS`, and `HOST_ARCH`
can be overridden as Make variables. MiniStack defaults are listed in
`.env.example`.

## AWS Provider definition inventory

`internal/entity/aws` contains a generated registry of every action, data
source, ephemeral resource, provider function, list resource, and managed
resource exported by the pinned HashiCorp AWS Provider release. Known managed
resources and data sources without a reviewed xaligo catalog icon use an
explicit neutral rectangle policy; genuinely unknown definitions still emit
`MAPPING-W002`.

Generated definitions are partitioned first by kind directory and then by
service file and exported variable. For example, EC2 managed resources live in
`internal/entity/aws/resource/ec2_definitions_gen.go` as
`resource.Ec2Definitions`. The root `aws` package assembles these groups into
the stable lookup API.

The registry stores only definition names, owning services, and release
provenance. It neither embeds the AWS Provider implementation nor adds a
runtime dependency on its binary. To refresh the registry and its test
snapshot from the latest official release:

```sh
make aws-definitions-update
```

The update command streams the GitHub release archive, parses the generated
registration files, and atomically regenerates both artifacts. The weekly
`.github/workflows/aws-provider-definitions.yml` workflow fails when a newer
release is available. `make aws-definitions-generate` is an offline
re-generation command, while `make aws-definitions-check` verifies generated
code and checks the latest upstream release. `GITHUB_TOKEN` is optional for
local use. Version, commit, counts, and snapshot hash in the entity test are a
deliberate review gate; update those assertions only after inspecting an
upstream refresh diff.

## Terraform Public Registry release

The tag-triggered `.github/workflows/release.yml` workflow builds supported
platform archives, signs their checksum with GPG, and creates the GitHub
Release consumed by the Terraform Public Registry.

Publication requires the public GitHub repository to be named exactly
`xaligo/terraform-provider-xaligo`. Add the RSA public key to the `xaligo`
Public Registry namespace and configure these GitHub Actions repository
secrets:

- `GPG_PRIVATE_KEY`: ASCII-armored private key, including its BEGIN and END
  lines.
- `PASSPHRASE`: passphrase for that private key.

After registering the provider repository in the Public Registry, publish a
new immutable semantic version by pushing a tag such as `v0.1.0`. The workflow
uses the automatically provided `GITHUB_TOKEN`; an HCP Terraform API token is
not required for a public provider release.

## Development checks

```sh
go test ./...
go vet ./...
go build ./...
```

See [samples/simple-vpc](samples/simple-vpc) for an end-to-end local example.
