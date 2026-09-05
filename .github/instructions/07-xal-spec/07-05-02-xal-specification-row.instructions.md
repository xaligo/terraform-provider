---
applyTo: ".github/instructions/manual/**"
---

# 07.05.02 XAL specification: `<row>`

### `<row>`

Lays out children **horizontally** in a 12-column grid.

```xml
<row gap="20">
  <col span="8">...</col>
  <col span="4">...</col>
</row>
```

| Attribute | Type | Default | Description |
|---|---|---|---|
| `gap` | float | `16` | Column spacing (px) |
| `overflow` | string | `error` | Child containment policy: `error` or `visible` |

> `<row>` is a **pure layout tag** — it does not render any border or label in the output.
> The `<col>` children are also pure layout containers.

An explicit child `width` is reserved before the grid share and is excluded
from `span` allocation. Among children without fixed width, an omitted `span`
defaults to `12 / number_of_flexible_children`; explicit flexible spans must
total at most `12`. Unused span leaves intentional trailing space.
