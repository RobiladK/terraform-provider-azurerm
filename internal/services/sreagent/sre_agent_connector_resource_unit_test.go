// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package sreagent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-azure-helpers/lang/pointer"
	"github.com/hashicorp/go-azure-sdk/resource-manager/containerapps/2026-07-01/agentsconnectors"
	"github.com/hashicorp/go-azure-sdk/sdk/environments"
	"github.com/hashicorp/terraform-provider-azurerm/internal/clients"
	"github.com/hashicorp/terraform-provider-azurerm/internal/features"
	"github.com/hashicorp/terraform-provider-azurerm/internal/sdk"
	sreclient "github.com/hashicorp/terraform-provider-azurerm/internal/services/sreagent/client"
)

const testConnectorSecret = "do-not-log-this-value"

func sreAgentConnectorTestMetadata(t *testing.T, transport sreAgentTransport) (sdk.ResourceMetaData, agentsconnectors.ConnectorId, context.Context) {
	t.Helper()
	client, err := agentsconnectors.NewAgentsConnectorsClientWithBaseURI(environments.AzurePublic().ResourceManager)
	if err != nil {
		t.Fatal(err)
	}
	client.Client.SetTransport(transport)
	client.Client.DisableRetries = true
	client.Client.AuthorizeRequest = nil
	id := agentsconnectors.NewConnectorID("00000000-0000-0000-0000-000000000000", "example", "agent", "learn-docs")
	md := sdk.NewResourceMetaData(&clients.Client{
		Account:  &clients.ResourceManagerAccount{SubscriptionId: id.SubscriptionId},
		Features: features.UserFeatures{PersistIDOnCreateBeforePollingForCompletion: true},
		SreAgent: &sreclient.Client{Connectors: client},
	}, SreAgentConnectorResource{})
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)
	return md, id, ctx
}

