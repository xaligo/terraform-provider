---
applyTo: ".github/instructions/manual/**"
---

# 07.04 XAL specification: Numeric and Geometry Contract

## Numeric and Geometry Contract

Numeric attributes are validated before layout. A numeric value must be a
finite base-10 number; `NaN`, positive or negative infinity, an empty numeric
value, and malformed trailing text are errors. The current implementation
validates the source attributes and then reads those validated values during
layout; replacing the string attribute map with a typed normalized layout
specification is a separate roadmap item.

The following domain rules apply:

| Attributes | Required domain |
|---|---|
| `width`, `height`, `component-width`, `component-height`, `interface-width`, `content-width`, `content-height`, `item-size`, `font-size`, `key-width` | greater than `0` when specified |
| `row`, `col` | greater than `0` when specified |
| `span` | greater than `0` and at most `12`; flexible sibling spans in one `<row>` must total at most `12` |
| `gap`, `row-gap`, margins, spacing-class padding | greater than or equal to `0` |
| `scale`, `coordinate-scale`, `grid`, `stroke-width` | greater than `0` when specified |
| `x`, `y`, `dx`, `dy`, bend coordinates | any finite value, subject to the containing geometry rule |

An omitted attribute uses its documented default. An explicitly empty
`align` is treated as omitted; it must not produce an invalid-alignment warning.
Unknown non-empty enum values remain errors or source-positioned warnings as
specified by that attribute.

V1 intentionally distinguishes strict values from compatibility fallbacks:

| Input | V1 behavior |
|---|---|
| Invalid `overflow`, connection side, or connection anchor | Validation error |
| Unknown `layout`, connection `kind`, stroke style, arrowhead, or arrowhead-size value | Validation error |
| Unknown render mode, format, theme, paper/orientation, arrow-style option, or SVG legend position | Render-option error. Only `svg` and `pptx` are valid formats; retired aliases are not normalized |
| Recognized but unavailable render mode (`aws-2.5d` or `topology`) | Not-implemented error |
| Empty `align` | Omitted; defaults to `top-left` |
| Malformed or unknown non-empty `align` | Warning; each unsupported component keeps its `top` or `left` default |
| Unknown nested attribute or malformed/unrecognized spacing-class token | Ignored; a recognized numeric negative spacing class remains an error |

These fallbacks are part of V1 compatibility, not a mechanism for opting into
V2. The distinct V2 root prevents new V2 constructs from being silently
treated as V1 extensions.

`validate` and every render format use the same normalized values and resolved
geometry checks. Successfully validated input must not later produce `NaN`,
`Inf`, a negative drawable size, or an output serialization error caused
by geometry.
