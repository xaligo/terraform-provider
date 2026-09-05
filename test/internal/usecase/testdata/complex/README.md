# Complex Terraform conversion fixture

This fixture exercises 28 top-level infrastructure blocks across four direct
Terraform source files: 26 resources, one data source, and one local module.
It covers multiple VPC availability zones and subnets, EC2, ALB, NAT, RDS,
Lambda, SQS, S3, IAM, CloudWatch, explicit references, `count`, `for_each`, and
a dynamic block.

`modules/observability/main.tf` exists so the local module declaration is
realistic. It is intentionally not traversed because the current input
contract reads direct `.tf` files only. The module remains visible as a
generic node in `expected.xal`.

`90_xaligo.tf` is the provider-side diagram definition. It exercises the
complete `container -> row -> col` path and an item `layout` modifier;
`expected.xal` must preserve those declarations.

The expected 32 warnings are part of the regression contract. They prove that
unexpanded resources, dynamic blocks, generic fallbacks, and dependencies that
do not prove traffic are reported instead of silently guessed or discarded.
