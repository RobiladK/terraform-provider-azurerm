---
subcategory: "SRE Agent"
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_sre_agent_connector"
description: |-
  Lists SRE Agent Connector resources.
---

# List resource: azurerm_sre_agent_connector

Lists SRE Agent Connector resources (`Microsoft.App/agents/connectors`) for an SRE Agent.

List queries require Terraform 1.14 or later.

## Example Usage

### List all connectors of an SRE Agent

```hcl
list "azurerm_sre_agent_connector" "example" {
  provider = azurerm
  config {
    sre_agent_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-sre/providers/Microsoft.App/agents/example-sre-agent"
  }
}
```

## Argument Reference

This list resource supports the following arguments:

* `sre_agent_id` - (Required) The ID of the SRE Agent to list connectors for.

~> **Note:** The service never returns a connector's data source, so `data_source` is empty in listed results. Set it in the generated configuration before applying it.
