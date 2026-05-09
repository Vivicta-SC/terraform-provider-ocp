terraform {
  required_providers {
    ocp = {
      source = "hashicorp.com/Vivicta-SC/ocp"
    }
  }
}
provider "ocp" { debug = true }

data "ocp_customer" "main" { prefix = "ed2" }
data "ocp_tier" "bronze" { name = "Bronze" }
data "ocp_tier" "silver" { name = "Silver" }
data "ocp_domain" "main" { name = "ed2.local" }
data "ocp_network" "prod" {
  name   = "ED2_PRODUCTION"
  region = "FINLAND"
}
data "ocp_network" "staas" {
  name   = "ED2_STAAS"
  region = "FINLAND"
}
data "ocp_data_protection_policy" "no_backup" {
  customer_id            = data.ocp_customer.main.id
  primary_snapshot_count = 1
  secondary_backup_count = 0
  archive_count          = 0
  # note        = "no backup"
}
data "ocp_tag" "main" {
  customer_id = data.ocp_customer.main.id
  name        = "Application"
  content     = "web server"
}
data "ocp_template" "rhel10" {
  name = "ed2-rhel10-template-20251002"
}
data "ocp_vserver" "main" {
  name        = "vs_c05ok_ED2_STaaS01"
  customer_id = data.ocp_customer.main.id
  region      = "FINLAND"
}

resource "ocp_separation_pod" "main" {
  customer_id                = data.ocp_customer.main.id
  name                       = "terraform"
  os_distributions           = ["ROCKY", "REDHAT"]
  domain_ids                 = [data.ocp_domain.main.id]
  network_ids                = [data.ocp_network.prod.id, data.ocp_network.staas.id]
  tier_ids                   = [data.ocp_tier.bronze.id, data.ocp_tier.silver.id]
  data_protection_policy_ids = [data.ocp_data_protection_policy.no_backup.id]
}
resource "ocp_project" "main" {
  customer_id       = data.ocp_customer.main.id
  separation_pod_id = resource.ocp_separation_pod.main.id
  name              = "terraform"
  note              = "Created via terraform"
}

resource "ocp_vm" "one" {
  customer_id = data.ocp_customer.main.id
  domain_id   = data.ocp_domain.main.id
  project_id  = resource.ocp_project.main.id
  tier_id     = data.ocp_tier.bronze.id

  template_id               = data.ocp_template.rhel10.id
  data_protection_policy_id = data.ocp_data_protection_policy.no_backup.id
  hostname                  = "terraform-1"
  cpu_count                 = 2
  memory_size_gb            = 6
  region                    = "FINLAND"
  note                      = "Created via terraform"
  tag_ids                   = [data.ocp_tag.main.id]

  nics = [
    {
      network_id                = data.ocp_network.prod.id
      use_as_default_gateway_wo = true
      auto_assign_ip_wo         = true
    },
    {
      network_id        = data.ocp_network.staas.id
      auto_assign_ip_wo = true
    }
  ]
  disks = [
    {
      size_gb = 35
    },
  ]

  config = {
    await_deletion_task = true
    timeouts = {
      create = "15m"
    }
  }
}

resource "ocp_staas_group" "main" {
  project_id                = resource.ocp_project.main.id
  data_protection_policy_id = data.ocp_data_protection_policy.no_backup.id
  tier_id                   = data.ocp_tier.bronze.id
  vserver_id                = data.ocp_vserver.main.id
  name                      = "terraform"
  note                      = "Created via terraform"
  protocol                  = "NFS"
}
resource "ocp_staas_group" "second" {
  project_id                = resource.ocp_project.main.id
  data_protection_policy_id = data.ocp_data_protection_policy.no_backup.id
  tier_id                   = data.ocp_tier.bronze.id
  vserver_id                = data.ocp_vserver.main.id
  name                      = "terraform-2"
  note                      = "Created via terraform"
  protocol                  = "NFS"
  nfs_exports = [
    {
      subnet_id = data.ocp_network.staas.primary_subnet_id
    },
    {
      ip_id = resource.ocp_vm.one.nics[1].ipv4[0].id
    }
  ]
}

resource "ocp_staas_volume" "main" {
  staas_group_id = resource.ocp_staas_group.second.id
  size_gb        = 25
}

output "out" {
  value = resource.ocp_separation_pod.main.id
}


# action "ocp_await_task" "await_one" {
#   config {
#     task_id = resource.ocp_vm.one.creation_task_id
#   }
# }

# resource "ocp_vm" "two" {
#   customer_id               = data.ocp_customer.main.id
#   domain_id                 = data.ocp_domain.main.id
#   project_id                = resource.ocp_project.main.id
#   tier_id                   = data.ocp_tier.bronze.id
#   template_id               = "VGVtcGxhdGVOb2RlOjQ0ODIx"
#   data_protection_policy_id = "RGF0YVByb3RlY3Rpb25Qb2xpY3lOb2RlOjE5"
#   hostname                  = "terraform-2"
#   cpu_count                 = 1
#   memory_size_gb            = 4
#   region                    = "FINLAND"
#   note                      = "RHEL coz ROCKY is broken"

#   os_disk_size_gb_wo = 125

#   lifecycle {
#     action_trigger {
#       events  = [before_create]
#       actions = [action.ocp_await_task.await_one]
#     }
#   }
# }
