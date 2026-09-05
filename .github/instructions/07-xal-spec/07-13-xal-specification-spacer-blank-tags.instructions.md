---
applyTo: ".github/instructions/manual/**"
---

# 07.13 XAL specification: `<spacer>` / `<blank>` Tags

## `<spacer>` / `<blank>` Tags

Dedicated empty layout tags, usable as alternatives to `<item />`.
They occupy layout slots but render no icon, label, border, or text.

```xml
<public-subnet id="public-subnet" title="Public Subnet">
  <item id="1178" />
  <spacer />          <!-- empty slot: no icon -->
  <blank />           <!-- empty slot: no icon -->
  <item id="1189" />
</public-subnet>
```

No attributes (`id` is ignored if specified).
