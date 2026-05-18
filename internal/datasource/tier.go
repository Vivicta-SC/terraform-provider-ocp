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

var _ datasource.DataSourceWithConfigure = &tierDataSource{}

func NewTierDataSource() datasource.DataSource { return &tierDataSource{} }

type tierDataSource struct{ client *client.OCPClient }

func (d *tierDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tier"
}
func (d *tierDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*client.OCPClient)
}
func (d *tierDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Represents a platform's service level. Assign to volume or virtual machine" +
			" based on desired performance and availability requirements.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Optional: true, Computed: true},
			"name": schema.StringAttribute{
				Description: "Available tier names: `Platinum`, `Gold`, `Silver` & `Bronze`",
				Optional:    true,
				Computed:    true,
			},
			"solution": schema.StringAttribute{
				Optional:   true,
				Computed:   true,
				Validators: []validator.String{stringvalidator.OneOf("OCP", "TDL", "CAAS")},
			},
		},
	}
}

type tierDataSourceModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Solution types.String `tfsdk:"solution"`
}

const tierQuery = `
query get($id: GlobalID, $filters: TierFilter, $required: Boolean! = true) {
  data: tier(id: $id, filters: $filters, required: $required) { id name solutionType }
}
`

func (d *tierDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data tierDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Solution.IsNull() {
		data.Solution = types.StringValue("OCP")
	}
	filters := map[string]interface{}{"solutionType": map[string]string{"exact": data.Solution.ValueString()}}
	if utils.IsKnown(data.ID) {
		filters["id"] = map[string]string{"exact": data.ID.ValueString()}
	}
	if utils.IsKnown(data.Name) {
		filters["name"] = map[string]string{"exact": data.Name.ValueString()}
	}

	var result struct {
		Data struct {
			client.NodeGQL

			Name     string `json:"name"`
			Solution string `json:"solutionType"`
		}
	}
	if err := d.client.Do(
		ctx,
		client.GQLRequest{Query: tierQuery, Variables: map[string]interface{}{"filters": filters}, Operation: "get"},
		&result,
		&client.DoOpts{Diags: &resp.Diagnostics},
	); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	data.ID = types.StringValue(result.Data.ID)
	data.Name = types.StringValue(result.Data.Name)
	data.Solution = types.StringValue(result.Data.Solution)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
