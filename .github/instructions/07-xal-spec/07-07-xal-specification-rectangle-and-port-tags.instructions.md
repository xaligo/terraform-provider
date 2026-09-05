---
applyTo: ".github/instructions/manual/**"
---

# 07.07 XAL specification: `<rectangle>` and `<port>` Tags

## `<rectangle>` and `<port>` Tags

`<rectangle>` creates a general-purpose rectangle. Its label comes from
`title` or direct text content, and `font-size` controls the label size.
Unlike generic leaf tags, `<rectangle>` may contain multiple `<port>` children.

`<port>` creates a small rectangle inside a side of the parent rectangle.
Multiple ports on the same side are spaced evenly along that side. Its label
also comes from `title` or direct text content, and it supports `font-size`.

```xml
<rectangle id="service" title="Service" width="180" height="100" font-size="18">
  <port id="service-in" side="left" title="in" font-size="9" />
  <port id="service-out" side="right" title="out" font-size="10" />
</rectangle>
```

| Attribute | Target | Description |
|---|---|---|
| `id` | `rectangle`, `port` | Required connection reference ID |
| `width` / `height` | `rectangle`, `port` | Size in layout pixels |
| `title` / text content | `rectangle`, `port` | Text rendered inside the shape |
| `font-size` | `rectangle`, `port` | Text font size in layout pixels |
| `side` | `port` | Parent side: `top`, `right`, `bottom`, or `left`. Default `top` |
| `x` / `y` | `port` | Optional position relative to the parent rectangle's top-left corner. Values are clamped so the port remains inside the parent rectangle |

Port boxes must remain inside their parent rectangle. Explicit positions are
normalized before drawing, and overlapping ports on the same side are a layout
diagnostic rather than a renderer-specific accident. Port text carries the
shared text-layout policy, which SVG and PPTX enforce identically. Markdown
inherits the SVG result.
