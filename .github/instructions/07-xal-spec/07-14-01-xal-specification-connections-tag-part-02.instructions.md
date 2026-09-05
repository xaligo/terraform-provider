---
applyTo: ".github/instructions/manual/**"
---

07-14-01-xal-specification-connections-tag — part 2/4

Endpoint binding and logical frame-terminal geometry are separate. The
endpoint uses `src-anchor`/`dst-anchor`, then `src-side`/`dst-side`, then its
normal automatic binding. The logical page terminal uses
`src-frame-anchor`/`dst-frame-anchor`, then
`src-frame-side`/`dst-frame-side` as fixed choices. With neither frame-terminal
attribute, the legacy endpoint anchor, endpoint side, or normal nearest-border
result is only the preferred page side. The renderer keeps it when safe;
otherwise it chooses the nearest safe side from the endpoint's rendered visual
envelope. An item's envelope is the union of its icon and external label; other
endpoints use their rendered shape. Distance ties prefer a tied side facing the
remote frame, then `top`, `right`, `bottom`, `left`.

A side is safe when the resolved inset fits its normal frame dimension, it is
not the metadata edge, and an actual `top`/`bottom` terminal opposite metadata
does not enter the reservation strip. Validation of an automatic page terminal
checks only that this candidate set is non-empty. It must not infer the chosen
automatic side from layout `Box` geometry; final selection belongs to shared
scene construction after icon and label geometry is available. If the normal
preferred side is unsafe, scene construction remaps it to the nearest safe
candidate. No safe candidate is a source-positioned validation error at the
connection.

The endpoint- and frame-terminal-adjacent segments are perpendicular to their
own selected sides, so an endpoint may leave on `right` while its local stub
terminates at the page's bottom inset line. Frame-side and frame-anchor
attributes are valid only when the resolved endpoints belong to different
frames. Using any of them on a same-frame connection is a source-positioned
validation error.

Frame metadata reservation is a final safety constraint on that choice. For an
automatic page terminal, the metadata edge and any other unsafe side are
removed before the renderer's nearest-side choice. A terminal on a safe left
or right edge is clamped along that edge so it lies outside the top/bottom
reservation strip; any resulting coordinate difference is bridged
orthogonally. An explicit frame side or anchor that selects the reserved edge,
or an exact left/right anchor whose point lies inside the strip, is a
source-positioned validation error instead of being moved. Page-link paths and
labels remain outside the full strip.

When an explicit frame side is vertically opposite the metadata edge, its
actual terminal must remain outside the reservation strip. For bottom metadata
with explicit side `top`, the actual top terminal may not enter below the
strip's top boundary. For top metadata with explicit side `bottom`, the actual
bottom terminal may not enter above the strip's bottom boundary. A violation is
a source-positioned validation error at the connection. For an automatic page
terminal, the same conflict makes that candidate unsafe instead of immediately
rejecting the connection. A safe explicit `left` or `right` terminal remains
valid even if a hypothetical top/bottom inset line would enter the strip.

The parallel coordinate is resolved against the selected outer logical frame
edge before applying the normal inset. An explicit frame anchor keeps its exact
10/30/50/70/90-percent coordinate along the outer frame extent. An automatic
terminal's unconstrained parallel coordinate comes from the endpoint binding.
If it enters a 24-layout-px corner gutter, the parallel coordinate is clamped
and a two-bend orthogonal dogleg bridges the difference; a border shorter than
96 layout pixels uses one quarter of its length as an adaptive gutter. A
left/right terminal is also subject to the metadata reservation clamp described
above. Automatic left/right coincidence avoidance normally intersects that
corner-gutter range with an 8-layout-pixel clearance from the reservation. If a
very small non-reserved range cannot satisfy both preferences, it falls back to
the entire non-reserved interval, may touch its boundary, and never moves a
point outside the frame or inside the metadata strip.

The drawable terminal then lies on a page-terminal inset line parallel to that
outer edge. Let `i` be the resolved metadata `row-gap` when the frame has
metadata, or 4 layout pixels when it does not. The same `i` applies to every
terminal side regardless of metadata `position`; `i = 0` retains the outer
edge. An explicit `top`/`bottom` frame side requires `i < frame.height`; an
explicit `left`/`right` side requires `i < frame.width`. Failure is a
source-positioned validation error at that connection. For an automatic page
terminal, those inequalities classify candidates instead; only an empty safe
candidate set is an error. The resolved `i` is used exactly and is not reduced
to fit. With the resolved parallel coordinate represented by `u` for a
horizontal side or `v` for a vertical side, the terminal is:

```text
top:    (u, frame.y + i)
right:  (frame.x + frame.width - i, v)
bottom: (u, frame.y + frame.height - i)
left:   (frame.x + i, v)
```

The inset step changes only the normal coordinate. An explicit frame anchor
therefore retains its tangent slot and uses its local orthogonal stub for
visible separation. If an unconstrained final inset terminal would coincide
with the endpoint binding, its parallel coordinate moves by up to 24 layout
pixels within the available range so the stub remains visible. Manual bends do
not alter either local stub's geometry; bends remain logical routing metadata
for graph adapters.

There is one strict zero-inset case. When metadata is enabled with resolved
`row-gap="0"`, an endpoint resolves to its owning frame itself, and its explicit
frame anchor coincides with the resolved endpoint point, the connection is a
source-positioned validation error. An explicit endpoint anchor supplies that
point directly. An explicit endpoint side uses its center (`top` is `top-3`,
and likewise for the other sides); with neither endpoint attribute, the
automatically resolved endpoint side also uses its center. Fixed parallel
coordinates, perpendicular segments at both ends, and a visible local stub
cannot all be satisfied at that coincident point. The author must select a
different endpoint or frame anchor, or use a positive metadata `row-gap`.
