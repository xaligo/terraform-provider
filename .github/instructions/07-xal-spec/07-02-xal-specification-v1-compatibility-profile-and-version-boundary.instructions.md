---
applyTo: ".github/instructions/manual/**"
---

# 07.02 XAL specification: V1 Compatibility Profile and Version Boundary

## V1 Compatibility Profile and Version Boundary

Canonical source uses `<xaligo>` for every language version. V1 explicitly
sets `version="1"`; V2 explicitly sets `version="2"`. An unversioned
`<xaligo>` defaults to V1 with a warning. Any other document-root version is
invalid. Legacy `<frame>` and `<frames>` roots accept the historical V1 version
rules but always emit a warning recommending the canonical envelope.
This document-root `version` selects the DSL and is not visible page metadata.
By contrast, a non-empty `version` on an identified `<frame>` that is a direct
child of the document-root `<frames>` is that page's visible content revision;
it does not select a language version. Structural diff ignores only the
document-root DSL version and compares child-frame content revisions normally.

V2 uses the same document envelope with an explicit language version:

```xml
<xaligo version="2">
  <frames>
    <frame id="main">...</frame>
  </frames>
</xaligo>
```

The parent boundary selects V1 or V2 from the `<xaligo>` `version` attribute
before parsing nested syntax. A V1 parser rejects `<xaligo version="2">`
instead of partially rendering it as V1. Do not use `version="2"` on
`<frame>` or `<frames>`; those roots remain legacy V1 only.

A V2 implementation must preserve the V1 authoring profile—its concise tags,
defaults, and compatibility behavior—while allowing V2 extensions. It lowers
both language versions directly to the shared typed model. It must not rewrite
XML, parse the document twice, or invoke V1 through a serialized intermediate
representation. V1 has no dependency on, and no obligation to understand, V2.

Canonical V1 source uses lowercase XML tag names, attribute names, and enum
tokens exactly as documented here. Historical case-insensitive or directional
aliases that are not listed in this specification are accepted implementation
details, not part of the frozen compatibility profile. A V2 compatibility
frontend canonicalizes the documented V1 values once at its input boundary.
