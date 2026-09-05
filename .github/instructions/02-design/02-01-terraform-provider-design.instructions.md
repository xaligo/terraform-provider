---
applyTo: ".github/instructions/manual/**"
---

# 02.01 Design: Terraform provider

## Status and authority

This file is the implementation precondition for the Go Terraform Provider.
It defines the initial public contract, lifecycle ownership, package boundaries,
and verification requirements. The copied `07-xal-spec/` files remain
authoritative for generated `.xal` syntax and behavior.

## Product contract

The provider type is `xaligo`. Generation is controlled by the provider's
`export` mode and materialized by a managed diagram resource:

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
}
```

`export = "enable"` is a global permission gate, not an imperative provider
operation. Set `export = "disable"` to suppress generation. A
`xaligo_diagram` resource is required for every managed output. Provider
configuration alone does not generate a file because it has no resource
identity or apply lifecycle, and `Configure` is also called during planning.

The initial provider manages local `.xal` artifacts only. It does not manage
cloud infrastructure and does not render SVG or PPTX.

## Technology decisions

- Language: Go, with the supported version pinned by `go.mod`.
- Provider SDK: `github.com/hashicorp/terraform-plugin-framework`.
- Protocol: Terraform Plugin Protocol version 6.
- Provider tests: `github.com/hashicorp/terraform-plugin-testing`.
- Terraform parser: HashiCorp HCL v2 packages; never regex-based parsing.
- XML output: an explicit XAL document model serialized with an XML-aware Go
  encoder.
- Dependencies and minimum Terraform versions are pinned when the Go module is
  initialized; do not rely on floating dependency versions.

Use the official Plugin Framework scaffolding conventions as the starting
point. Do not copy example domain behavior from the scaffold into production.

## Provider and resource schema

### Provider `xaligo`

| Attribute | Type | Behavior |
|---|---|---|
| `export` | string | Optional enum: `enable` or `disable`; defaults to `disable`. Permits managed `.xal` writes only in `enable` mode |

`export` must be known during provider configuration and must use the exact
lowercase value `enable` or `disable`. `Configure` validates the value and
passes immutable provider data to resources through
`ConfigureResponse.ResourceData`. It performs no source scan, conversion, file
write, file deletion, or xaligo invocation.

### Resource `xaligo_diagram`

The `xaligo_items` data source exposes every direct resource, data source, and
module as `map(string)` keyed by Terraform address. Layout expressions should
derive lists from this map through locals, variables, comprehensions, and
standard Terraform functions rather than duplicating address literals in each
layout block.

| Attribute | Type | Behavior |
|---|---|---|
| `source_dir` | string | Required Terraform source directory; normalize to an absolute, clean path |
| `output_path` | string | Required `.xal` path; a relative value resolves against `source_dir` |
| `frame_id` | string | Optional, defaults to `main`; must satisfy XAL frame identity rules |
| `title` | string | Optional frame title |
| `paper_size` | string | Optional, defaults to `screen`; supports `A5`, `A4`, `A3`, `A2`, `A1`, `Letter`, `Legal`, and `Tabloid` |
| `orientation` | string | Optional, defaults to `landscape`; `landscape` or `portrait` |
| `grid_columns` | number | Optional, defaults to `1`; divides XAL's 12-column grid into `1`, `2`, `3`, `4`, `6`, or `12` columns |
| `grid_gap` | number | Optional, defaults to `16`; non-negative generated row gap in pixels |
| `row` | repeatable block | Explicit XAL row using shorthand `items` or nested `col` blocks that address the generated global items map |
| `container` | repeatable block | XAL container built from sibling items, with all V1 shared container layout attributes |
| `layout` | repeatable block | Applies V1 `row`, `col`, dimensions, alignment, spacing, and layout attributes to one generated item |
| `fail_on_warning` | bool | Optional, defaults to `false`; promotes conversion warnings to errors |
| `overwrite` | bool | Optional, defaults to `false`; permits replacing a pre-existing unowned output |
| `id` | string | Computed stable resource identity derived from the normalized output path |
| `effective_export` | string | Computed provider export mode captured in plan and state |
| `source_sha256` | string | Computed digest of ordered input paths and bytes |
| `content_sha256` | string | Computed digest of the last successfully managed canonical `.xal` bytes |
| `observed_content_sha256` | string | Computed digest observed on disk during refresh; null when the output is absent |

Do not store complete source files or generated `.xal` content in Terraform
state. State contains only configuration values, stable identity, gate status,
and digests.

The initial XAL version is canonical V1 and is not a user-selectable resource
attribute. Adding a language-version option requires a separate compatibility
decision and golden fixtures for every supported version.

## Lifecycle contract

| Terraform phase | `export` | Required behavior |
|---|---:|---|
| Schema or validation | either | Validate values only; no filesystem mutation |
| Configure | either | Pass immutable provider data; no filesystem mutation |
| Plan / `ModifyPlan` | `enable` | Read and convert source, emit diagnostics, and set planned digests; do not write |
| Apply / Create or Update | `enable` | Reconvert, verify planned digests, and atomically write the `.xal` file |
| Plan or Apply | `disable` | Set `effective_export = "disable"`; do not create, replace, or delete output |
| Refresh / Read | `enable` | Preserve the managed digest, update the observed digest, and detect drift without rewriting |
| Destroy / Delete | either | Delete only an unchanged provider-owned output; preserve modified or unowned files with a warning |

`effective_export` is set by resource plan modification from configured
provider data. This ensures changing only the provider's `export` value creates
an observable resource plan instead of relying on an untracked `Configure`
side effect.

On a transition from `enable` to `disable`, preserve the existing output file
and stop managing drift. On a transition back to `enable`, regenerate the
expected content during Apply.

In `enable` mode, `ModifyPlan` sets the planned managed and observed digests to
the newly computed expected content. A source change, missing file, or differing
observed digest therefore produces a resource change. Create/Update sets both
digests to the bytes actually written. An externally modified file still
requires `overwrite=true`; drift detection must not silently clobber it.

If source bytes change between Plan and Apply, return an error before writing;
the practitioner must create a new plan. Planned and final state values must
remain consistent with Terraform's data consistency rules.

## File ownership and safety

- Canonicalize `source_dir` and `output_path` before use. Relative output paths
  resolve from `source_dir`, never from an implicit provider process directory.
- Require the output extension `.xal`.
- Refuse to follow an output symlink.
- On Create, refuse to overwrite an existing unowned file unless
  `overwrite=true`.
- On Update, replace an existing file only when it matches the digest recorded
  in prior state or `overwrite=true` is explicitly configured.
- Write a temporary file in the destination directory, sync and close it, then
  rename it over the destination so success is atomic on the target filesystem.
- Serialize concurrent writes by canonical output path. Do not use mutable
  process-global conversion state.
- On Delete, remove only a regular file matching the last managed digest. If it
  was edited externally, preserve it and report a warning while allowing state
  removal.
- Never read `.tfstate`, plan files, credentials, `.terraform/`, VCS metadata,
  or the generated `.xal` as conversion input.

## Source-loading contract

- The provider protocol does not supply another provider's full configuration
  to this provider. `source_dir` is therefore an explicit resource argument,
  normally set with `abspath(path.module)`.
- Read regular `.tf` files directly under the selected module directory in
  lexical path order for the initial release.
- Exclude `provider "xaligo"`, `xaligo_diagram`, and future `xaligo_*` blocks
  from the infrastructure diagram unless an option explicitly requests tool
  metadata.
- Do not invoke `terraform init`, `validate`, `plan`, `show`, providers, or
  cloud APIs from provider code.
- Local child-module traversal and remote-module cache traversal are not part
  of the initial release. Add either only with explicit cycle, boundary,
  identity, and reproducibility rules.
- Preserve source ranges and Terraform addresses through parsing and graph
  construction for diagnostics.
- Unknown expressions remain unknown. HCL traversals may establish dependency
  candidates, but dependency does not automatically mean network traffic.

## Conversion pipeline

```text
main.go / cmd/*
  -> internal/base.go (public application lifecycle boundary)
       -> internal/router.go (composition root and invocation routing)
            -> internal/controller (Cobra and Plugin Framework adapters)
                 -> internal/usecase
                      -> internal/repository
                      -> internal/entity

internal/repository/terraform.go  -- HCL source loading
internal/repository/aws.go        -- AWS mapping
internal/repository/xaligo.go     -- XAL serialization
internal/repository/path.go       -- path safety and stable IDs
internal/repository/artifact.go   -- atomic artifact lifecycle
```

- `internal/entity` contains subpackages only; do not place Go files directly
  below it. `internal/entity/common` owns diagnostics, parsed Terraform values,
  the provider-neutral infrastructure graph, and shared conversion/lifecycle
  DTOs. `internal/entity/xaligo` owns the canonical XAL document and Terraform
  Framework state models. `internal/entity/aws` and `internal/entity/local`
  contain focused provider mappings and local artifact values. Entity packages
  do not perform I/O.
- `internal/repository` groups each public repository interface with its private
  implementation and constructor in one concept-specific file. Implementations
  may import HCL, XML, or operating-system APIs.
- `internal/usecase` coordinates conversion plus the plan/apply/read/delete
  artifact lifecycle using only entities and repository interfaces.
- `internal/controller` owns Plugin Framework schema behavior, Terraform
  diagnostics adaptation, Cobra commands, and provider protocol serving.
- `internal/base.go` exposes every function used by executable entry points,
  initializes the appropriate router, and runs it. It contains no concrete
  layer construction and imports no internal subpackage.
- `internal/router.go` is the sole composition root. It defines
  `newRepository`, `newUsecase`, and `newController`, constructs concrete
  components, injects dependencies, dispatches combined invocations, and
  assembles the Cobra root from concept-specific controller commands. Do not
  create a separate `internal/config` package.
- Components do not depend on peers in the same layer. Repositories never
  construct, retain, or call repositories; use cases never construct, retain,
  or call use cases; controllers never construct, retain, or call
  controllers. Cross-component coordination belongs to `internal/router.go`,
  with framework factories or callbacks injected where required.
- In `controller`, `usecase`, and `repository`, expose interfaces, keep concrete
  implementation structs private, and construct them through `New...`
  functions. Name every method receiver throughout the project `rcvr`.

Dependency direction is one way. Provider-neutral domain and use-case values do
not depend on `terraform-plugin-framework`; framework-backed delivery state is
isolated in `internal/entity/xaligo`; mapping does not perform filesystem I/O;
the XAL serializer does not parse Terraform; file I/O does not understand HCL
or XAL semantics.

The backward-compatible combined executable entry point remains at `main.go`,
following the official provider scaffold. Focused entry points may also live at
`cmd/provider` and `cmd/cli`; all entry points delegate construction to
`internal/base.go`, which delegates initialization to `internal/router.go`;
entry points do not import internal subpackages directly. Keep all Go
tests below `test/`, mirroring the production
path (for example, `internal/usecase` is tested from `test/internal/usecase`).
Use external test packages so tests exercise published package boundaries.

## Mapping and generation rules

- Build a normalized graph before selecting XAL tags. Do not render directly
  from HCL blocks.
- Treat explicit Terraform references as dependency evidence only. Emit
  `kind="traffic"` only when a reviewed provider mapping proves communication
  direction; otherwise use a structural connection or omit the edge with a
  diagnostic.
- Use AWS XAL groups only when Terraform attributes prove the corresponding
  AWS scope. Unknown subnet visibility or service scope uses a neutral group.
- Verify numeric item catalog IDs against the supported xaligo catalog.
- Keep a generated `internal/entity/aws` inventory of every definition in the
  pinned HashiCorp AWS Provider release. Partition generated definitions by
  kind directory and by service file/variable; the root `aws` package provides
  the canonical combined lookup. A known managed resource or data source
  without a reviewed icon mapping has an explicit neutral rectangle policy;
  only types absent from the pinned inventory produce the generic fallback
  warning.
- Refresh the inventory through an offline-reviewable development command that
  records the upstream version and commit. Provider runtime code and tests must
  never download or invoke the AWS Provider. Scheduled CI detects new upstream
  releases and fails with the documented refresh command.
- Derive XAL identities from full Terraform addresses, not display names. Keep
  a reversible in-memory address map for connection endpoint resolution.
- Sort source files, nodes, attributes, children, connections, and diagnostics
  by documented stable keys. Identical input and options must produce identical
  bytes and diagnostic order.
- Emit the canonical XAL envelope and every applicable constraint from
  `07-xal-spec/`. Validate integration fixtures with `xaligo validate`.

## Diagnostics

Domain diagnostics contain a stable code, severity, summary, detail, source
filename, and range where available. The provider adapter converts them to
Framework diagnostics without losing source context.

Fatal conditions include malformed HCL, unsafe output paths, identity
collisions, ambiguous required containment, unresolved connection endpoints,
catalog mismatches in reviewed mappings, source changes between Plan and Apply,
and invalid generated XAL.

Recoverable unsupported constructs and generic resource fallbacks are warnings
unless `fail_on_warning=true`.

Never log complete source content, generated content, environment variables,
state, credentials, or sensitive attribute values.

## Verification strategy

### Unit tests

- Provider schema defaults, unknown/null handling, and configuration
  diagnostics.
- HCL parsing, source ranges, Terraform addresses, and expression traversals.
- Graph containment, dependency classification, identity stability, and cycle
  handling.
- AWS definition snapshot completeness, generated-registry parity, exact
  lookup, mappings, catalog IDs, generic fallbacks, and warning ordering.
- Canonical XML, escaping, stable ordering, duplicate identities, and golden
  `.xal` files.
- Atomic file writes, overwrite protection, symlink refusal, digest checks,
  safe delete, and same-path concurrency.

### Acceptance tests

Use `terraform-plugin-testing` with an explicit `TF_ACC` gate, a local provider
factory, local state, and one temporary directory per test. Tests must cover:

1. `export = "disable"` produces no file.
2. `export = "enable"` plus `xaligo_diagram` produces a valid `.xal` on Apply.
3. Plan and validation never write a file.
4. A second plan after Apply is empty.
5. Terraform source changes produce an Update and deterministic new digest.
6. Switching `export` between `disable` and `enable` controls generation
   predictably.
7. Missing output is detected as drift.
8. Destroy removes an unchanged managed file and preserves an externally edited
   file.
9. Existing unowned files require `overwrite=true`.
10. Generated output passes the supported `xaligo validate` CLI.

Acceptance tests must not contact a remote backend, download unrelated
providers, use cloud credentials, or manage cloud resources.

## Initial non-goals

- Generating an output from a provider block without a `xaligo_diagram`
  resource.
- Hiding provider-block-only generation inside `Configure`; that UX requires a
  separately approved Terraform action or wrapper-command design.
- Running generation during schema discovery, validation, Provider Configure,
  or Read.
- Evaluating Terraform with provider schemas or reading Terraform state/plan
  JSON.
- Resolving remote modules or downloading providers/modules.
- Supporting every Terraform provider with specialized mappings.
- Rendering SVG/PPTX or embedding the xaligo rendering engine.
- Watching files or running as a daemon.

## Official references

- [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework)
- [Provider implementation and Configure lifecycle](https://developer.hashicorp.com/terraform/plugin/framework/providers)
- [Resource configuration](https://developer.hashicorp.com/terraform/plugin/framework/resources/configure)
- [Resource Create lifecycle](https://developer.hashicorp.com/terraform/plugin/framework/resources/create)
- [Resource plan modification](https://developer.hashicorp.com/terraform/plugin/framework/resources/plan-modification)
- [Terraform provider scaffolding framework](https://github.com/hashicorp/terraform-provider-scaffolding-framework)
- [Provider acceptance testing](https://developer.hashicorp.com/terraform/plugin/testing/acceptance-tests)
