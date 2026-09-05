# Simple VPC sample

This sample reduces the VPC, subnet, and resource-reference style used by
[tf-mark1](https://github.com/ryo-arima/tf-mark1) to three AWS resource blocks.
Host-side Terraform provisions those resources in MiniStack while the xaligo
provider creates the corresponding local diagram source.

- `main.tf` configures the xaligo provider with `export = "enable"` and owns
  `generated.xal` through `xaligo_diagram.architecture`. Its AWS provider sends
  EC2 API requests only to the local MiniStack endpoint with test credentials.
- `source/main.tf` is both the local Terraform module applied to MiniStack and
  the direct input directory read by the converter.
- `expected.xal` is the canonical XAL V1 golden output. The subnet remains a
  neutral group because `map_public_ip_on_launch` alone does not prove an
  internet route.

From the repository root, build the host-native provider in Docker, start
MiniStack, and run the complete sample with the host Terraform executable:

```sh
make sample
xaligo validate samples/simple-vpc/generated.xal
```

The individual lifecycle targets are `make tf-init`, `make tf-plan`,
`make tf-apply`, and `make tf-destroy`. They intentionally execute
Terraform on the host with `samples/simple-vpc` as its working directory;
Compose contains no Terraform or provider service.

The same conversion core is available without Terraform:

```sh
make convert
```

Changing the provider setting to `export = "disable"` keeps Terraform from
creating or replacing `generated.xal`.
