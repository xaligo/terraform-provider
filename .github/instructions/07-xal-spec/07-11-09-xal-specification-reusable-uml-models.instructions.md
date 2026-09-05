---
applyTo: ".github/instructions/manual/**"
---

# 07.11.09 XAL specification: Reusable UML models

### Reusable UML models

Reusable definitions use `<uml-model id="...">` directly below document
`<data>`. A diagram-kind child selects one with `data="model-id"`:

```xml
<data>
  <uml-model id="order-domain">
    <class id="customer" title="Customer" />
    <class id="order" title="Order" />
    <association src="customer" dst="order" title="places" />
  </uml-model>
</data>
<frames>
  <frame id="domain">
    <uml id="model"><class-diagram data="order-domain" direction="right" /></uml>
  </frame>
</frames>
```

`<uml-model>` requires a document-unique ID. A missing model, duplicate model
ID, or a diagram that combines `data` with inline children is an error. The
model itself does not declare a UML family; after expansion, all of its element,
compartment, ownership, relation, order, and endpoint rules are validated
against the selecting diagram kind. One reusable model is therefore reusable
across selectors only when every child belongs to each selector's closed
vocabulary.
