---
applyTo: ".github/instructions/manual/**"
---

# 07.11.03 XAL specification: Component diagram sizing

### Component diagram sizing

Component boxes use automatic height by default. The resolved height reserves
the component header, one compact row for every declared boundary interface,
and additional vertical space when multiple incoming component associations
bind to the same destination interface. Interface groups are packed from the
header downward; unused diagram height is not redistributed into component
rows.

`component-width` and `component-height` on `<component-diagram>` set positive
diagram-wide defaults. `width` and `height` on an individual `<component>`
override the corresponding diagram default. When neither height attribute is
present, automatic height remains active. An explicit height is authoritative
and must be large enough for its interface and connection visuals to remain
inside the component.

A positive `interface-width` on `<component>` sets one common width for every
interface-name box in that component. It is not accepted on individual
`<interface>` children. The configured width must leave enough horizontal
space for the component's interface descriptions when they are present.
