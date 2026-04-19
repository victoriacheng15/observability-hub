resource_group_name  = "observability-rg"
storage_account_name = "obshub"
container_name       = "terraform-state"
key                  = "observability-hub.tofu.terraform.tfstate"
use_azuread_auth     = true
