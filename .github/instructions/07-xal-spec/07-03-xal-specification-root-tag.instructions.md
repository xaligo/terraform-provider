---
applyTo: ".github/instructions/manual/**"
---

# 07.03 XAL specification: Root Tag

## Root Tag

```xml
<xaligo version="1">
  <data>
    <!-- reusable definitions -->
  </data>
  <frames gap="48">
    <frame id="overview" width="1440" height="900" class="pa-4">
      ...
    </frame>
  </frames>
</xaligo>
```

`<xaligo>` permits document-level metadata, imports, data, and styles, and
requires exactly one `<frames>`. Give every child `<frame>` a stable `id`.

```xml
<xaligo version="1">
<frames gap="48">
  <frame id="overview" width="1440" height="900">
    ...
  </frame>
  <frame id="detail" width="1440" height="900">
    ...
  </frame>
</frames>
</xaligo>
```

| Attribute | Type | Default | Description |
|---|---|---|---|
| `version` | string | `"1"` with warning when omitted | On `<xaligo>` or a legacy root, selects V1 and only `"1"` is accepted. On an identified direct child `<frame>`, a non-empty value is the visible page revision |
| `title` | string | — | On a page `<frame>`, enables the metadata band and supplies its built-in `title` tag |
| `width` | float | `1280` | Frame width (px) |
| `height` | float | `720` | Frame height (px) |
| `class` | string | — | Spacing class |
| `layout` | string | — | Set to `"horizontal"` to arrange children horizontally |
| `gap` | float | `16` | Gap between child elements (px) |
| `item-size` | float | render-context default, normally `32` | Max icon size (px) applied to all `<item>` elements in this file. Overrides the native `item.icon_size` or embedded asset-source value |
| `margin` / `margin-*` | float | — | DSL content whitespace in pixels. On root `<frame>`, paper-frame size is preserved and content is inset. This is separate from PPTX CLI `--paper-margin*` flags, which are inch-based export fitting margins |
| `content-width` / `content-height` | float | — | Shrink usable inner layout area |
| `align` | string | — | Align usable content area (`top|middle|bottom` + `left|center|right`) |
| `overflow` | string | `error` | Child containment policy: `error` or `visible` |

Legacy input may still use root `<frame>` or `<frames>`. It remains renderable,
but diagnostics recommend wrapping identified frames in the canonical
`<xaligo version="1"><frames>...</frames></xaligo>` envelope.

`<frames>` accepts `gap` and optional `layout="vertical"`. Without
`layout="vertical"`, frames are arranged horizontally. A `<frame>` inside
`<frames>` requires a non-empty `id`.
