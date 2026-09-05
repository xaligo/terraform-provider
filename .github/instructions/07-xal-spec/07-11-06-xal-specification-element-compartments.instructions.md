---
applyTo: ".github/instructions/manual/**"
---

# 07.11.06 XAL specification: Element compartments

### Element compartments

An element's direct child tags are ordered text compartments. Each compartment
must have non-whitespace direct text, `title`, or `name` and must not contain
child elements. Nested UML elements and relations are not compartments. The
typed compartment vocabulary is:

| Element | Allowed typed compartments |
|---|---|
| `class` | `attribute`, `operation`, `constraint`, `note` |
| `interface` | `operation`, `constraint`, `note` |
| `enumeration` | `literal`, `operation`, `note` |
| `component` | `interface`, `provided-interface`, `required-interface`, `property`, `constraint`, `note` |
| `artifact` | `property`, `responsibility`, `note` |
| `activity`, `action` | `responsibility`, `constraint`, `note` |
| `state` | `entry`, `do`, `exit`, `region`, `note` |

Elements absent from this table do not accept compartments. The generic
`<compartment>` child is a compatibility spelling accepted wherever a typed
compartment is allowed; new source should use the typed tag because its meaning
survives future semantic processing. Compartment source order is preserved,
but compartments are not independent connection endpoints. In a class diagram,
adjacent structural (`attribute` or `literal`) compartments and adjacent
behavioral compartments may share one graphical section; a transition between
those kinds starts a new section without reordering either kind. Every
newline-separated compartment line contributes to the classifier's intrinsic
height.
