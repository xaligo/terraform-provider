---
applyTo: ".github/instructions/manual/**"
---

# 07.14 XAL specification: `<connection>` Tag

## `<connection>` Tag

Draws an **elbowed arrow** between `<item>` elements or group borders.
Must be written as a direct child of `<frame>` or inside a frame-level
`<connections>` tag, **outside** layout tags.
Use the same catalog IDs as `<item id="N">`, or assign `id`, `name`, or `ref`
to an AWS/group tag, for `src` / `dst`.

```xml
<frame width="1122" height="794" class="pa-4">
  <aws-cloud id="cloud" title="AWS Cloud">
    <public-subnet id="public" title="Public Subnet">
      <item id="1178" />
      <item id="1189" />
    </public-subnet>
  </aws-cloud>

  <!-- connections go last, as direct children of frame or inside <connections> -->
  <connections grid="8">
    <connection src="1178" dst="1189" />
    <connection src="public" dst="cloud" kind="route" />
  </connections>
</frame>
```
