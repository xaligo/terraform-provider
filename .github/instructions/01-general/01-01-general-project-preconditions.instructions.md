---
applyTo: ".github/instructions/manual/**"
---

# 01.01 General: Project preconditions

## Product boundary

This repository develops the Go Terraform Provider for xaligo. It reads
Terraform configuration (`.tf`) and manages deterministic xaligo diagram
source (`.xal`) as a local Terraform resource.

- Implement the provider with HashiCorp Terraform Plugin Framework and protocol
  version 6. Do not start new work on Terraform Plugin SDKv2.
- `provider "xaligo" { export = "enable" }` is an explicit global generation
  gate. The only valid modes are `enable` and `disable`; provider configuration
  performs no file I/O by itself.
- A managed `xaligo_diagram` resource owns each output file. Its `Create` and
  `Update` operations generate `.xal` during `terraform apply` when the provider
  export mode is `enable`.
- Provider code must not invoke Terraform CLI commands, provision cloud
  resources, call cloud-provider APIs, or inspect Terraform state files.
- The required output artifact is `.xal`. Rendering SVG/PPTX remains xaligo's
  responsibility.
- Input interpretation must be conservative. Unsupported or unresolved
  Terraform constructs produce source-positioned diagnostics; they must not be
  silently invented, discarded, or presented as known runtime topology.

## xaligo compatibility boundary

`github.com/xaligo/xaligo` is the source of truth for the output language and
validation behavior. In particular, consult these upstream areas when changing
the generator:

- `.github/instructions/07-xal-spec/` for authoritative `.xal` syntax.
- `.github/instructions/11-diagram-creation/` for catalog IDs, AWS grouping,
  service scope, and diagram-authoring rules.
- `docs/src/xal/` and `docs/src/examples/samples/` for public documentation and
  executable examples.
- `etc/resources/aws/service-index.csv` and
  `etc/resources/aws/service-catalog.csv` for catalog identity.

The two repositories remain separately buildable and releasable. Integrate
through canonical `.xal` files and the public `xaligo validate` CLI boundary;
do not import xaligo `internal/...` packages, copy its parser/layout engine, or
depend on a sibling checkout in production code.

The default generated language profile is the canonical V1 envelope for broad
compatibility:

```xml
<xaligo version="1">
  <frames>
    <frame id="main" width="1280" height="720" item-size="32">
      ...
    </frame>
  </frames>
</xaligo>
```

Changing the default language version is a compatibility decision and requires
fixtures plus validation against the targeted xaligo release.

## Architecture preconditions

### Go composition and dependency rules

`internal/base.go` is the public application lifecycle boundary, and
`internal/router.go` is the sole composition root:

```text
main.go / cmd/provider / cmd/cli
  -> internal/base.go
       -> internal/router.go
            -> newController
                 -> newUsecase
                      -> newRepository

controller -> usecase -> repository -> entity
```

- Define every function called through the repository's root `internal`
  package by `main.go`, `cmd/provider`, or `cmd/cli` in `internal/base.go`.
  Executable entry points must not import internal subpackages directly.
- Construct and inject controllers, use cases, and repositories only in
  `internal/router.go`. Keep the `newController`, `newUsecase`, and
  `newRepository` composition functions there. `internal/base.go` initializes
  a router and runs it; it does not import controller, usecase, or repository
  packages. Do not create a separate configuration or dependency-injection
  package.
- Production dependencies flow only from controller to usecase, usecase to
  repository interfaces, and repository to entity packages. Entities never
  depend on an outer layer.
- Components in the same layer are peers and must remain independent. A
  repository must not construct, retain, or call another repository; a use
  case must not construct, retain, or call another use case; and a controller
  must not construct, retain, or call another controller.
- Coordinate multiple repositories inside one use case, and coordinate
  multiple use cases or controllers only at `internal/router.go`. Inject a
  callback or framework factory from the composition root when an adapter
  must expose another adapter to a framework.
- Assemble the Cobra root command in `internal/router.go`. Register CLI
  commands from concept-specific controllers; keep shared `version` and
  `serve` commands in `controller/common.go`, and conversion behavior in its
  own controller. Command controllers must not construct use cases.
- In `controller`, `usecase`, and `repository`, keep each public interface,
  private implementation, and constructor together in its concept-specific
  file. All non-component data structures belong in an `internal/entity/*`
  concept package, and every method receiver is named `rcvr`.
- Keep all Go tests under `test/` in a directory that mirrors the production
  package path. Architecture tests must enforce these composition and
  dependency rules.

Preserve a one-way provider and conversion pipeline:

