---
applyTo: ".github/instructions/manual/**"
---

# 07.01 XAL specification: Overview

## Overview

`.xal` is a Vue-style layout DSL with XML syntax. Canonical V1 documents use a
`<xaligo>` envelope containing document-wide data and one `<frames>` page
collection. Historical `<frame>` and `<frames>` roots remain readable but emit
a migration warning.
The parser uses `encoding/xml` and handles attributes, nested tags, and text content.
