---
applyTo: ".github/instructions/manual/**"
---

# 07.03.02 XAL specification: Frame metadata tag band

### Frame metadata tag band

An identified page frame may expose `id`, `title`, a page-content `version`,
and arbitrary key/value entries as a two-cell tag band. The resolved metadata
`row-gap`, 4 layout pixels by default, is both the space between wrapped rows
and the metadata page-edge inset at the selected vertical edge and both
horizontal edges. Frame padding, content margins, and the content box never
replace or add to that inset. The band is enabled when the page frame has a
non-empty `title`, a child-frame content `version`, or a direct `<metadata>`
child. Existing identified frames that have only an `id` remain visually
unchanged. Once the band is enabled, non-empty built-ins are emitted in stable
`id`, `title`, `version` order, followed by `<entry>` children in source order.

```xml
<frame id="aws-architecture" title="AWS Architecture" version="1.0.0"
       width="720" height="400"
       margin-top="52" margin-right="24"
       margin-bottom="52" margin-left="24">
  <metadata position="top" align="right" font-family="helvetica">
    <entry key="owner" value="Platform Engineering" />
    <entry key="status" value="Approved" break-before="true"
           width="180" key-width="56" />
  </metadata>
  <rectangle id="diagram" title="Page content" />
</frame>
```

`<metadata>` is a non-layout direct child of a page `<frame>` and may occur at
most once. This context is distinct from document-level
`<xaligo><metadata>`. It contains only empty `<entry>` children; every entry
requires non-empty `key` and `value` attributes. Duplicate keys are retained.

| Attribute | Target | Default | Contract |
|---|---|---|---|
| `position` | `metadata` | `top` | Closed enum `top|bottom` |
| `align` | `metadata` | `left` | Closed enum `left|center|right`; applied independently to each resolved row |
| `font-family` | `metadata` | `virgil` | `virgil|helvetica|cascadia|assistant|excalifont|nunito|lilita-one|comic-shanns|liberation-sans` |
| `font-size` | `metadata` | `12` | Positive layout pixels; tag height is exactly `ceil(font-size * 1.2) + 4` |
| `color` | `metadata` | `#64748b` | Value text color; also key text color unless `key-color` is set |
| `key-color` | `metadata` | value of `color` | Key text color |
| `background-color` | `metadata` | `transparent` | Value-cell fill |
| `key-background-color` | `metadata` | `#f8fafc` | Key-cell fill |
| `border-color` | `metadata` | `#cbd5e1` | Cell border color; the cell stroke is fixed at `0.75` layout pixels |
| `width` | `metadata`, `entry` | auto | Positive total key/value tag width. An entry value overrides the metadata-level default |
| `key-width` | `metadata`, `entry` | auto | Positive key-cell width smaller than total width. An entry value overrides the metadata-level default |
| `gap` | `metadata` | `8` | Non-negative horizontal gap between tags |
| `row-gap` | `metadata` | `4` | Non-negative gap between wrapped rows and the same-sized inset from the selected top/bottom edge and both horizontal page edges |
| `break-before` | `entry` | `false` | Closed boolean `true|false`; `true` starts this entry on a new row when a preceding tag exists |

Colors use `#RRGGBB` or `transparent`. Auto width measures both cells with the
selected font and full-width-rune-aware metrics. Omit `width` or `key-width`
to request auto sizing; the literal string `auto` is not a V1 numeric value.
Fixed widths use no-wrap shrink-to-fit with clipping as the final overflow
guard. Tags preserve input order and use greedy left-to-right packing against
the usable width `frame.width - 2 * row-gap`, which produces the minimum row
count without reordering. The usable width must remain positive.
`break-before="true"` forces a row boundary before that custom entry. The
metadata `align` is then applied to each row separately against that same
usable width: left starts at `frame.x + row-gap`, right ends at
`frame.x + frame.width - row-gap`, and center still uses the frame center.

For `position="top"`, the band starts at `frame.y + row-gap`; for
`position="bottom"`, it ends at `frame.y + frame.height - row-gap`. The
metadata-side reservation strip spans the full frame width from the outer
logical frame edge to the corresponding boundary of the final content box.
Its depth is never less than `row-gap` plus the complete band height plus the
fixed 8-pixel content gap: if the normal content boundary is closer, it is
moved inward to that minimum; if it is already farther inward, it is retained.
Normal items and their text, local connector paths and labels, UML connector
paths and labels, and cross-frame page-link paths and labels cannot enter this
strip. `overflow="visible"` never overrides this page-decoration exclusion.
The frame's outer page size and invisible logical edge do not change.
The inset is measured from that logical frame edge before any common PPTX slide
centering and is unrelated to the export-only `--paper-margin*` options.
For a cross-frame page link, the same resolved `row-gap` is also the inward
normal inset of safe frame terminals on all four sides, independent of metadata
`position`; zero retains the outer edge. A frame without metadata instead uses
a 4-layout-pixel terminal inset. An explicit `src/dst-frame-side` or
`src/dst-frame-anchor` requires the inset to be strictly smaller than its
specified side's normal frame dimension: height for `top`/`bottom`, width for
`left`/`right`. Without an explicit frame terminal, validation requires at
least one side that satisfies this inset bound and the metadata reservation;
the shared scene later selects among those safe sides from rendered visual
geometry. The inset is never implicitly clamped.

The shared layout and presentation plan own this geometry. SVG and PPTX render
the owning frame's tags; per-frame projection cannot leak another page's band,
and combined output retains every band. Markdown inherits the SVG projection.
The entire reservation strip, rather than only the tag cells, is a hard
exclusion zone for normal rendered geometry.
