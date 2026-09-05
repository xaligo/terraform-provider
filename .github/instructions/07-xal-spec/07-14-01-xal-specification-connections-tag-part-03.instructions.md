---
applyTo: ".github/instructions/manual/**"
---

07-14-01-xal-specification-connections-tag — part 3/4

Each `to <...>` / `from <...>` label is placed from the final inset terminal
with a 4-layout-pixel inward gap and a minimum 4-layout-pixel tangent gap.
Placement chooses the closest tangent position that avoids the endpoint
envelope and metadata reservation. Tiny pages clamp or shrink the label
fallback instead of moving it farther inward from that terminal.

Both internal scene stubs carry the same logical connector ID, original
endpoint/frame IDs, and V1 routing metadata so the shared plan can project
them consistently.

Default page-oriented export projects only the local stub belonging to each
frame: the source SVG/slide contains `to <destination frame
ID>`, and the destination one contains `from <source frame ID>`.
`--combine-frames` places both local stubs on the combined canvas or slide but
never reconnects them across the frame gap.

SVG and PPTX are projections of this resolved V1 meaning. They use native
constructs where available and must not add private, schema-breaking fields. A
V2 compatibility frontend consumes V1 directly and must never use an output
format as an intermediate representation.

When `src-side`, `dst-side`, `src-anchor`, and `dst-anchor` are omitted,
endpoint sides and anchor positions are calculated automatically from endpoint
geometry. Use `src-anchor` and `dst-anchor` to pin an endpoint to a specific
perimeter anchor. Cross-frame `src-frame-anchor` and `dst-frame-anchor` use the
same grammar to select the logical page side and tangent slot independently.
Each side has five positions at 10, 30, 50, 70, and 90 percent of the outer
frame extent, giving 20 unique tangent anchors. The drawable frame terminal
then moves only in the inward normal direction to the page-terminal inset line.
Corner anchors are not shared: `top-1` keeps a horizontal coordinate near the
left corner, while `left-1` keeps a vertical coordinate near the top corner.

```text
top:    top-1    top-2    top-3    top-4    top-5
right:  right-1  right-2  right-3  right-4  right-5
bottom: bottom-1 bottom-2 bottom-3 bottom-4 bottom-5
left:   left-1   left-2   left-3   left-4   left-5
```

Position numbers run left-to-right on `top` and `bottom`, and top-to-bottom on
`left` and `right`. Anchor positions are `1` through `5` from top/left to
bottom/right on the named side, inset from corners so each side owns its five
positions.
The aliases map one-to-one as `start=1`, `near=2`, `center=3`, `far=4`, and
`end=5`.

```xml
<connection src="web" dst="app"
            src-anchor="right-3"
            dst-anchor="left-3" />
<connection src="web" dst="app"
            src-side="right" src-anchor="3"
            dst-side="left" dst-anchor="3" />

<!-- The item and logical page terminal may use different sides. -->
<connection src="web" dst="detail.app"
            src-side="right" src-anchor="near"
            src-frame-side="bottom" src-frame-anchor="far"
            dst-side="left" dst-anchor="far"
            dst-frame-side="top" dst-frame-anchor="near" />
```

`src` and `dst` can also be expressed as child tags when the endpoint reference
and anchor should be declared together. The endpoint token can be tag text or
one of `id`, `ref`, `name`, or `target`.

```xml
<connection kind="traffic">
  <src anchor="right-3" frame-side="bottom" frame-anchor="far">web</src>
  <dst side="left" anchor="5" frame-anchor="top-2" ref="detail.app" />
</connection>
```

On `<src>` and `<dst>`, `frame-side` and `frame-anchor` map to the corresponding
source/destination connection attributes. A complete anchor such as
`bottom-4` supplies its side. With a separate side, slots accept `1..5` or the
aliases `start`, `near`, `center`, `far`, and `end`. Conflicting side and
complete-anchor values are validation errors for both endpoint and frame
anchors.

The current internal V1 compatibility scene stores the smallest legacy
arrowhead size (`"s"`) while the shared plan retains the logical arrowhead type
and style used by SVG and PPTX.

Manual bend coordinates are expressed as child tags in the same Cartesian
layout coordinate space as the frame, with the origin at the upper-left of the
rendered frame and positive `x`/`y` extending right/down. SVG and PPTX route
calculations keep the connector orthogonal while forcing the route through each
listed bend in order. The internal V1 compatibility scene retains the routing
metadata until the neutral V2 resolved model replaces it.

```xml
<connection src="web" dst="db"
            scale="1" grid="8">
  <bend x="120" y="80" />
  <bend x="120" y="220" />
  <bend x="300" y="220" />
</connection>
```

`<point>`, `<via>`, and `<waypoint>` are accepted aliases for `<bend>`.
Coordinates can also be grouped inside `<bends>`, `<points>`, or `<path>`.

Items and group tags may define a connection reference with `id`, `name`, or
`ref`:

```xml
<item id="1178" name="web" />
<item id="1189" name="db" />
<vpc id="prod-vpc" />
web --- db
web ==> db
prod-vpc --- web
```

- `---` expands to `kind="route"`.
- `==>` expands to `kind="traffic"`.
- Operands may also be numeric item IDs or group IDs.
- Explicit `<connection src=... dst=...>` attributes resolve the same way.
- Shorthands must be direct text children of `<frame>`.
- References must be unique and must belong to an item or group with a
  non-empty ID.
- Use an explicit `<connection>` for color, width, or stroke overrides, and for
  arrowhead overrides on normal connections or traffic flows. Routes remain
  headless.
