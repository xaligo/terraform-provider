---
applyTo: ".github/instructions/manual/**"
---

# 07.11.07 XAL specification: Relation attributes, order, and time

### Relation attributes, order, and time

Every relation requires `src` and `dst`. `title` or `label` supplies its
visible text. `guard` is allowed only on flows/transitions and is appended as
`[guard]`. `route` is retained as UML relation metadata for flow/transition
routing hints such as `route="loop"`. `src-multiplicity` and
`dst-multiplicity` are allowed only on association, aggregation, and
composition and are appended in
source-to-destination order. Relation color and normal connector side, anchor,
stroke-width, bend, scale, and grid attributes use the `<connection>` rules.
`kind`, `stroke-style`, `arrowhead`, `start-arrowhead`, and `end-arrowhead` are
invalid because the UML relation kind owns line and marker semantics.

Sequence message kinds require `order`. Its canonical form
is one or more positive decimal integers without leading zeroes, separated by
dots, for example `1`, `2`, or `1.1`. The complete order string must be unique
across all messages in one diagram. Numeric order is prepended to the rendered
label and assigns top-to-bottom connector anchors on participant/lifeline
shapes. It does not reorder declared elements or create activation boxes or a
separate interaction axis. Sequence message anchors always use a vertical
element edge so the ordering remains vertical: explicit `top` is normalized to
`left`, explicit `bottom` to `right`, and an explicit anchor slot is superseded
by the normalized order position.
