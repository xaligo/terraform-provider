---
applyTo: ".github/instructions/manual/**"
---

# 07.16.04 XAL specification: Semantics

### Semantics

| Kind | Target tag | Behavior |
|---|---|---|
| `padding` | frame / container / col | Inner whitespace. Child layout starts `pad` pixels inward |
| `padding` | AWS group tags / unknown containers | **Added to** `defaultGroupTopInset(44)` / `defaultGroupSideInset(12)`. `pa-2` adds +16px below the header |
| `margin` | any child element | Read by the parent layout (`layoutStack` / `layoutRow`) and used as inter-sibling spacing (equivalent to CSS flex margin) |
