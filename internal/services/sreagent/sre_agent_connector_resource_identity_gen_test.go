// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package sreagent_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-provider-azurerm/internal/acceptance"
	customstatecheck "github.com/hashicorp/terraform-provider-azurerm/internal/acceptance/statecheck"
)

func TestAccSreAgentConnector_resourceIdentity(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_sre_agent_connector", "test")
	r := SreAgentConnectorResource{}

	checkedFields := map[string]struct{}{
		"name":                {},
		"agent_name":          {},
		"resource_group_name": {},
		"subscription_id":     {},
	}

	data.ResourceIdentityTest(t, []acceptance.TestStep{
		{
			Config: r.basic(data),
			ConfigStateChecks: []statecheck.StateCheck{
				customstatecheck.ExpectAllIdentityFieldsAreChecked("azurerm_sre_agent_connector.test", checkedFields),
				statecheck.ExpectIdentityValueMatchesStateAtPath("azurerm_sre_agent_connector.test", tfjsonpath.New("name"), tfjsonpath.New("name")),
				customstatecheck.ExpectStateContainsIdentityValueAtPath("azurerm_sre_agent_connector.test", tfjsonpath.New("agent_name"), tfjsonpath.New("sre_agent_id")),
				customstatecheck.ExpectStateContainsIdentityValueAtPath("azurerm_sre_agent_connector.test", tfjsonpath.New("resource_group_name"), tfjsonpath.New("sre_agent_id")),
				customstatecheck.ExpectStateContainsIdentityValueAtPath("azurerm_sre_agent_connector.test", tfjsonpath.New("subscription_id"), tfjsonpath.New("sre_agent_id")),
			},
		},
		data.ImportBlockWithResourceIdentityStep(true),
		data.ImportBlockWithIDStep(true),
	}, false)
}
