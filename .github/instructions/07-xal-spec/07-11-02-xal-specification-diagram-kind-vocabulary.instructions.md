---
applyTo: ".github/instructions/manual/**"
---

# 07.11.02 XAL specification: Diagram-kind vocabulary

### Diagram-kind vocabulary

| Diagram kind | Allowed elements | Allowed relations | Additional V1 semantic checks |
|---|---|---|---|
| `class-diagram` | `class`, `interface`, `enumeration` | `association`, `aggregation`, `composition`, `generalization`, `realization`, `dependency` | Requires one classifier. Aggregation/composition are class to class; generalization joins equal classifier kinds; realization is class to interface. |
| `component-diagram` | `component`, `interface`, `port`, `artifact` | `dependency`, `realization`, `association`, `assembly`, `delegation` | Requires one component. Realization is component to interface. Assembly requires at least one port endpoint. Delegation starts at a port. |
| `activity-diagram` | `initial`, `final`, `activity`, `action`, `decision`, `merge`, `fork`, `join`, `object-node` | `control-flow`, `object-flow` | Requires an activity/action. Control-flow excludes object-node. Object-flow requires an object-node endpoint. Initial/final direction and control-node degrees are validated. |
| `state-machine-diagram` | `initial`, `final`, `state`, `history`, `choice`, `fork`, `join` | `transition` | Requires a state. Initial/final direction and pseudostate degrees are validated. |
| `sequence-diagram` | `participant`, `lifeline` | `message`, `return-message`, `create-message`, `destroy-message` | Requires a participant/lifeline. Every message has a diagram-unique order. Create/destroy cannot be self messages. |

The endpoint contracts above are closed. An admitted relation with an endpoint
pair not described by its row is a validation error.
