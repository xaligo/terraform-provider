---
applyTo: ".github/instructions/manual/**"
---

# 07.05.03 XAL specification: `<col>`

### `<col>`

A vertical stack container inside `<row>`. Use `span` to set the number of columns occupied.

| Attribute | Type | Default | Description |
|---|---|---|---|
| `span` | float | `12 / num_columns` | Columns to occupy (out of 12) |
| `class` | string | — | Spacing class |
| `overflow` | string | `error` | Child containment policy: `error` or `visible` |
