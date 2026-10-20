// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package sreagent_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/hashicorp/terraform-provider-azurerm/internal/acceptance"
	"github.com/hashicorp/terraform-provider-azurerm/internal/provider/framework"
)

func TestAccSreAgentConnector_list_basic(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_sre_agent_connector", "test")
	r := SreAgentConnectorResource{}
	listResourceAddress := "azurerm_sre_agent_connector.list"
	name := fmt.Sprintf("acctest-conn-0-%d", data.RandomIntOfLength(8))
	agentName := fmt.Sprintf("acctest-sre-%d", data.RandomIntOfLength(8))
	resourceGroupName := fmt.Sprintf("acctest-rg-sre-%d", data.RandomInteger)

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		ProtoV5ProviderFactories: framework.ProtoV5ProviderFactoriesInit(context.Background(), "azurerm"),
		Steps: []resource.TestStep{
			{
				Config: r.basicList(data),
			},
			{
				Query:  true,
				Config: r.basicQuery(data),
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLength(listResourceAddress, 2),
					querycheck.ExpectIdentity(listResourceAddress, map[string]knownvalue.Check{
						"name":                knownvalue.StringExact(name),
						"agent_name":          knownvalue.StringExact(agentName),
						"resource_group_name": knownvalue.StringExact(resourceGroupName),
						"subscription_id":     knownvalue.StringExact(data.Subscriptions.Primary),
					}),
				},
			},
		},
	})
}

func (r SreAgentConnectorResource) basicList(data acceptance.TestData) string {
	return fmt.Sprintf(`
%[1]s

resource "azurerm_sre_agent_connector" "test0" {
  name                 = "acctest-conn-0-%[2]d"
  sre_agent_id         = azurerm_sre_agent.test.id
  data_connector_type  = "MonitorClient"
  data_source          = "n/a"
  data_source_identity = "system"
}

resource "azurerm_sre_agent_connector" "test1" {
  name                = "acctest-conn-1-%[2]d"
  sre_agent_id        = azurerm_sre_agent.test.id
  data_connector_type = "Mcp"
  data_source         = "placeholder"

  extended_properties_json = jsonencode({
    type     = "http"
    endpoint = "https://learn.microsoft.com/api/mcp"
  })
}
`, SreAgentResource{}.basic(data), data.RandomIntOfLength(8))
}
func (r SreAgentConnectorResource) basicQuery(data acceptance.TestData) string {
	return fmt.Sprintf(`
list "azurerm_sre_agent_connector" "list" {
  provider = azurerm

  config {
    sre_agent_id = "/subscriptions/%s/resourceGroups/acctest-rg-sre-%d/providers/Microsoft.App/agents/acctest-sre-%d"
  }
}
`, data.Subscriptions.Primary, data.RandomInteger, data.RandomIntOfLength(8))
}
