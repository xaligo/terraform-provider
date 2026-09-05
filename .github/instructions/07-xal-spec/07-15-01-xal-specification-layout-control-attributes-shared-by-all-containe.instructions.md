---
applyTo: ".github/instructions/manual/**"
---

# 07.15.01 XAL specification: Layout Control Attributes (shared by all containers)

### Layout Control Attributes (shared by all containers)

Available on `frame` / `container` / `col`, all AWS group tags, and unknown
child-bearing container tags where noted.

| Attribute | Value | Description |
|---|---|---|
| `layout` | `"horizontal"` | Arrange children **horizontally** with proportional widths (use the `col` attribute for ratio) |
| `layout` | `"staggered"` | Stack children with a depth offset (AWS group tags and unknown child-bearing containers) |
| `gap` | float | Child spacing (px). Default `16` |
| `align` | `"{vertical}-{horizontal}"` | Position of content area and `<item>` icons. Item grids also support `spread`. Default item-grid alignment is `"middle-center"` |
| `content-width` / `content-height` | float | Shrink usable inner layout area, leaving whitespace |
| `width` / `height` | float | Fixed child size (root frame dimensions remain the paper/content frame) |
| `overflow` | `"error"` \| `"visible"` | Child containment policy. Default `error` |

**`align` values** — combine a vertical part and a horizontal part with `-`:

| Part | Values |
|---|---|
| vertical | `top` \| `middle` \| `bottom` |
| horizontal | `left` \| `center` \| `right` \| `spread` |

All 12 combinations are valid: `top-left`, `top-center`, `top-right`, `top-spread`, `middle-left`, `middle-center`, `middle-right`, `middle-spread`, `bottom-left`, `bottom-center`, `bottom-right`, `bottom-spread`.

> **`center` (default):** icons are packed together and the group is centred within the available area
> (equivalent to CSS `justify-content: center`).
>
> **`spread`:** icons are distributed with equal gaps between each icon and the container edges
> (equivalent to CSS `justify-content: space-evenly`).
>
> **`left` / `right`:** icons are packed at the respective edge with a fixed `8 px` gap between icons.

```xml
<!-- Icons centred vertically and horizontally inside the group (default) -->
<private-subnet id="app-tier" title="App Tier" align="middle-center">
  <item id="27" />
  <item id="547" />
</private-subnet>

<!-- Icons spread evenly across the full width -->
<generic-group id="global-services" title="Global Services" align="middle-spread">
  <item id="1179" />
  <item id="1178" />
  <item id="216" />
  <item id="227" />
</generic-group>

<!-- Icons pinned to the top-left -->
<generic-group id="security-services" title="Security" align="top-left">
  <item id="216" />
  <item id="227" />
</generic-group>
```
