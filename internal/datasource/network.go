// Copyright @ Vivicta. All Rights Reserved. 2026
// SPDX-License-Identifier: MPL-2.0

package datasource

import (
	"context"

	"github.com/Vivicta-SC/terraform-provider-ocp/internal/client"
	"github.com/Vivicta-SC/terraform-provider-ocp/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSourceWithConfigure = &networkDataSource{}

func NewNetworkDataSource() datasource.DataSource { return &networkDataSource{} }

type networkDataSource struct{ client *client.OCPClient }

func (d *networkDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network"
}
func (d *networkDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*client.OCPClient)
}
func (d *networkDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Optional: true, Computed: true},
			"name":        schema.StringAttribute{Optional: true, Computed: true},
			"customer_id": schema.StringAttribute{Optional: true, Computed: true},
			"region": schema.StringAttribute{
				Optional:   true,
				Computed:   true,
				Validators: []validator.String{stringvalidator.OneOf("SWEDEN", "NORWAY", "FINLAND")},
			},
			"primary_subnet_id": schema.StringAttribute{Computed: true},
		},
	}
}

type networkDataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Region          types.String `tfsdk:"region"`
	CustomerID      types.String `tfsdk:"customer_id"`
	PrimarySubnetID types.String `tfsdk:"primary_subnet_id"`
}

const networkQuery = `
query get($id: GlobalID, $filters: NetworkFilter, $required: Boolean! = true) {
  data: network(id: $id, filters: $filters, required: $required) {
    id
    name
    customer { id }
    type
    core { region }
    solutionType
    primarySubnet { id }
  }
}
`

func (d *networkDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data networkDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	filters := map[string]interface{}{"isEnabled": map[string]bool{"exact": true}}
	if utils.IsKnown(data.ID) {
		filters["id"] = map[string]string{"exact": data.ID.ValueString()}
	}
	if utils.IsKnown(data.Name) {
		filters["name"] = map[string]string{"exact": data.Name.ValueString()}
	}
	if utils.IsKnown(data.CustomerID) {
		filters["customer"] = map[string]interface{}{"id": map[string]string{"exact": data.CustomerID.ValueString()}}
	}
	if utils.IsKnown(data.Region) {
		filters["core"] = map[string]interface{}{"region": map[string]interface{}{"exact": data.Region.ValueString()}}
	}
	var result struct {
		Data struct {
			client.NodeGQL

			Name          string
			Customer      client.NodeGQL
			PrimarySubnet client.NodeGQL
			Core          struct{ Region string }
		}
	}

	if err := d.client.Do(
		ctx,
		client.GQLRequest{Query: networkQuery, Variables: map[string]interface{}{"filters": filters}, Operation: "get"},
		&result,
		&client.DoOpts{Diags: &resp.Diagnostics},
	); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	data.ID = types.StringValue(result.Data.ID)
	data.Name = types.StringValue(result.Data.Name)
	data.Region = types.StringValue(result.Data.Core.Region)
	data.CustomerID = types.StringValue(result.Data.Customer.ID)
	data.PrimarySubnetID = types.StringValue(result.Data.PrimarySubnet.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
