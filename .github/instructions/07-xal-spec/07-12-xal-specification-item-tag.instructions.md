---
applyTo: ".github/instructions/manual/**"
---

# 07.12 XAL specification: `<item>` Tag

## `<item>` Tag

A leaf element that places an AWS service icon inside a container.
Specify a positive signed 32-bit decimal ID from `service-catalog.csv` as the
`id` attribute (`1` through `2147483647`).
The icon is rendered to fit within the specified size (`item-size`).

The effective item size is resolved from the root `item-size` when present;
otherwise it comes from the render context. Native configuration and the
canonical embedded-asset profile default to `32` layout pixels. Callers that
provide a custom asset source may intentionally choose another value. For
cross-environment reproducibility, declare `item-size="32"` (or another fixed
positive value) on the document root.

```xml
<public-subnet id="public-subnet" title="Public Subnet">
  <item id="1178" />   <!-- with icon -->
  <item />             <!-- spacer: no icon, only a layout slot -->
  <item id="1189" />   <!-- with icon -->
</public-subnet>
```

| Attribute | Type | Required | Description |
|---|---|---|---|
| `id` | positive int32 | — | Decimal service ID `1..2147483647` from `service-catalog.csv`. Omitted or empty → treated as spacer; zero, signs, non-decimal syntax, and out-of-range values are invalid |
| `dx` / `dy` | float | — | Relative icon offset in pixels from the icon's normal layout `x,y` position. The moved icon rectangle must remain inside the parent frame/group border |

> If no icon is found for the given `id`, rendering skips the item and emits a warning rather than failing the document.
