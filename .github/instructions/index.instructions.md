---
applyTo: "**"
---

# Instruction index

This index and `operations.instructions.md` are the only always-loaded files.
Read the general project preconditions for every repository task, then every
XAL parent and detail file matching generated output being changed. Re-evaluate
the selected files whenever task scope changes.

## Required general instructions

Read [Project preconditions](01-general/01-01-general-project-preconditions.instructions.md)
before any repository work. Follow `operations.instructions.md` for inspection,
editing, verification, and handoff.

Read [Terraform provider design](02-design/02-01-terraform-provider-design.instructions.md)
before changing Go code, provider schemas, lifecycle behavior, Terraform source
loading, generation behavior, state, or filesystem ownership.

## Required XAL baseline

Read these files for every change that can affect generated `.xal`:

1. [Overview](07-xal-spec/07-01-xal-specification-overview.instructions.md)
2. [V1 compatibility and version boundary](07-xal-spec/07-02-xal-specification-v1-compatibility-profile-and-version-boundary.instructions.md)
3. [Root tag](07-xal-spec/07-03-xal-specification-root-tag.instructions.md)
4. [Constraints and notes](07-xal-spec/07-19-xal-specification-constraints-and-notes.instructions.md)

## Task routes

| Task | Read in order |
|---|---|
| Any repository work | Project preconditions, then `operations.instructions.md` |
| Go code, provider schema, lifecycle, state, or filesystem behavior | Project preconditions, Terraform provider design, then `operations.instructions.md` |
| Terraform parsing, module loading, or expression analysis | Project preconditions, then Terraform provider design: Source-loading and conversion pipeline |
| Resource mapping or XAL serialization | Project preconditions, Terraform provider design: Mapping and generation, then all matching XAL specification sections |
| Canonical envelope, frames, pages, or metadata | 07.03, 07.03.01, and 07.03.02 |
| Numbers, geometry, sizing, or allocation | 07.04, 07.04.01, 07.08, and 07.17 |
| Containers, rows, columns, or custom nodes | 07.05, 07.05.01–03, and 07.06 |
| Rectangles or ports | 07.07 |
| Tables or relational databases | 07.09 or 07.10 respectively |
| UML generation | 07.11 and every applicable 07.11.01–10 detail |
| Items, spacers, or catalog-backed icons | 07.12 and 07.13 |
| Connections, endpoints, routes, or traffic | 07.14 and all four 07.14.01 parts |
| AWS groups or nested infrastructure layout | 07.15, 07.15.01, and 07.15.02 |
| Spacing classes, margin, or padding | 07.16 and 07.16.01–04 |
| Golden examples or end-to-end fixtures | All matching sections plus 07.18 |

## Snapshot provenance

The files under `07-xal-spec/` are copied verbatim from the sibling xaligo
checkout at commit `2333a3fff015ac3dd667037934b7eb5c4623cf8b`. Keep local
adaptation in this index or converter code; resynchronize specification changes
from upstream instead of editing the copied contract independently.
