---
applyTo: ".github/instructions/manual/**"
---

# 07.14.01 XAL specification: `<connections>` Tag — part 1/4

### `<connections>` Tag

`<connections>` is an optional direct child of `<frame>` that groups
`<connection>` elements and provides shared defaults. It does not render a
shape or occupy layout space. Any per-connection attribute overrides the parent
default.

Only these non-empty group attributes are inherited:
`arrowhead-size`, `kind`, `color`, `stroke-width`, `width`, `stroke-style`,
`start-arrowhead`, `end-arrowhead`, `arrowhead`, `scale`,
`coordinate-scale`, and `grid`. Endpoint identity and geometry are deliberately
not inherited: every child must supply its own `src` and `dst`, and
`src-side`, `dst-side`, `src-anchor`, `dst-anchor`, `src-frame-side`,
`dst-frame-side`, `src-frame-anchor`, `dst-frame-anchor`, bends, points, and
via data remain child-local. Defaults are applied to a connection snapshot
during scene construction; the parsed child node is not mutated.

`stroke-width`/`width`, `end-arrowhead`/`arrowhead`, and
`coordinate-scale`/`scale` are semantic alias pairs. If a child supplies either
name, neither name is inherited from the parent. When the child itself supplies
both, the first canonical name in each pair takes precedence.

`<connections>` may contain only `<connection>` children. A misspelled or
otherwise unknown child is a validation error rather than a silently skipped
connector.

```xml
<connections kind="traffic" color="#2563eb" grid="8" scale="1">
  <connection src="web" dst="app" />
  <connection src="app" dst="db" color="#059669" />
</connections>
```

| Attribute | Type | Required | Description |
|---|---|---|---|
| `src` | string | ✓ | Catalog ID, or `id`/`name`/`ref` of the arrow start item, AWS group, rectangle, port, or identified child frame |
| `dst` | string | ✓ | Catalog ID, or `id`/`name`/`ref` of the arrow end item, AWS group, rectangle, port, or identified child frame |
| `src-side` / `dst-side` | string | — | Optional endpoint side: `top`, `right`, `bottom`, or `left` |
| `src-anchor` / `dst-anchor` | string | — | Optional edge anchor. Each side has five inset positions (`top-1` through `top-5`, etc.) for 20 unique perimeter anchors |
| `src-frame-side` / `dst-frame-side` | string | — | Cross-frame-only logical page side, independent of the endpoint side; the drawable terminal uses that side's inward inset line |
| `src-frame-anchor` / `dst-frame-anchor` | string | — | Cross-frame-only logical page side and tangent slot. Uses `top|right|bottom|left-1..5`, or a side plus numeric/named slot; inward inset does not change the slot coordinate |
| `arrowhead-size` | string | — | V1 fixed arrowhead size: `"s"` (small). This is the default; `m` and `l` are not V1 values because V1 cannot preserve them across all render formats |
| `kind` | string | — | `connection` for the normal connector, `route` for a structural path without arrows, or `traffic` for directional flow drawn beside a matching route |
| `color` | `#RRGGBB` | — | Six-digit hexadecimal stroke color override. Named, short, and alpha colors are invalid in V1 so every format preserves the same color |
| `stroke-width` / `width` | float | — | Positive stroke width override; `width` is the compatibility alias |
| `stroke-style` | string | — | `solid`, `dashed`, or `dotted` |
| `start-arrowhead` / `end-arrowhead` | string | — | Independently set either end to `none`, `arrow`, `triangle`, `stealth`, `diamond`, or `oval`. An effective `kind="route"` permits only `none` |
| `arrowhead` | string | — | Backward-compatible alias for `end-arrowhead`; an effective route permits only `none` |
| `bends` / `points` / `via` | string | — | Backward-compatible inline coordinate list. Prefer child tags for multiple bend coordinates |
| `scale` / `coordinate-scale` | float | — | Positive multiplier applied to bend coordinates before routing. Default `1` |
| `grid` | float | — | Positive per-connection snap grid in layout pixels. Defaults to the router grid |

Default connections and `kind="traffic"` use a thin 1px line with
`start-arrowhead="none"` and a slender `stealth` end arrowhead. `kind="route"`
uses `start-arrowhead="none"` and `end-arrowhead="none"` by default. Default
colors are `#1e1e1e` for normal connections, `#64748b` for routes, and
`#2563eb` for traffic. A route is always headless in V1: after applying
`<connections>` defaults and child alias overrides, any effective non-`none`
`start-arrowhead`, `end-arrowhead`, or `arrowhead` is a source-positioned
validation error. Explicit `none` is accepted. Explicit `stroke-width`, color,
and stroke style are preserved for every kind; non-route arrowhead attributes
are also preserved.

For SVG and PPTX plan output, the render option `arrow-style`
supplies the global arrowhead (and, for `thin`/`standard`, width) only when the connection
does not explicitly set that semantic value. Explicit DSL or inherited group
values take precedence, and `kind="route"` remains headless.

When a connection references endpoints in different frames, the shared scene
represents it as a page link instead of drawing one line across the inter-frame
gap. SVG and PPTX derive exactly two local stubs:

- the source stub runs from the source endpoint to the page-terminal inset line
  of its owning frame and has the exact label `to <destination frame ID>`; and
- the destination stub runs from the page-terminal inset line of its owning
  frame to the destination endpoint and has the exact label
  `from <source frame ID>`.

Angle brackets in those forms are rendered as literal punctuation. For a
connection from frame `overview` to frame `detail`, the visible strings are
therefore `to <detail>` and `from <overview>`.
