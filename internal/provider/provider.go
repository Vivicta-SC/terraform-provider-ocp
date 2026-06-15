// Copyright @ Vivicta. All Rights Reserved. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	ocpaction "github.com/Vivicta-SC/terraform-provider-ocp/internal/action"
	"github.com/Vivicta-SC/terraform-provider-ocp/internal/client"
	ocpdatasource "github.com/Vivicta-SC/terraform-provider-ocp/internal/datasource"
	ocpresource "github.com/Vivicta-SC/terraform-provider-ocp/internal/resource"
)

var _ provider.ProviderWithActions = &ocpProvider{}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ocpProvider{version: version}
	}
}

type ocpProvider struct {
	version string
}
type ocpProviderModel struct {
	Endpoint  types.String `tfsdk:"endpoint"`
	VerifySsl types.Bool   `tfsdk:"verify_ssl"`
	Debug     types.Bool   `tfsdk:"debug"`
}

func (p *ocpProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "ocp"
	resp.Version = p.version
}

func (p *ocpProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Vivicta OneCloud Platinum (OCP) Terraform Provider." +
			" For detailed information, see the white papers available on the portal website.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Optional: true,
				Description: "OCP GraphQL endpoint. Can be loaded from env `OCP_ENDPOINT`." +
					" Defaults to latest production endpoint (https://ocp.service.tietoevry.com/v2/graphql)",
			},
			"verify_ssl": schema.BoolAttribute{Optional: true, Description: "Skip TLS certificate verification. Defaults to true"},
			"debug": schema.BoolAttribute{
				Optional:    true,
				Description: "Enables additional OCP GraphQL usage data (warning, deprecation) - subject to permissions. Default to false",
			},
		},
	}
}

func (p *ocpProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg ocpProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := os.Getenv("OCP_ENDPOINT")
	if !cfg.Endpoint.IsNull() {
		endpoint = cfg.Endpoint.ValueString()
	}
	if endpoint == "" {
		endpoint = "https://ocp.service.tietoevry.com/v2/graphql"
	}

	token := os.Getenv("OCP_TOKEN")
	if token == "" {
		resp.Diagnostics.AddError("Missing OCP Token", "The OCP provider requires an authentication token.")
		return
	}

	verifySsl := true
	if !cfg.VerifySsl.IsNull() {
		verifySsl = cfg.VerifySsl.ValueBool()
	} else {
		parsed, err := strconv.ParseBool(os.Getenv("OCP_VERIFY_SSL"))
		if err != nil {
			verifySsl = parsed
		}
	}

	var debug bool
	if !cfg.Debug.IsNull() {
		debug = cfg.Debug.ValueBool()
	} else {
		debug, _ = strconv.ParseBool(os.Getenv("OCP_DEBUG"))
	}

	ocp_client := client.New(endpoint, token, verifySsl, debug)
	resp.DataSourceData = ocp_client
	resp.ResourceData = ocp_client
	resp.EphemeralResourceData = ocp_client
	resp.ActionData = ocp_client
	resp.ListResourceData = ocp_client
}

func (p *ocpProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		ocpdatasource.NewCustomerDataSource,
		ocpdatasource.NewDataProtectionPolicyDataSource,
		ocpdatasource.NewDomainDataSource,
		ocpdatasource.NewNetworkDataSource,
		ocpdatasource.NewTagDataSource,
		ocpdatasource.NewTemplateDataSource,
		ocpdatasource.NewTierDataSource,
		ocpdatasource.NewVserverDataSource,
	}
}
func (p *ocpProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		ocpresource.NewProjectResource,
		ocpresource.NewSeparationPodResource,
		ocpresource.NewSTAASVolumeResource,
		ocpresource.NewVMResource,
	}
}
func (p *ocpProvider) Actions(_ context.Context) []func() action.Action {
	return []func() action.Action{
		ocpaction.NewAwaitTaskAction,
	}
}