func setSreAgentConnectorConfig(t *testing.T, md sdk.ResourceMetaData, id agentsconnectors.ConnectorId, extra map[string]string) {
	t.Helper()
	values := map[string]string{
		"name":                id.ConnectorName,
		"sre_agent_id":        agentsconnectors.NewAgentID(id.SubscriptionId, id.ResourceGroupName, id.AgentName).ID(),
		"data_connector_type": "Mcp",
		"data_source":         testConnectorSecret,
	}
	for key, value := range extra {
		values[key] = value
	}
	for key, value := range values {
		if err := md.ResourceData.Set(key, value); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSreAgentConnectorSchema(t *testing.T) {
	r := SreAgentConnectorResource{}
	wrapped := sdk.WrappedResource(r)
	if err := wrapped.InternalValidate(nil, true); err != nil {
		t.Fatal(err)
	}

	secret := wrapped.Schema["data_source"]
	if !secret.Sensitive || !secret.Optional || secret.Computed || secret.ForceNew {
		t.Fatalf("data_source must be an optional, sensitive, non-computed update-in-place field: %#v", secret)
	}
	for _, name := range []string{"name", "sre_agent_id", "data_connector_type"} {
		if !wrapped.Schema[name].ForceNew {
			t.Fatalf("%s must force replacement", name)
		}
	}
	for _, name := range []string{"data_source_identity", "extended_properties_json"} {
		if wrapped.Schema[name].ForceNew {
			t.Fatalf("%s must update in place", name)
		}
	}

	validateName := wrapped.Schema["name"].ValidateFunc
	for _, value := range []string{"ab", "learn-docs", "a" + strings.Repeat("b", 31)} {
		if _, errs := validateName(value, "name"); len(errs) != 0 {
			t.Fatalf("valid name %q rejected: %v", value, errs)
		}
	}
	for _, value := range []string{"a", "1abc", "abc-", "ab_c", "a" + strings.Repeat("b", 32)} {
		if _, errs := validateName(value, "name"); len(errs) == 0 {
			t.Fatalf("invalid name %q accepted", value)
		}
	}

	validateJSON := wrapped.Schema["extended_properties_json"].ValidateFunc
	if _, errs := validateJSON(`{"a":[1,2]}`, "extended_properties_json"); len(errs) != 0 {
		t.Fatalf("object rejected: %v", errs)
	}
	for _, value := range []string{`[1]`, `"text"`, `{`} {
		if _, errs := validateJSON(value, "extended_properties_json"); len(errs) == 0 {
			t.Fatalf("%q accepted", value)
		}
	}
	suppress := wrapped.Schema["extended_properties_json"].DiffSuppressFunc
	if !suppress("", `{"a":1,"b":[1,2]}`, `{ "b":[1,2], "a":1 }`, nil) || suppress("", `{"a":1}`, `{"a":2}`, nil) {
		t.Fatal("extended properties must compare semantically")
	}
}

func TestSreAgentConnectorExpand(t *testing.T) {
	r := SreAgentConnectorResource{}
	payload, err := r.expand(SreAgentConnectorModel{
		DataConnectorType:     "Mcp",
		DataSource:            testConnectorSecret,
		DataSourceIdentity:    "system",
		ExtendedPropertiesRaw: `{"type":"http","selectedTools":["a","b"]}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, encoded, `{"properties":{"dataConnectorType":"Mcp","dataSource":"`+testConnectorSecret+`","identity":"system","extendedProperties":{"type":"http","selectedTools":["a","b"]}}}`)

	payload, err = r.expand(SreAgentConnectorModel{DataConnectorType: "MonitorClient"})
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ = json.Marshal(payload)
	assertJSONEqual(t, encoded, `{"properties":{"dataConnectorType":"MonitorClient"}}`)
}

func TestSreAgentConnectorFlattenNeverReadsDataSource(t *testing.T) {
	r := SreAgentConnectorResource{}
	id := agentsconnectors.NewConnectorID("00000000-0000-0000-0000-000000000000", "example", "agent", "learn-docs")
	md := sdk.NewResourceMetaData(nil, r)
	md.SetID(id)
	model := agentsconnectors.AgentConnector{Properties: &agentsconnectors.AgentConnectorProperties{
		DataConnectorType:  pointer.To("Mcp"),
		DataSource:         pointer.To("value-returned-by-service"),
		Identity:           pointer.To("system"),
		ExtendedProperties: &map[string]interface{}{"type": "http"},
	}}

	for _, prior := range []string{"value-from-state", ""} {
		if err := r.flatten(md, &id, &model, prior, ""); err != nil {
			t.Fatal(err)
		}
		var state SreAgentConnectorModel
		if err := md.Decode(&state); err != nil {
			t.Fatal(err)
		}
		if state.DataSource != prior {
			t.Fatalf("data_source must come from prior state only: got %q, want %q", state.DataSource, prior)
		}
		if state.Name != "learn-docs" || state.DataConnectorType != "Mcp" || state.DataSourceIdentity != "system" || state.ExtendedPropertiesRaw != `{"type":"http"}` {
			t.Fatalf("unexpected state: %#v", state)
		}
	}

	model.Properties = &agentsconnectors.AgentConnectorProperties{DataConnectorType: pointer.To("Mcp")}
	if err := r.flatten(md, &id, &model, "", ""); err != nil {
		t.Fatal(err)
	}
	var state SreAgentConnectorModel
	if err := md.Decode(&state); err != nil {
		t.Fatal(err)
	}
	if state.ExtendedPropertiesRaw != "" || state.DataSourceIdentity != "" {
		t.Fatalf("stale optional state: %#v", state)
	}
}

func TestSreAgentConnectorFlattenExtendedProperties(t *testing.T) {
	r := SreAgentConnectorResource{}
	id := agentsconnectors.NewConnectorID("00000000-0000-0000-0000-000000000000", "example", "agent", "learn-docs")
	md := sdk.NewResourceMetaData(nil, r)
	md.SetID(id)
	// This is what the live service returns for an `Mcp` connector: most keys come back null.
	model := agentsconnectors.AgentConnector{Properties: &agentsconnectors.AgentConnectorProperties{
		DataConnectorType: pointer.To("Mcp"),
		ExtendedProperties: &map[string]interface{}{
			"type": nil, "endpoint": nil, "selectedTools": nil,
			"toolsVisibleToMetaAgent": []interface{}{"a"},
		},
	}}

	for name, tc := range map[string]struct{ prior, want string }{
		"state is kept":                     {`{"type":"http","selectedTools":["a"]}`, `{"type":"http","selectedTools":["a"]}`},
		"nulls dropped when state is empty": {"", `{"toolsVisibleToMetaAgent":["a"]}`},
	} {
		t.Run(name, func(t *testing.T) {
			if err := r.flatten(md, &id, &model, "", tc.prior); err != nil {
				t.Fatal(err)
			}
			var state SreAgentConnectorModel
			if err := md.Decode(&state); err != nil {
				t.Fatal(err)
			}
			if state.ExtendedPropertiesRaw != tc.want {
				t.Fatalf("got %q, want %q", state.ExtendedPropertiesRaw, tc.want)
			}
		})
	}

	model.Properties.ExtendedProperties = &map[string]interface{}{"type": nil}
	if err := r.flatten(md, &id, &model, "", ""); err != nil {
		t.Fatal(err)
	}
	var state SreAgentConnectorModel
	if err := md.Decode(&state); err != nil {
		t.Fatal(err)
	}
	if state.ExtendedPropertiesRaw != "" {
		t.Fatalf("all-null extended properties must flatten to empty, got %q", state.ExtendedPropertiesRaw)
	}
}

func TestSreAgentConnectorCreateRequiresImport(t *testing.T) {
	md, id, ctx := sreAgentConnectorTestMetadata(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("existing connector must not be overwritten: %s", req.Method)
		}
		return sreAgentHTTPResponse(req, http.StatusOK, `{"properties":{"dataConnectorType":"Mcp"}}`), nil
	})
	setSreAgentConnectorConfig(t, md, id, nil)
	if err := (SreAgentConnectorResource{}).Create().Func(ctx, md); err == nil || !strings.Contains(err.Error(), "import") {
		t.Fatalf("expected import error, got %v", err)
	}
}

func TestSreAgentConnectorCreateAndUpdateSendSecret(t *testing.T) {
	for _, update := range []bool{false, true} {
		t.Run(fmt.Sprintf("update=%t", update), func(t *testing.T) {
			var sent []byte
			md, id, ctx := sreAgentConnectorTestMetadata(t, func(req *http.Request) (*http.Response, error) {
				if strings.Contains(req.URL.Path, "/operations/") {
					return sreAgentHTTPResponse(req, http.StatusOK, `{"status":"Succeeded"}`), nil
				}
				switch req.Method {
				case http.MethodGet:
					return sreAgentHTTPResponse(req, http.StatusNotFound, `{"error":{"code":"ResourceNotFound","message":"absent"}}`), nil
				case http.MethodPut:
					body, _ := io.ReadAll(req.Body)
					sent = body
					result := sreAgentHTTPResponse(req, http.StatusCreated, `{"properties":{"provisioningState":"InProgress"}}`)
					result.Header.Set("Azure-AsyncOperation", "https://management.azure.com/operations/example?api-version=2026-07-01")
					return result, nil
				default:
					return nil, fmt.Errorf("unexpected %s request", req.Method)
				}
			})
			setSreAgentConnectorConfig(t, md, id, map[string]string{"extended_properties_json": `{"type":"http"}`})

			var err error
			if update {
				md.SetID(id)
				err = (SreAgentConnectorResource{}).Update().Func(ctx, md)
			} else {
				err = (SreAgentConnectorResource{}).Create().Func(ctx, md)
			}
			if err != nil {
				t.Fatal(err)
			}
			assertJSONEqual(t, sent, `{"properties":{"dataConnectorType":"Mcp","dataSource":"`+testConnectorSecret+`","extendedProperties":{"type":"http"}}}`)
			if md.ResourceData.Id() != id.ID() {
				t.Fatalf("unexpected ID %q", md.ResourceData.Id())
			}
		})
	}
}

func TestSreAgentConnectorReadKeepsDataSourceFromState(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusNotFound} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			md, id, ctx := sreAgentConnectorTestMetadata(t, func(req *http.Request) (*http.Response, error) {
				if status == http.StatusNotFound {
					return sreAgentHTTPResponse(req, status, `{"error":{"code":"ResourceNotFound","message":"absent"}}`), nil
				}
				return sreAgentHTTPResponse(req, status, `{"properties":{"dataConnectorType":"Mcp","dataSource":null,"identity":"system"}}`), nil
			})
			md.SetID(id)
			setSreAgentConnectorConfig(t, md, id, nil)
			if err := (SreAgentConnectorResource{}).Read().Func(ctx, md); err != nil {
				t.Fatal(err)
			}
			if status == http.StatusNotFound {
				if md.ResourceData.Id() != "" {
					t.Fatal("a missing connector must be removed from state")
				}
				return
			}
			var state SreAgentConnectorModel
			if err := md.Decode(&state); err != nil {
				t.Fatal(err)
			}
			if state.DataSource != testConnectorSecret || state.DataSourceIdentity != "system" {
				t.Fatalf("unexpected state: %#v", state)
			}
		})
	}
}

func TestSreAgentConnectorDeleteAbsent(t *testing.T) {
	md, id, ctx := sreAgentConnectorTestMetadata(t, func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodGet {
			return sreAgentHTTPResponse(req, http.StatusNotFound, `{"error":{"code":"ResourceNotFound","message":"absent"}}`), nil
		}
		if req.Method != http.MethodDelete {
			return nil, fmt.Errorf("unexpected %s request", req.Method)
		}
		return sreAgentHTTPResponse(req, http.StatusNoContent, ""), nil
	})
	md.SetID(id)
	if err := (SreAgentConnectorResource{}).Delete().Func(ctx, md); err != nil {
		t.Fatal(err)
	}
}