```text
Terraform CLI
  -> Terraform Plugin Protocol v6
  -> Plugin Framework provider/resource adapter
  -> .tf source loader
  -> HCL syntax and source diagnostics
  -> normalized Terraform/infrastructure graph
  -> provider-specific XAL mapping profile
  -> canonical XAL document model
  -> deterministic XML serialization
  -> xaligo validation
```

- Parse HCL with HashiCorp's HCL parser. Do not parse Terraform with regular
  expressions or line-oriented string matching.
- Keep all Terraform Plugin Framework types in the provider adapter. The core
  converter must not import provider, resource, schema, or Terraform state
  types.
- Keep HCL parsing, Terraform semantic normalization, provider mapping, XAL
  modeling, and XML serialization as separate responsibilities.
- Keep the normalized graph provider-neutral. AWS-specific resource knowledge
  belongs in an AWS mapping profile, not in the parser or serializer.
- Filesystem and CLI I/O belong at the adapter boundary. Core conversion logic
  accepts explicit inputs and returns models, diagnostics, or bytes.
- Return contextual errors; do not panic for user input.

## Terraform input contract

- Treat `.tf` files as UTF-8 Terraform configuration source. `.tf.json`, saved
  plans, state files, CDKTF output, and OpenTofu-specific extensions require an
  explicit scope decision before support is claimed.
- For directory input, read eligible `.tf` files in deterministic lexical path
  order. Ignore `.terraform/`, VCS metadata, generated artifacts, state, and
  plan files.
- The provider is installed through Terraform's normal initialization flow,
  but the conversion core must not invoke `terraform init` or require other
  provider schemas, credentials, backend access, refresh, or network access.
- Preserve source filenames, line/column ranges, and Terraform addresses in the
  intermediate representation so diagnostics remain actionable.
- Resource addresses, including module paths and instance keys when statically
  known, are semantic identities. Display labels such as a `Name` tag are not
  identities.
- References visible in HCL traversals may establish candidate graph edges.
  Do not infer an edge only because two resource names or literal strings look
  similar.
- Module loading, `count`/`for_each` expansion, dynamic blocks, provider
  aliases, data sources, and expression evaluation must each have an explicit
  support policy and tests. Until implemented, retain what can be represented
  and emit a diagnostic for the unresolved portion.
- Never include sensitive values, credentials, private keys, tokens, or
  Terraform state contents in generated `.xal`, logs, diagnostics, or fixtures.

## Intermediate model and graph rules

- Represent resources, data sources, modules, containment, references, and
  diagnostics explicitly. Do not pass raw HCL nodes directly to XML templates.
- Distinguish structural containment from communication/traffic connections.
  A Terraform reference proves dependency, not necessarily network traffic.
- Prefer explicit resource attributes and documented provider semantics when
  deriving containment. If public/private subnet classification or scope is
  not provable, use a neutral group and emit a warning rather than guessing.
- Preserve unknown resource types as visible generic nodes when possible.
  Unsupported resources must not disappear silently.
- Sort nodes and edges by stable semantic keys so repeated conversion of the
  same input is byte-for-byte reproducible.

## Terraform-to-XAL mapping contract

- Maintain resource-type mappings as reviewable data or focused mapping
  components. Every mapping records the Terraform type, XAL representation,
  catalog identity if any, containment rules, edge rules, and fallback.
- Use AWS-specific group tags only for matching infrastructure semantics. Use
  `<generic-group>` or `<rectangle>` for neutral or unsupported concepts.
- Place AWS definitions under a deterministic `<aws-cloud>` hierarchy. Put
  global services directly under the cloud; put regional services and VPCs
  under a `<region>` only when the region is derivable from configuration.
  Within a VPC, preserve the VPC -> availability zone -> subnet containment
  order and use reviewed AWS-specific groups such as `<security-group>`.
- Classify a subnet as `<public-subnet>` or `<private-subnet>` only when its
  route-table association and a static route to an Internet gateway or a NAT
  gateway in the same VPC can be proven. Support routes declared inline on
  `aws_route_table` and as standalone `aws_route` resources. Otherwise retain
  a neutral group and emit a warning.
- Numeric `<item id="N">` values are xaligo catalog IDs, not Terraform
  resource identities. Give repeated or connected items a unique `name` or
  `ref` derived from the Terraform address.
- Group, rectangle, port, frame, `name`, and `ref` identities must be stable,
  non-empty, and unique in their XAL scope. Sanitize whitespace and dots while
  retaining a reversible address-to-XAL-ID mapping in memory for connections.
- Verify every emitted catalog ID against the catalog associated with the
  targeted xaligo version. Never assign an icon by visual similarity alone.
- A fallback must preserve the Terraform type and local name in a human-readable
  label and emit a warning identifying the missing mapping.

## XAL output contract

- Emit well-formed UTF-8 XML using the canonical `<xaligo version="1">`,
  `<frames>`, and identified `<frame>` hierarchy. Do not generate legacy root
  `<frame>` or `<frames>` documents.
