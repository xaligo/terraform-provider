---
applyTo: ".github/instructions/manual/**"
---

# 07.15 XAL specification: AWS Group Tags

## AWS Group Tags

Like `container`, these stack children vertically, but are rendered with **AWS architecture diagram group border styles**.
Templates are in `etc/resources/aws/templates/xal/` (`.xal`).
Icon SVGs are sourced from `etc/resources/aws/svg/Architecture-Group-Icons/`.

```xml
<aws-cloud id="production" title="Production Environment">
  <vpc id="vpc-main" title="vpc-0a1b2c3d">
    <private-subnet id="private-a" title="Private Subnet A">
      <card title="App Server" />
    </private-subnet>
  </vpc>
</aws-cloud>
```

| Tag | Display Name | Border Color | Style | Icon |
|---|---|---|---|---|
| `<aws-cloud>` | AWS Cloud | `#000000` | solid | AWS-Cloud-logo_32.svg |
| `<aws-cloud-alt>` | AWS Cloud | `#000000` | solid | AWS-Cloud_32.svg |
| `<region>` | Region | `#00A1C9` | dashed | Region_32.svg |
| `<availability-zone>` | Availability Zone | `#00A1C9` | dashed | — |
| `<security-group>` | Security group | `#CC0000` | dashed | — |
| `<auto-scaling-group>` | Auto Scaling group | `#E7601B` | dashed | Auto-Scaling-group_32.svg |
| `<vpc>` | Virtual private cloud (VPC) | `#8C4FFF` | solid | Virtual-private-cloud-VPC_32.svg |
| `<private-subnet>` | Private subnet | `#00A1C9` | solid | Private-subnet_32.svg |
| `<public-subnet>` | Public subnet | `#3F8624` | solid | Public-subnet_32.svg |
| `<server-contents>` | Server contents | `#7A7C7F` | solid | Server-contents_32.svg |
| `<corporate-data-center>` | Corporate data center | `#7A7C7F` | solid | Corporate-data-center_32.svg |
| `<ec2-instance-contents>` | EC2 instance contents | `#E7601B` | solid | EC2-instance-contents_32.svg |
| `<spot-fleet>` | Spot Fleet | `#E7601B` | solid | Spot-Fleet_32.svg |
| `<aws-account>` | AWS account | `#E7008A` | solid | AWS-Account_32.svg |
| `<aws-iot-greengrass-deployment>` | AWS IoT Greengrass Deployment | `#3F8624` | solid | AWS-IoT-Greengrass-Deployment_32.svg |
| `<aws-iot-greengrass>` | AWS IoT Greengrass | `#3F8624` | solid | — |
| `<elastic-beanstalk-container>` | Elastic Beanstalk container | `#E7601B` | solid | — |
| `<aws-step-functions-workflow>` | AWS Step Functions workflow | `#E7008A` | solid | — |
| `<generic-group>` | Generic group | `#AAB7B8` | dashed | Configurable with `icon-id` |
| `<capture>` | Capture | `#F5A623` | dashed | — |

All AWS group tags require a non-empty `id`. IDs for group tags, `<rectangle>`,
and `<port>` must be unique among frame-like components. Group tags otherwise
accept the same attributes as `container` (`title`, `class`, `gap`, etc.).

`<capture>` is a lightweight structural annotation container rather than an
AWS/architectural boundary. It participates in normal nested layout: its
children are allocated within its bordered content box, including the same
padding and optional title band used by other group tags. The border uses the
same title/text/tag-name fallback as every other group tag without implying
cloud/network semantics. Like every group tag, a `<capture>` is connectable by
`id`/`name`/`ref` from `<connection>`, including the `frameId.id` qualified
form, so a connection that starts or ends on a `<capture>` in another frame
renders as the same "to `<frame>`" / "from `<frame>`" cross-frame page-link
stubs used for any other connectable endpoint — no separate cross-boundary
arrow mechanism exists for captures.

`generic-group` additionally accepts `icon-id`, a positive signed 32-bit
decimal ID (`1..2147483647`) from `service-catalog.csv`. Zero, signs,
non-decimal syntax, and out-of-range values are invalid. It uses the same
embedded AWS, Tabler, and Yamaha icon catalog as `<item>` and renders a 32px
icon to the left of the title.
This matches the built-in group icon size. Every group header receives an
opaque mask matching its local background behind the icon and label, preventing
solid or dashed border strokes from crossing the header content.
Group header tag labels use the shared single-line text policy. The tag
background and label box use a conservative width estimate so no-wrap text
remains inside the tag in SVG and PowerPoint. Keep group tag text concise; if
changing group tag font, padding, or geometry, update the shared text-layout
policy, renderer width estimate, and regression tests together.
East Asian full-width characters, including Japanese labels, count as
double-width in group header and item label width estimates.

```xml
<generic-group id="network-topology" title="Network Topology" icon-id="104635">
  <item id="200036" />
</generic-group>
```
