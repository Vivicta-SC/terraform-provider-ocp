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

var _ datasource.DataSourceWithConfigure = &templateDataSource{}

func NewTemplateDataSource() datasource.DataSource { return &templateDataSource{} }

type templateDataSource struct{ client *client.OCPClient }

func (d *templateDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_template"
}
func (d *templateDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*client.OCPClient)
}
func (d *templateDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":   schema.StringAttribute{Optional: true, Computed: true},
			"name": schema.StringAttribute{Optional: true, Computed: true},
		},
	}
}

type templateDataSourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

const templateQuery = `
query get($id: GlobalID, $filters: TemplateFilter, $required: Boolean! = true) {
  data: template(id: $id, filters: $filters, required: $required) { id name }
}
`

func (d *templateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data templateDataSourceModel
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

	var res struct {
		Data struct {
			client.NodeGQL
			Name string `json:"name"`
		}
	}
	if err := d.client.Do(
		ctx,
		client.GQLRequest{Query: templateQuery, Variables: map[string]interface{}{"filters": filters}, Operation: "get"},
		&res,
		&client.DoOpts{Diags: &resp.Diagnostics},
	); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	data.ID = types.StringValue(res.Data.ID)
	data.Name = types.StringValue(res.Data.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
