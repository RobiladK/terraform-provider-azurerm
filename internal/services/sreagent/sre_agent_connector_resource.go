// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package sreagent

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"time"

	"github.com/hashicorp/go-azure-helpers/lang/pointer"
	"github.com/hashicorp/go-azure-helpers/lang/response"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonschema"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/resourceids"
	"github.com/hashicorp/go-azure-sdk/resource-manager/containerapps/2026-07-01/agentsconnectors"
	"github.com/hashicorp/terraform-provider-azurerm/internal/sdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/validation"
)

//go:generate go run ../../tools/generator-tests resourceidentity -resource-name sre_agent_connector -properties "name" -compare-values "subscription_id:sre_agent_id,resource_group_name:sre_agent_id,agent_name:sre_agent_id"

type SreAgentConnectorResource struct{}

var (
	_ sdk.ResourceWithIdentity = SreAgentConnectorResource{}
	_ sdk.ResourceWithUpdate   = SreAgentConnectorResource{}
)

type SreAgentConnectorModel struct {
	Name                  string `tfschema:"name"`
	SreAgentId            string `tfschema:"sre_agent_id"`
	DataConnectorType     string `tfschema:"data_connector_type"`
	DataSource            string `tfschema:"data_source"`
	DataSourceIdentity    string `tfschema:"data_source_identity"`
	ExtendedPropertiesRaw string `tfschema:"extended_properties_json"`
}

func (SreAgentConnectorResource) ModelObject() interface{} {
	return &SreAgentConnectorModel{}
}

func (SreAgentConnectorResource) ResourceType() string {
	return "azurerm_sre_agent_connector"
}

func (SreAgentConnectorResource) IDValidationFunc() pluginsdk.SchemaValidateFunc {
	return agentsconnectors.ValidateConnectorID
}

func (SreAgentConnectorResource) Identity() resourceids.ResourceId {
	return &agentsconnectors.ConnectorId{}
}

func (SreAgentConnectorResource) Arguments() map[string]*pluginsdk.Schema {
	return map[string]*pluginsdk.Schema{
		"name": {
			Type: pluginsdk.TypeString, Required: true, ForceNew: true,
			ValidateFunc: validation.StringMatch(regexp.MustCompile(`^[A-Za-z]([-A-Za-z0-9]{0,30}[A-Za-z0-9])$`),
				"must be 2 to 32 letters, digits or hyphens, start with a letter and end with a letter or digit"),
		},

		"sre_agent_id": commonschema.ResourceIDReferenceRequiredForceNew(&agentsconnectors.AgentId{}),

		"data_connector_type": {
			Type: pluginsdk.TypeString, Required: true, ForceNew: true,
			ValidateFunc: validation.StringIsNotEmpty,
		},

		// The service never returns `dataSource`, so it is kept from configuration and state only.
		"data_source": {
			Type: pluginsdk.TypeString, Optional: true, Sensitive: true,
			ValidateFunc: validation.StringIsNotEmpty,
		},

		"data_source_identity": {
			Type: pluginsdk.TypeString, Optional: true,
			ValidateFunc: validation.StringIsNotEmpty,
		},

		"extended_properties_json": {
			Type: pluginsdk.TypeString, Optional: true,
			ValidateFunc:     validateSreAgentConnectorExtendedProperties,
			DiffSuppressFunc: suppressSreAgentConnectorJSONDifference,
		},
	}
}

func (SreAgentConnectorResource) Attributes() map[string]*pluginsdk.Schema {
	return map[string]*pluginsdk.Schema{}
}

func (r SreAgentConnectorResource) Create() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Timeout: 30 * time.Minute,
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			client := metadata.Client.SreAgent.Connectors

			var config SreAgentConnectorModel
			if err := metadata.Decode(&config); err != nil {
				return fmt.Errorf("decoding: %+v", err)
			}

			agentId, err := agentsconnectors.ParseAgentID(config.SreAgentId)
			if err != nil {
				return err
			}

			id := agentsconnectors.NewConnectorID(agentId.SubscriptionId, agentId.ResourceGroupName, agentId.AgentName, config.Name)
			if !metadata.Client.Features.SkipImportCheckOnCreateAndAllowOverwritingExistingResources {
				existing, err := client.Get(ctx, id)
				if err != nil && !response.WasNotFound(existing.HttpResponse) {
					return fmt.Errorf("checking for presence of existing %s: %+v", id, err)
				}
				if !response.WasNotFound(existing.HttpResponse) {
					return metadata.ResourceRequiresImport(r.ResourceType(), id)
				}
			}

			payload, err := r.expand(config)
			if err != nil {
				return err
			}

			if err := client.CreateOrUpdateCallbackThenPoll(ctx, id, *payload, metadata.SetIDAndIdentityCallback(&id)); err != nil {
				return fmt.Errorf("creating %s: %+v", id, err)
			}

			metadata.SetID(id)
			return pluginsdk.SetResourceIdentityData(metadata.ResourceData, &id)
		},
	}
}

