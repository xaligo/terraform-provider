---
applyTo: ".github/instructions/manual/**"
---

# 07.06 XAL specification: Custom Leaf and Container Tags

## Custom Leaf and Container Tags

An otherwise unknown nested tag with no layout children is a generic leaf and
is rendered as a rectangle plus text. An unknown nested tag with layout
children is a generic group/container: it receives the normal group header
insets and lays out those children vertically by default, horizontally for
`layout="horizontal"`, or with the V1 staggered layout for
`layout="staggered"`. If every child is item-like (`item`, `spacer`, or
`blank`), the children use the item-grid row behavior instead.

This rule applies only below a valid document root. An unknown root is always a
parse error and can never be mistaken for a generic group.

```xml
<card title="Dashboard" />
<panel title="Main Chart" />
<text>Any label</text>
```

| Attribute | Behavior |
|---|---|
| `title` | Display label (takes priority) |
| Text content | Label when `title` is absent |
| (none) | Tag name used as label |
| `border` | Set to `"none"` to hide the border |
| `visible` | Set to `"false"` to hide only this component (border, icon, label). Children are still rendered individually. Layout space is preserved |
| `font-size` | Text font size in layout pixels |
