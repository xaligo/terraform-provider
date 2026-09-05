---
applyTo: ".github/instructions/manual/**"
---

# 07.15.02 XAL specification: Child Size Ratio Attributes

### Child Size Ratio Attributes

| Attribute | Direction | Description |
|---|---|---|
| `row` | Vertical (`layoutStack`) | **Height ratio** among children without explicit `height`. Default `1.0` |
| `col` | Horizontal (`layout="horizontal"`) | **Width ratio** among children without explicit `width`. Default `1.0` |

```xml
<!-- Horizontal: left 2 : right 1 width ratio -->
<vpc id="vpc-main" title="VPC" layout="horizontal">
  <public-subnet id="public-subnet" title="Public" col="2" />
  <private-subnet id="private-subnet" title="Private" col="1" />
</vpc>

<!-- Vertical: top 1 : bottom 2 height ratio -->
<region id="region-main" title="Region">
  <vpc id="vpc-a" title="VPC A" row="1" />
  <vpc id="vpc-b" title="VPC B" row="2" />
</region>
```
