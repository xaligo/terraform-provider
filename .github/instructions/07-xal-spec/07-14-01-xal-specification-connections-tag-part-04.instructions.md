---
applyTo: ".github/instructions/manual/**"
---

07-14-01-xal-specification-connections-tag — part 4/4

**Arrow spec:**
- `elbowed: true` — always right-angle connectors.
- Arrowhead at end only by default; the shared plan records the logical
  SVG/PPTX head as `stealth`.
- Stroke color `#1e1e1e`, stroke width `1px` for normal connections
- `kind="route"` defaults to `#64748b`, `1px`, lower route layer, no arrowheads
- `kind="traffic"` defaults to `#2563eb`, `1px`, higher traffic layer, directional end arrowhead
- A traffic line with the same endpoints as a route line is drawn beside that
  route in SVG and PPTX draw paths when possible.
- Start/end connect to the **edge midpoint** of the element
  - When direction is **downward**: label text element (`{id}-item-lbl`) bottom edge
  - Otherwise: icon image element (`{id}-item`) corresponding edge
- The temporary V1 compatibility scene uses normalized fixed points and stable
  connector IDs before shared-plan projection.
- Item icons and labels use a 5x5 anchor grid.
  Anchor grid cells are drawn above connectors and below the item content so
  lines do not cover icons/labels while endpoints remain visible.
- Shared routing uses previously placed lines to offset exact or near-exact
  lane overlaps. Group header tags, item icons, and labels are treated as
  routing obstacles where possible.
- SVG/PPTX routing may additionally add automatic junction markers and line
  jump masks after shared route resolution. These are plan/rendering features,
  not extra `.xal` tags.

**Edge selection logic:**

| Direction (dst relative to src) | Start edge | End edge |
|---|---|---|
| Right (dx ≥ dy) | right | left |
| Left | left | right |
| Down (dy > dx) | bottom (label) | top |
| Up | top | bottom (label) |

> If `src` / `dst` items are not rendered, a warning is emitted and the connection is skipped.

Connection endpoints must resolve to exactly one item, AWS group, rectangle,
port, or identified child frame. Numeric catalog IDs are valid only when that
ID appears once in the document; when the same service appears multiple times,
use unique `name` or `ref` values. Missing endpoints, ambiguous numeric IDs,
duplicate aliases, and `<connection>` tags nested below any tag other than
`<frame>` or its direct `<connections>` child are validation errors.
