---
applyTo: ".github/instructions/manual/**"
---

# 07.04.01 XAL specification: Fixed and flexible child allocation

### Fixed and flexible child allocation

For a vertical parent, an explicit child `height` is a fixed main-axis size;
for a horizontal parent, an explicit child `width` is fixed. The parent first
reserves fixed sizes, margins, and gaps. Children without a fixed main-axis size
divide the remaining space using their positive `row` or `col` weights. A
`<row>` uses validated `span` values against its 12-column grid.

The resolved child size is the size used both for recursive layout and for
placing the next sibling. A child cannot replace its assigned size after the
parent cursor has advanced. Explicit cross-axis sizes must fit the parent's
content box unless overflow is explicitly allowed.

Layout parents accept `overflow`:

| Value | Behavior |
|---|---|
| `error` | Default. A child outside the parent's content box is a source-positioned validation error. |
| `visible` | The child may extend outside the content box, but all coordinates and sizes must remain finite and sibling cursors still use resolved sizes. |

The policy belongs to a parent and applies only to its direct children; it is
not inherited. If fixed children consume the full main axis under `visible`,
the parent's original usable extent is used as the flex pool and the flexible
children receive their weighted sizes while all children retain source order.
Sibling cursors use each resolved size, gap, and margin, making the resulting
overflow explicit. Under the default `error` policy the same layout is
rejected.

Overflow is never silently introduced by a renderer. Clipping is a drawing and
text policy and does not make invalid layout geometry valid.
