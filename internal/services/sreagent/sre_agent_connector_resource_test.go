// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package sreagent_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/go-azure-helpers/lang/pointer"
	"github.com/hashicorp/go-azure-sdk/resource-manager/containerapps/2026-07-01/agentsconnectors"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-provider-azurerm/internal/acceptance"
	"github.com/hashicorp/terraform-provider-azurerm/internal/acceptance/check"
	"github.com/hashicorp/terraform-provider-azurerm/internal/clients"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
)

type SreAgentConnectorResource struct{}

func TestAccSreAgentConnector_basic(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_sre_agent_connector", "test")
	r := SreAgentConnectorResource{}

	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.basic(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
				check.That(data.ResourceName).Key("data_connector_type").HasValue("MonitorClient"),
				check.That(data.ResourceName).Key("data_source_identity").HasValue("system"),
			),
		},
		// The service never returns `data_source`, so it cannot be verified after import.
		data.ImportStep("data_source"),
	})
}

func TestAccSreAgentConnector_requiresImport(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_sre_agent_connector", "test")
	r := SreAgentConnectorResource{}

	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.basic(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.RequiresImportErrorStep(r.requiresImport),
	})
}

func TestAccSreAgentConnector_complete(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_sre_agent_connector", "test")
	r := SreAgentConnectorResource{}

	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.complete(data, "placeholder", `["learn-docs_microsoft_docs_search", "learn-docs_microsoft_docs_fetch"]`),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
				check.That(data.ResourceName).Key("data_connector_type").HasValue("Mcp"),
				resource.TestCheckResourceAttrSet(data.ResourceName, "extended_properties_json"),
			),
		},
		data.ImportStep("data_source", "extended_properties_json"),
	})
}

func TestAccSreAgentConnector_update(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_sre_agent_connector", "test")
	r := SreAgentConnectorResource{}

	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.complete(data, "placeholder", `["learn-docs_microsoft_docs_search", "learn-docs_microsoft_docs_fetch"]`),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.ImportStep("data_source", "extended_properties_json"),
		{
			Config: r.complete(data, "placeholder-updated", `["learn-docs_microsoft_docs_search"]`),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.ImportStep("data_source", "extended_properties_json"),
	})
}

func (SreAgentConnectorResource) Exists(ctx context.Context, client *clients.Client, state *pluginsdk.InstanceState) (*bool, error) {
	id, err := agentsconnectors.ParseConnectorID(state.ID)
	if err != nil {
		return nil, err
	}

	resp, err := client.SreAgent.Connectors.Get(ctx, *id)
	if err != nil {
		return nil, fmt.Errorf("retrieving %s: %+v", *id, err)
	}

	return pointer.To(resp.Model != nil), nil
}

func (r SreAgentConnectorResource) basic(data acceptance.TestData) string {
	return fmt.Sprintf(`
%s

resource "azurerm_sre_agent_connector" "test" {
  name                 = "acctest-conn-%d"
  sre_agent_id         = azurerm_sre_agent.test.id
  data_connector_type  = "MonitorClient"
  data_source          = "n/a"
  data_source_identity = "system"
}
`, SreAgentResource{}.basic(data), data.RandomIntOfLength(8))
}

func (r SreAgentConnectorResource) requiresImport(data acceptance.TestData) string {
	return fmt.Sprintf(`
%s

resource "azurerm_sre_agent_connector" "import" {
  name                 = azurerm_sre_agent_connector.test.name
  sre_agent_id         = azurerm_sre_agent_connector.test.sre_agent_id
  data_connector_type  = azurerm_sre_agent_connector.test.data_connector_type
  data_source          = "n/a"
  data_source_identity = "system"
}
`, r.basic(data))
}

func (r SreAgentConnectorResource) complete(data acceptance.TestData, dataSource, selectedTools string) string {
	return fmt.Sprintf(`
%s

resource "azurerm_sre_agent_connector" "test" {
  name                = "acctest-conn-%d"
  sre_agent_id        = azurerm_sre_agent.test.id
  data_connector_type = "Mcp"
  data_source         = "%s"

  extended_properties_json = jsonencode({
    type                    = "http"
    endpoint                = "https://learn.microsoft.com/api/mcp"
    selectedTools           = %s
    toolsVisibleToMetaAgent = %s
  })
}
`, SreAgentResource{}.basic(data), data.RandomIntOfLength(8), dataSource, selectedTools, selectedTools)
}