- Use documented lowercase tag names, attribute names, and enum values. Escape
  XML text and attributes with an XML-aware serializer.
- Set `item-size="32"` explicitly unless a user option intentionally selects a
  different supported value, keeping geometry stable across xaligo runtimes.
- Without an explicit paper size, treat 1280×720 as the minimum generated frame
  size. Deterministically expand
  the frame and emit documented `row` ratios when hierarchy depth or density
  would otherwise leave a nested container without a positive render area.
- Provider-selected paper sizes establish a physical aspect ratio. If the
  calculated hierarchy is taller than the nominal 96 dpi paper frame, expand
  width and height together while preserving that ratio. Generate grid layout
  with XAL's native 12-column `<row>`/`<col span>` elements, and never squeeze
  semantic AWS hierarchy containers into columns that leave invalid content.
- Register every generated semantic element in one stable items map. Explicit
  Terraform `row` blocks reference that map with string keys because HCL does
  not permit bare provider-local expressions. Resolve Terraform addresses and
  generated XAL IDs, reject unknown or repeated keys, and allow one row to
  contain only unique siblings under the same semantic parent.
- Expose direct Terraform resource, data-source, and module addresses through
  the read-only `xaligo_items` data source as `map(string)`. Layout blocks must
  accept values derived from that map via variables, locals, comprehensions,
  conditions, and standard Terraform collection functions.
- Mirror every XAL V1 layout tag in provider configuration: `container`,
  `row`, and nested `col`. Preserve decimal geometry; validate the 12-column
  span budget, layout and alignment enums, positive ratios and dimensions, and
  non-negative gaps. Use a targeted `layout` block for shared attributes on
  generated semantic elements without changing their containment.
- Connections must be direct frame children or children of a frame-level
  `<connections>` element. Every endpoint must resolve exactly once.
- Emit only relationships supported by the normalized graph. Terraform
  dependency edges should not be mislabeled as `traffic` without provider
  semantics proving communication direction.
- Output order, formatting, generated IDs, and warnings must be deterministic.
  Do not embed timestamps, host paths, random values, or machine-specific data.
- Write output atomically. On an error, do not replace a previously valid
  destination with a partial document.

## Diagnostics and provider behavior

- Syntax errors are fatal and include source filename, line, and column.
- Unsupported but recoverable constructs produce warnings and visible fallback
  nodes where possible.
- Ambiguous identity, containment, catalog, or connection resolution is an
  error unless a documented deterministic fallback exists.
- Report errors and warnings through Terraform Plugin Framework diagnostics.
  Generated `.xal` goes only to the `xaligo_diagram` resource's selected output
  path.
- Stable diagnostic codes should be introduced before external tooling is
  expected to consume diagnostics programmatically.

## Quality gates

Every behavior change requires focused tests at the lowest useful layer.

- Parser tests cover valid HCL, malformed HCL, source ranges, and expressions.
- Graph tests cover stable addresses, containment, references, ambiguity, and
  unsupported constructs.
- Mapping tests cover known resources, catalog IDs, unknown-resource fallback,
  and duplicate display names.
- Serializer golden tests cover XML escaping, stable ordering, stable IDs, and
  canonical envelope output.
- Integration fixtures convert representative Terraform directories and run
  `xaligo validate` against every generated `.xal` artifact. Complex layout
  fixtures also run `xaligo render` and require a non-empty output artifact.
- Provider acceptance tests exercise plan, apply, refresh, update, and destroy
  with isolated temporary directories and no cloud credentials.
- A determinism test converts the same input more than once and compares exact
  bytes and diagnostic order.
- Tests must not require cloud credentials, a remote Terraform backend, other
  providers, or network access. Acceptance tests may run a local Terraform CLI
  and the provider under test.

Before considering a conversion slice complete:

1. Relevant unit and integration tests pass.
2. `go test ./...`, `go build ./...`, and `git diff --check` pass once the Go
   module and commands exist.
3. Generated golden `.xal` fixtures pass the supported `xaligo validate` CLI.
4. Unsupported input is represented or diagnosed, never silently lost.
5. Repeated runs produce byte-identical output.

## Explicit non-goals unless separately authorized

- Managing cloud or remote infrastructure; this provider manages only local
  diagram artifacts.
- Invoking Terraform CLI recursively from provider code.
- Reading Terraform state to discover secrets or runtime-only values.
- Downloading providers or remote modules implicitly.
- Generating files as a side effect of provider `Configure`, validation, or
  schema discovery.
- Rendering SVG/PPTX inside this converter.
- Forking or embedding xaligo's parser, layout, routing, or renderer.
- Claiming complete support for every Terraform provider or resource type.
