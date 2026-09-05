---
applyTo: ".github/instructions/manual/**"
---

# 07.11.01 XAL specification: Component, identity, and layout contract

### Component, identity, and layout contract

```xml
<xaligo version="1">
  <data>
    <uml-model id="domain-model">
      <class id="order" title="Order">
        <attribute>- id: UUID</attribute>
        <operation>+ confirm()</operation>
      </class>
      <interface id="repository" title="OrderRepository">
        <operation>save(order: Order)</operation>
      </interface>
      <realization src="order" dst="repository" title="persists" />
    </uml-model>
  </data>
  <frames>
    <frame id="domain" width="960" height="540">
      <uml id="model" title="Domain Model">
        <class-diagram data="domain-model" direction="right" />
      </uml>
    </frame>
  </frames>
</xaligo>
```

The following rules are normative:

- `<uml>` must be inside a frame, requires a non-empty `id`, and contains
  exactly one supported diagram-kind child. UML IDs must be unique within that
  frame. The same UML ID may be reused in a different frame.
- UML component IDs and diagram-local element IDs must not contain whitespace,
  `.` or `/`. `.` is reserved for the frame boundary and `/` for the UML
  boundary in public connection references.
- The diagram-kind child contains a non-empty set of direct element and
  relation children. Unknown diagram kinds and unknown children are errors;
  arbitrary custom tags are not generic UML elements.
- Every UML element requires a non-empty diagram-local `id`, unique within the
  UML component. A UML relation's `src` and `dst` use those local IDs, without
  a UML or frame prefix, and both endpoints must exist in the selected model.
- `direction` on the diagram-kind child accepts only `right` or `down`. When
  `<uml layout>` is omitted, `direction="right"` and sequence diagrams
  diagrams select horizontal xaligo layout; the other cases select vertical
  layout. This controls the V1 projection and is not a UML semantic ordering
  rule.
- When `<uml title>` is omitted, the selector name without `-diagram` is used.
  Element labels resolve in the order `title`, `name`, direct text, then local
  `id`. UML elements default to `font-family="helvetica"` and `font-size="14"`;
  normal element font attributes override those defaults. An element `name`
  is display text only and never becomes a frame-level connection alias; use
  the public UML reference described below.
- A class-diagram classifier with a non-empty `stereotype` renders
  `«stereotype»` as a separate first header line. `abstract="true"` and
  `static="true"` append `{abstract}` and `{static}` to the classifier-name
  header line. These lines remain one graphical header even when the
  classifier has no compartments.
- The compatibility tags `<element>` and `<relation>` are not part of the
  strict V1 UML profile. A model must use one of the element and relation tags
  allowed for its selected family.

UML elements are also normal xaligo connection endpoints. A frame-level
`<connection>` uses the following public references; the internal hex-scoped
scene ID is opaque and must not be written in source:

| Location | Public endpoint reference | Meaning |
|---|---|---|
| Same frame | `uml-id/local-id` | Element `local-id` in UML component `uml-id` |
| Another frame | `frame-id.uml-id/local-id` | The same UML element reached across a frame boundary |

For example, element `order` in `<uml id="model">` inside frame `overview` is
`model/order` to a normal connection in that frame and
`overview.model/order` to a normal connection in another frame. Omitting the
`frame-id.` prefix for a cross-frame endpoint is an unresolved-reference
error. UML-native relations continue to use `src="order"`, not either public
connection form.
