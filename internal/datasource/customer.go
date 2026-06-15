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

var _ datasource.DataSourceWithConfigure = &customerDataSource{}

func NewCustomerDataSource() datasource.DataSource { return &customerDataSource{} }

type customerDataSource struct{ client *client.OCPClient }

func (d *customerDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_customer"
}
func (d *customerDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*client.OCPClient)
}
func (d *customerDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Represents customer/tenant of OCP platform." +
			" Serves as a root aggregation point for most resources belonging to customer, notably:" +
			" `Domain`, `SeparationPod`, `Project`, `DataProtectionPolicy`, `Network`, `ProvisioningTemplate`," +
			" `SchedulePolicy`, `Tag`, `Template`, `VirtualHost` & `Workflow`." +
			" Houses configurations for billing, Service NOW resouse lifecycle management, Virtual Host" +
			" deployment & antivirus.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Optional: true, Computed: true},
			"prefix": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Natural unique customer identificator",
			},
			"name": schema.StringAttribute{Optional: true, Computed: true},
		},
	}
}

type customerDataSourceModel struct {
	ID     types.String `tfsdk:"id"`
	Prefix types.String `tfsdk:"prefix"`
	Name   types.String `tfsdk:"name"`
}

const customerQuery = `
query get($id: GlobalID, $filters: CustomerFilter, $required: Boolean! = true) {
  data: customer(id: $id, filters: $filters, required: $required) { id name prefix }
}
`

func (d *customerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data customerDataSourceModel
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
	if utils.IsKnown(data.Prefix) {
		filters["prefix"] = map[string]string{"exact": data.Prefix.ValueString()}
	}

	var res struct {
		Data struct {
			client.NodeGQL
			Prefix string `json:"prefix"`
			Name   string `json:"name"`
		}
	}
	if err := d.client.Do(
		ctx,
		client.GQLRequest{Query: customerQuery, Variables: map[string]interface{}{"filters": filters}, Operation: "get"},
		&res,
		&client.DoOpts{Diags: &resp.Diagnostics},
	); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	data.ID = types.StringValue(res.Data.ID)
	data.Prefix = types.StringValue(res.Data.Prefix)
	data.Name = types.StringValue(res.Data.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
