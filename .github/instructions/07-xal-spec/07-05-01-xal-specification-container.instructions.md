---
applyTo: ".github/instructions/manual/**"
---

# 07.05.01 XAL specification: `<container>`

### `<container>`

Stacks children **vertically** (same behavior as `frame`). Use `layout="horizontal"` for horizontal arrangement.

```xml
<container class="pa-4" gap="16">
  ...
</container>
```

| Attribute | Type | Default | Description |
|---|---|---|---|
| `layout` | string | — | `"horizontal"` to arrange children side by side |
| `gap` | float | `16` | Gap between child elements (px) |
| `content-width` / `content-height` | float | — | Shrink usable inner layout area |
| `align` | string | — | Align usable content area |
| `overflow` | string | `error` | Child containment policy: `error` or `visible` |
