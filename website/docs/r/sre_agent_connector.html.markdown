---
subcategory: "SRE Agent"
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_sre_agent_connector"
description: |-
  Manages a connector for an SRE Agent.
---

# azurerm_sre_agent_connector

Manages a connector for an SRE Agent (`Microsoft.App/agents/connectors`).

## Example Usage

```hcl
resource "azurerm_resource_group" "example" {
  name     = "example-sre"
  location = "East US 2"
}

resource "azurerm_sre_agent" "example" {
  name                = "example-sre-agent"
  resource_group_name = azurerm_resource_group.example.name
  location            = azurerm_resource_group.example.location

  identity {
    type = "SystemAssigned"
  }
}

resource "azurerm_sre_agent_connector" "example" {
  name                 = "azure-monitor"
  sre_agent_id         = azurerm_sre_agent.example.id
  data_connector_type  = "MonitorClient"
  data_source          = "n/a"
  data_source_identity = "system"
}
```

## Arguments Reference

The following arguments are supported:

* `name` - (Required) The name of the connector. Must contain 2 to 32 letters, digits, or hyphens, start with a letter, and end with a letter or digit. Changing this forces a new resource to be created.

* `sre_agent_id` - (Required) The ID of the SRE Agent this connector belongs to. Changing this forces a new resource to be created.

* `data_connector_type` - (Required) The type of the data connector, for example `AppInsights`, `LogAnalytics`, `Mcp` or `MonitorClient`. The service defines the accepted values, and allows only one connector of type `MonitorClient` per SRE Agent. Changing this forces a new resource to be created.

* `data_source` - (Optional) The connection string or endpoint of the data source. This value is sensitive. The service never returns it, so Terraform keeps the configured value in state and cannot detect changes made outside Terraform.

* `data_source_identity` - (Optional) The identity the agent uses to access the data source, for example `system` for the agent's system-assigned identity or a user-assigned identity resource ID.

* `extended_properties_json` - (Optional) A JSON object of additional connector properties. The content depends on `data_connector_type`. The service does not return all of the values it accepts here (for `Mcp`, it returns `type`, `endpoint` and `selectedTools` as null), so Terraform keeps the configured value in state and cannot detect changes made outside Terraform.

## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The ID of the SRE Agent Connector.

## Timeouts

The `timeouts` block allows you to specify [timeouts](https://developer.hashicorp.com/terraform/language/resources/configure#define-operation-timeouts) for certain actions:

* `create` - (Defaults to 30 minutes) Used when creating the SRE Agent Connector.
* `read` - (Defaults to 5 minutes) Used when retrieving the SRE Agent Connector.
* `update` - (Defaults to 30 minutes) Used when updating the SRE Agent Connector.
* `delete` - (Defaults to 30 minutes) Used when deleting the SRE Agent Connector.

## Import

SRE Agent Connectors can be imported using the `resource id`, for example:

```shell
terraform import azurerm_sre_agent_connector.example /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-sre/providers/Microsoft.App/agents/example-sre-agent/connectors/azure-monitor
```

Because the service does not return `data_source`, an imported connector has no `data_source` in state. The next `terraform apply` sends the configured value.

## API Providers
<!-- This section is generated, changes will be overwritten -->
This resource uses the following Azure API Providers:

* `Microsoft.App` - 2026-07-01
