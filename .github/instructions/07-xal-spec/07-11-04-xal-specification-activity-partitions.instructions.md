---
applyTo: ".github/instructions/manual/**"
---

# 07.11.04 XAL specification: Activity partitions

### Activity partitions

An `activity-diagram` may group activity elements in swimlanes with direct
`<partition id="..." title="...">` children. A partition may contain only
activity elements allowed by the `activity-diagram` row above. Partition IDs
are diagram-local identifiers and must be unique. A nested element may repeat
`lane="partition-id"`; when present, it must match the enclosing partition.
Elements may also be declared directly under the diagram with
`lane="partition-id"`, but the referenced partition must exist.

`lanes="vertical|horizontal"` is accepted only on `activity-diagram`.
`theme="xaligo"` selects the supported activity swimlane visual theme. Other
diagram families reject `lanes` and `theme`.
