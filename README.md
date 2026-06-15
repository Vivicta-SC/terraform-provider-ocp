# Terraform Provider Vivicta OCP

This repository contains the Terraform provider for Vivicta OneCloud Platinum (OCP).

Learn more in [documentation](https://registry.terraform.io/providers/Vivicta-SC/ocp/latest/docs).

## Requirements
- [Go](https://go.dev/doc/install) 1.26+ (when bulding)
- [Terraform](https://developer.hashicorp.com/terraform/downloads) 1.15+
- Access to Vivicta OCP GraphQL API

## Usage (Quick Start)
### Provider Configuration
#### Arguments
| Name | Required | Default | Description |
|-|-|-|-|
| `endpoint` | no | Latest production endpoint | Can be provided via `OCP_ENDPOINT` |
| `verify_ssl` | no | true | Skip TLS certificate verification |

```hcl
terraform {
  required_providers {
    ocp = {
      source = "hashicorp.com/Vivicta-SC/ocp"
    }
  }
}
provider "ocp" {
    endpoint = "https://ocp.service.tietoevry.com/v2/graphql"
    verify_ssl = true
}
```

#### Authentication

```bash
export OCP_TOKEN="your_token"
```

## Data source overview
- `ocp_customer`
- `ocp_data_protection_policy`
- `ocp_domain`
- `ocp_network`
- `ocp_tag`
- `ocp_template`
- `ocp_tier`
- `ocp_vserver`
## Resource overview
- `ocp_project`
- `ocp_separation_pod`
- `ocp_staas_volume`
- `ocp_vm`

## Contributing
TODO: [contribution guide](./CONTRIBUTING.md)

## License
The OCP Provider Provider is under [Mozilla Public License Version 2.0](./LICENSE)