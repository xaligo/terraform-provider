---
applyTo: ".github/instructions/manual/**"
---

# 07.08 XAL specification: Resolved Text Layout

## Resolved Text Layout

Text has both a geometry box and a semantic role. Scene and plan construction
must preserve the resolved role, wrapping, fitting, clipping, line height, and
padding instead of making each encoder infer them from generated IDs.

Built-in defaults are:

| Role | Wrap | Fit | Overflow |
|---|---|---|---|
| group header | no | shrink | clip to text box |
| ordinary label | yes | shrink | clip to text box |
| item label | yes | shrink | clip to text box |
| port label | yes | shrink | clip to port box |
| connector label | yes | shrink | clip to text box |

The default line-height multiplier is `1.2` unless the source scene carries a
valid positive value. Font sizes originate in layout pixels and are converted
with the same effective scale as the containing geometry. Changing
`--px-per-inch` or paper fitting therefore preserves the text-to-shape ratio.

An encoder may use native text fitting or deterministic line breaking, but the
visible result must obey the resolved policy. The temporary V1 compatibility
scene carries the same `xaligoTextLayout` metadata and must not become a
separate layout authority. Encoders apply text policy in this order: resolve
padding, wrap when enabled, shrink when requested, then clip when
`TextLayout.overflow="clip"`.
