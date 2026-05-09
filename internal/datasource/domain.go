// Copyright @ Vivicta. All Rights Reserved. 2026
// SPDX-License-Identifier: MPL-2.0

package datasource

import (
	"context"

	"github.com/Vivicta-SC/terraform-provider-ocp/internal/client"
	"github.com/Vivicta-SC/terraform-provider-ocp/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSourceWithConfigure = &domainDataSource{}

func NewDomainDataSource() datasource.DataSource { return &domainDataSource{} }

type domainDataSource struct{ client *client.OCPClient }

func (d *domainDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}
func (d *domainDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*client.OCPClient)
}
func (d *domainDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Optional: true, Computed: true},
			"name":        schema.StringAttribute{Optional: true, Computed: true},
			"customer_id": schema.StringAttribute{Optional: true, Computed: true},
		},
	}
}

type domainDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	CustomerID types.String `tfsdk:"customer_id"`
}

const domainQuery = `
query get($id: GlobalID, $filters: DomainFilter, $required: Boolean! = true) {
  data: domain(id: $id, filters: $filters, required: $required) { id name customer { id } }
}
`

func (d *domainDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data domainDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	filters := map[string]interface{}{}
	if utils.IsKnown(data.ID) {
		filters["id"] = map[string]string{"exact": data.ID.ValueString()}
	}
	if utils.IsKnown(data.Name) {
		filters["name"] = map[string]string{"exact": data.Name.ValueString()}
	}
	if utils.IsKnown(data.CustomerID) {
		filters["customer"] = map[string]interface{}{"id": map[string]string{"exact": data.CustomerID.ValueString()}}
	}

	var result struct {
		Data struct {
			client.NodeGQL
			Name     string         `json:"name"`
			Customer client.NodeGQL `json:"customer"`
		}
	}
	if err := d.client.Do(
		ctx,
		client.GQLRequest{Query: domainQuery, Variables: map[string]interface{}{"filters": filters}, Operation: "get"},
		&result,
		&client.DoOpts{Diags: &resp.Diagnostics},
	); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	data.ID = types.StringValue(result.Data.ID)
	data.Name = types.StringValue(result.Data.Name)
	data.CustomerID = types.StringValue(result.Data.Customer.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
