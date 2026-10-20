// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package sreagent

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-azure-helpers/framework/typehelpers"
	"github.com/hashicorp/go-azure-helpers/lang/pointer"
	"github.com/hashicorp/go-azure-sdk/resource-manager/containerapps/2026-07-01/agentsconnectors"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-azurerm/internal/sdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
)

type SreAgentConnectorListResource struct{}

type SreAgentConnectorListModel struct {
	SreAgentId types.String `tfsdk:"sre_agent_id"`
}

var _ sdk.FrameworkListWrappedResource = new(SreAgentConnectorListResource)

func (SreAgentConnectorListResource) ResourceFunc() *pluginsdk.Resource {
	return sdk.WrappedResource(SreAgentConnectorResource{})
}

func (SreAgentConnectorListResource) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = SreAgentConnectorResource{}.ResourceType()
}

func (SreAgentConnectorListResource) ListResourceConfigSchema(_ context.Context, _ list.ListResourceSchemaRequest, response *list.ListResourceSchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"sre_agent_id": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					typehelpers.WrappedStringValidator{Func: agentsconnectors.ValidateAgentID},
				},
			},
		},
	}
}

func (SreAgentConnectorListResource) List(ctx context.Context, request list.ListRequest, stream *list.ListResultsStream, metadata sdk.ResourceMetadata) {
	client := metadata.Client.SreAgent.Connectors

	var data SreAgentConnectorListModel
	diags := request.Config.Get(ctx, &data)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	r := SreAgentConnectorResource{}

	agentId, err := agentsconnectors.ParseAgentID(data.SreAgentId.ValueString())
	if err != nil {
		sdk.SetResponseErrorDiagnostic(stream, "parsing `sre_agent_id`", err)
		return
	}

	resp, err := client.ListByAgentComplete(ctx, *agentId)
	if err != nil {
		sdk.SetResponseErrorDiagnostic(stream, fmt.Sprintf("listing `%s`", r.ResourceType()), err)
		return
	}

	stream.Results = func(push func(list.ListResult) bool) {
		for _, item := range resp.Items {
			result := request.NewListResult(ctx)
			result.DisplayName = pointer.From(item.Name)

			id, err := agentsconnectors.ParseConnectorIDInsensitively(pointer.From(item.Id))
			if err != nil {
				sdk.SetErrorDiagnosticAndPushListResult(result, push, "parsing SRE Agent Connector ID", err)
				return
			}

			rmd := sdk.NewResourceMetaData(metadata.Client, r)
			rmd.SetID(id)
			// The service never returns the connector's data source, so list results leave it empty. Extended properties are limited to what the service returns.
			if err := r.flatten(rmd, id, &item, "", ""); err != nil {
				sdk.SetErrorDiagnosticAndPushListResult(result, push, fmt.Sprintf("encoding `%s` resource data", r.ResourceType()), err)
				return
			}

			sdk.EncodeListResult(ctx, rmd.ResourceData, &result)
			if result.Diagnostics.HasError() {
				push(result)
				return
			}

			if !push(result) {
				return
			}
		}
	}
}
