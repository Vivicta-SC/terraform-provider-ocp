provider "ocp" {
  endpoint   = "https://ocp.service.tietoevry.com/v2/graphql"
  verify_ssl = true
}

data "ocp_customer" "ttt" { prefix = "ttt" }
data "ocp_tier" "bronze" { name = "Bronze" }
data "ocp_tier" "silver" { name = "Silver" }
data "ocp_domain" "ttt" { name = "ttt.local" }
data "ocp_network" "prod" {
  name   = "TTT_PRODUCTION"
  region = "FINLAND"
}
data "ocp_network" "staas" {
  name   = "TTT_STAAS"
  region = "FINLAND"
}
data "ocp_data_protection_policy" "no_backup" {
  customer_id            = data.ocp_customer.ttt.id
  primary_snapshot_count = 1
  secondary_backup_count = 0
  archive_count          = 0
}
data "ocp_tag" "web" {
  customer_id = data.ocp_customer.ttt.id
  name        = "Application"
  content     = "web server"
}
data "ocp_template" "rhel10" {
  name = "ttt-rhel10-template-20251002"
}
data "ocp_vserver" "main" {
  name        = "vs_c05ok_TTT_STaaS01"
  customer_id = data.ocp_customer.ttt.id
  region      = "FINLAND"
}

resource "ocp_separation_pod" "main" {
  customer_id                = data.ocp_customer.ttt.id
  name                       = "terraform"
  os_distributions           = ["ROCKY", "REDHAT"]
  domain_ids                 = [data.ocp_domain.main.id]
  network_ids                = [data.ocp_network.prod.id, data.ocp_network.staas.id]
  tier_ids                   = [data.ocp_tier.bronze.id, data.ocp_tier.silver.id]
  data_protection_policy_ids = [data.ocp_data_protection_policy.no_backup.id]
}
resource "ocp_project" "main" {
  customer_id       = data.ocp_customer.ttt.id
  separation_pod_id = resource.ocp_separation_pod.main.id
  name              = "terraform"
  note              = "Created via terraform"
}

resource "ocp_vm" "one" {
  customer_id = data.ocp_customer.ttt.id
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
  tag_ids                   = [data.ocp_tag.web.id]

  nics = [
    {
      network_id             = data.ocp_network.prod.id
      use_as_default_gateway = true
      auto_assign_ip         = true
    },
    {
      network_id     = data.ocp_network.staas.id
      auto_assign_ip = true
    }
  ]
  os_disk_size_gb = 150
  disks = [
    {
      size_gb = 35
    },
  ]
}

resource "ocp_staas_volume" "main" {
  project_id                = resource.ocp_project.main.id
  data_protection_policy_id = data.ocp_data_protection_policy.no_backup.id
  tier_id                   = data.ocp_tier.bronze.id
  vserver_id                = data.ocp_vserver.main.id
  note                      = "terraform-main"
  protocol                  = "NFS"
  nfs_exports = [
    {
      subnet_id = data.ocp_network.staas.primary_subnet_id
    },
    {
      ip_id = resource.ocp_vm.one.nics[1].ipv4[0].id
    }
  ]
  size_gb = 100
}