func (r SreAgentConnectorResource) Read() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Timeout: 5 * time.Minute,
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			id, err := agentsconnectors.ParseConnectorID(metadata.ResourceData.Id())
			if err != nil {
				return err
			}

			resp, err := metadata.Client.SreAgent.Connectors.Get(ctx, *id)
			if err != nil {
				if response.WasNotFound(resp.HttpResponse) {
					return metadata.MarkAsGone(id)
				}
				return fmt.Errorf("retrieving %s: %+v", id, err)
			}

			var prior SreAgentConnectorModel
			if err := metadata.Decode(&prior); err != nil {
				return fmt.Errorf("decoding: %+v", err)
			}

			return r.flatten(metadata, id, resp.Model, prior.DataSource, prior.ExtendedPropertiesRaw)
		},
	}
}

// Update replaces the connector with the configured values (the API is create-or-replace), so
// `data_source` is always sent again from configuration.
func (r SreAgentConnectorResource) Update() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Timeout: 30 * time.Minute,
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			id, err := agentsconnectors.ParseConnectorID(metadata.ResourceData.Id())
			if err != nil {
				return err
			}

			var config SreAgentConnectorModel
			if err := metadata.Decode(&config); err != nil {
				return fmt.Errorf("decoding: %+v", err)
			}

			payload, err := r.expand(config)
			if err != nil {
				return err
			}

			if err := metadata.Client.SreAgent.Connectors.CreateOrUpdateThenPoll(ctx, *id, *payload); err != nil {
				return fmt.Errorf("updating %s: %+v", id, err)
			}

			return nil
		},
	}
}

func (SreAgentConnectorResource) Delete() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Timeout: 30 * time.Minute,
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			id, err := agentsconnectors.ParseConnectorID(metadata.ResourceData.Id())
			if err != nil {
				return err
			}

			if err := metadata.Client.SreAgent.Connectors.DeleteThenPoll(ctx, *id); err != nil {
				return fmt.Errorf("deleting %s: %+v", id, err)
			}

			return nil
		},
	}
}

func (SreAgentConnectorResource) expand(config SreAgentConnectorModel) (*agentsconnectors.AgentConnector, error) {
	properties := agentsconnectors.AgentConnectorProperties{
		DataConnectorType: pointer.To(config.DataConnectorType),
	}

	if config.DataSource != "" {
		properties.DataSource = pointer.To(config.DataSource)
	}

	if config.DataSourceIdentity != "" {
		properties.Identity = pointer.To(config.DataSourceIdentity)
	}

	if config.ExtendedPropertiesRaw != "" {
		extended := map[string]interface{}{}
		if err := json.Unmarshal([]byte(config.ExtendedPropertiesRaw), &extended); err != nil {
			return nil, fmt.Errorf("parsing `extended_properties_json`: %+v", err)
		}
		properties.ExtendedProperties = &extended
	}

	return &agentsconnectors.AgentConnector{Properties: &properties}, nil
}

// flatten never reads `dataSource` from the service: it is write-only, so the value from
// configuration or state is carried through unchanged. The service also returns only part of
// `extendedProperties` (for the `Mcp` type, `type`, `endpoint` and `selectedTools` come back
// null), so a value already in state is kept and the API value is only used when state is empty,
// for example after an import.
func (SreAgentConnectorResource) flatten(metadata sdk.ResourceMetaData, id *agentsconnectors.ConnectorId, model *agentsconnectors.AgentConnector, dataSource, priorExtendedProperties string) error {
	state := SreAgentConnectorModel{
		Name:       id.ConnectorName,
		SreAgentId: agentsconnectors.NewAgentID(id.SubscriptionId, id.ResourceGroupName, id.AgentName).ID(),
		DataSource: dataSource,
	}

	if model != nil && model.Properties != nil {
		props := model.Properties
		state.DataConnectorType = pointer.From(props.DataConnectorType)
		state.DataSourceIdentity = pointer.From(props.Identity)

		state.ExtendedPropertiesRaw = priorExtendedProperties
		if priorExtendedProperties == "" && props.ExtendedProperties != nil {
			returned := map[string]interface{}{}
			for key, value := range *props.ExtendedProperties {
				if value != nil {
					returned[key] = value
				}
			}
			if len(returned) > 0 {
				encoded, err := json.Marshal(returned)
				if err != nil {
					return fmt.Errorf("encoding `extended_properties_json`: %+v", err)
				}
				state.ExtendedPropertiesRaw = string(encoded)
			}
		}
	}

	if err := pluginsdk.SetResourceIdentityData(metadata.ResourceData, id); err != nil {
		return err
	}

	return metadata.Encode(&state)
}

func validateSreAgentConnectorExtendedProperties(i interface{}, k string) ([]string, []error) {
	if warnings, errs := validation.StringIsJSON(i, k); len(errs) > 0 {
		return warnings, errs
	}

	var object map[string]interface{}
	if err := json.Unmarshal([]byte(i.(string)), &object); err != nil {
		return nil, []error{fmt.Errorf("%q must be a JSON object: %+v", k, err)}
	}

	return nil, nil
}

func suppressSreAgentConnectorJSONDifference(_, oldValue, newValue string, _ *pluginsdk.ResourceData) bool {
	var oldDecoded, newDecoded interface{}
	if err := json.Unmarshal([]byte(oldValue), &oldDecoded); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(newValue), &newDecoded); err != nil {
		return false
	}

	return reflect.DeepEqual(oldDecoded, newDecoded)
}
