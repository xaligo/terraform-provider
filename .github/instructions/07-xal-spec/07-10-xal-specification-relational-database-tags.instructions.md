---
applyTo: ".github/instructions/manual/**"
---

# 07.10 XAL specification: Relational Database Tags

## Relational Database Tags

Reusable `<database-schema id="...">` definitions belong under `<data>`.
Frames render them through `<database data="schema-id">`. A schema contains
identified `<entity>` definitions; each entity contains typed `<column>`
definitions and optional single-column `<foreign-key>` definitions.

```xml
<database-schema id="app">
  <entity id="roles"><column name="id" type="bigint" primary-key="true" /></entity>
  <entity id="users">
    <column name="role_id" type="bigint" nullable="false" />
    <foreign-key columns="role_id" references="roles.id" />
  </entity>
</database-schema>
```

Columns support `name`, `type`, `primary-key`, `nullable`, `unique`, and
`default`. A foreign key requires one local column and one
`references="entity.column"` target and generates a relation between the
entities. Duplicate or missing schema/entity/column references and mixed
inline/data-backed database content are positioned errors. Composite keys,
indexes, checks, and import dialects remain planned V1 extensions.
