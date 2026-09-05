---
applyTo: ".github/instructions/manual/**"
---

# 07.11.08 XAL specification: Relation projection

### Relation projection

UML relations lower to the shared orthogonal connector model with the
following fixed semantic defaults:

| Projection | Relation kinds |
|---|---|
| Dashed line with destination triangle | `dependency`, `realization`, `return-message` |
| Solid line with destination triangle | `generalization`, `control-flow`, `object-flow`, `transition`, `message`, `create-message`, `destroy-message` |
| Source diamond | `aggregation`, `composition` |
| No destination arrowhead | `association` |

The visible relation label is placed near the routed connector midpoint and
the UML diagram/relation kind is retained as semantic metadata where the target
format can carry it.
