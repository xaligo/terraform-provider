---
applyTo: ".github/instructions/manual/**"
---

# 07.09 XAL specification: `<table>` Tag

## `<table>` Tag

`<table>` represents general tabular data and is distinct from the `<grid>`
layout primitive and relational `<entity>` definitions. It accepts either a
GFM-like pipe table or explicit `<header>`/`<row>` children containing
`<cell>` elements. One pipe header may be followed by explicit rows; a second
header is invalid. Both syntaxes normalize to the same typed rows before
layout.

```xml
<table id="services" title="Services">
  | Service | Role    | Port |
  |:--------|:--------|-----:|
  | API     | Backend | 8080 |
  <row><cell>DB</cell><cell>Database</cell><cell align="right">5432</cell></row>
</table>
```

Pipe separators require at least three hyphens per cell and use colons for
left, center, and right alignment. A literal pipe is escaped as `\|`. Every
header and row must have the same positive cell count. Tagged cell `align`
accepts `left`, `center`, `right`, or the normal vertical-horizontal values.
Unknown children, duplicate headers, malformed separators, and mismatched
column counts are positioned errors.

Table presentation attributes inherit from `<table>` to every row and cell.
Tagged `<header>`, `<row>`, and `<cell>` attributes override inherited values.
Pipe cells have no inline attributes, so style them with table attributes and
the `header-*` variants:

```xml
<table color="#172033" background-color="#ffffff" border-color="#94a3b8"
       font-family="nunito" font-size="16"
       header-color="#ffffff" header-background-color="#2563eb"
       header-font-family="cascadia" header-font-size="18">
  | Service | Port |
  |:--------|-----:|
  | API     | 8080 |
</table>
```

| Attribute | Values | Description |
|---|---|---|
| `color` | `#RRGGBB` or `transparent` | Cell text color |
| `background-color` | `#RRGGBB` or `transparent` | Cell fill color |
| `border-color` | `#RRGGBB` or `transparent` | Cell border color |
| `font-family` | `virgil`, `helvetica`, `cascadia`, `assistant`, `excalifont`, `nunito`, `lilita-one`, `comic-shanns`, or `liberation-sans` | Cell font family |
| `font-size` | positive number | Cell font size in layout pixels |
| `header-*` | corresponding value above | Pipe/tag header override declared on `<table>` |

Colors require six-digit hexadecimal notation. The style precedence is
`cell > header/row > table > built-in default`. Font-family names are carried
through the renderer-neutral scene and mapped to the corresponding output font
face; an output environment may substitute a missing installed font.
