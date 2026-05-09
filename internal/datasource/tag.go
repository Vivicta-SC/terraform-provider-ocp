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

var _ datasource.DataSourceWithConfigure = &tagDataSource{}

func NewTagDataSource() datasource.DataSource {
	return &tagDataSource{}
}

type tagDataSource struct{ client *client.OCPClient }

func (d *tagDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag"
}
func (d *tagDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*client.OCPClient)
}
func (d *tagDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Optional: true, Computed: true},
			"customer_id": schema.StringAttribute{Optional: true, Computed: true},
			"name":        schema.StringAttribute{Optional: true, Computed: true},
			"content":     schema.StringAttribute{Optional: true, Computed: true},
		},
	}
}

type tagDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	CustomerID types.String `tfsdk:"customer_id"`
	Name       types.String `tfsdk:"name"`
	Content    types.String `tfsdk:"content"`
}

const tagQuery = `
query get($id: GlobalID, $filters: TagFilter, $required: Boolean! = true) {
  data: tag(id: $id, filters: $filters, required: $required) {
    id
    name
    content
    customer {
      id
    }
  }
}
`

func (d *tagDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data tagDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	filters := map[string]interface{}{}
	if utils.IsKnown(data.ID) {
		filters["id"] = map[string]string{"exact": data.ID.ValueString()}
	}
	if utils.IsKnown(data.CustomerID) {
		filters["customer"] = map[string]interface{}{"id": map[string]string{"exact": data.CustomerID.ValueString()}}
	}
	if utils.IsKnown(data.Name) {
		filters["name"] = map[string]string{"iExact": data.Name.ValueString()}
	}
	if utils.IsKnown(data.Content) {
		filters["content"] = map[string]string{"iExact": data.Content.ValueString()}
	}

	var result struct {
		Data struct {
			client.NodeGQL
			Name     string
			Content  string
			Customer client.NodeGQL
		}
	}
	if err := d.client.Do(
		ctx,
		client.GQLRequest{Query: tagQuery, Variables: map[string]interface{}{"filters": filters}, Operation: "get"},
		&result,
		&client.DoOpts{Diags: &resp.Diagnostics},
	); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	data.ID = types.StringValue(result.Data.ID)
	data.CustomerID = types.StringValue(result.Data.Customer.ID)
	data.Name = types.StringValue(result.Data.Name)
	data.Content = types.StringValue(result.Data.Content)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
