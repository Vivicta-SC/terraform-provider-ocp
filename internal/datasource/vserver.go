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

var _ datasource.DataSourceWithConfigure = &vserverDataSource{}

func NewVserverDataSource() datasource.DataSource { return &vserverDataSource{} }

type vserverDataSource struct{ client *client.OCPClient }

func (d *vserverDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vserver"
}
func (d *vserverDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*client.OCPClient)
}
func (d *vserverDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Optional: true, Computed: true},
			"customer_id": schema.StringAttribute{Optional: true, Computed: true},
			"name":        schema.StringAttribute{Optional: true, Computed: true},
			"cluster_type": schema.StringAttribute{
				Description: "Defaults to `PRIMARY`",
				Optional:    true,
				Computed:    true,
				Validators:  []validator.String{stringvalidator.OneOf("PRIMARY", "DR_BACKUP")},
			},
			"solution": schema.StringAttribute{
				Description: "Defaults to `OCP`",
				Optional:    true,
				Computed:    true,
				Validators:  []validator.String{stringvalidator.OneOf("OCP", "TDL", "CAAS")},
			},
			"region": schema.StringAttribute{
				Optional:   true,
				Computed:   true,
				Validators: []validator.String{stringvalidator.OneOf("SWEDEN", "NORWAY", "FINLAND")},
			},
		},
	}
}

type vserverDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	CustomerID types.String `tfsdk:"customer_id"`
	CluterType types.String `tfsdk:"cluster_type"`
	Solution   types.String `tfsdk:"solution"`
	Region     types.String `tfsdk:"region"`
}

const vserverQuery = `
query get($id: GlobalID, $filters: VserverFilter, $required: Boolean! = true) {
  data: vserver(id: $id, filters: $filters, required: $required) {
    id
    name
	customer { id }
    storageCluster { storageType }
    region
    solutionType
  }
}
`

func (d *vserverDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data vserverDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.CluterType.IsNull() {
		data.CluterType = types.StringValue("PRIMARY")
	}
	if data.Solution.IsNull() {
		data.Solution = types.StringValue("OCP")
	}

	filters := map[string]interface{}{
		"DISTINCT":       true,
		"isDeleted":      map[string]bool{"exact": false},
		"type":           map[string]string{"exact": "STAAS"},
		"storageCluster": map[string]interface{}{"storageType": map[string]string{"exact": data.CluterType.ValueString()}},
	}

	if utils.IsKnown(data.ID) {
		filters["id"] = map[string]string{"exact": data.ID.ValueString()}
	}
	if utils.IsKnown(data.Name) {
		filters["name"] = map[string]string{"exact": data.Name.ValueString()}
	}
	if utils.IsKnown(data.CustomerID) {
		filters["customer"] = map[string]interface{}{"id": map[string]string{"exact": data.CustomerID.ValueString()}}
	}
	if utils.IsKnown(data.Solution) {
		filters["solutionType"] = map[string]string{"exact": data.Solution.ValueString()}
	}
	if utils.IsKnown(data.Region) {
		filters["storageCluster"].(map[string]interface{})["core"] = map[string]interface{}{
			"region": map[string]string{"exact": data.Region.ValueString()},
		}
	}

	var res struct {
		Data struct {
			client.NodeGQL
			Name           string
			Customer       client.NodeGQL
			StorageCluster struct{ storageType string }
			Region         string
			SolutionType   string
		}
	}
	if err := d.client.Do(
		ctx,
		client.GQLRequest{Query: vserverQuery, Variables: map[string]interface{}{"filters": filters}, Operation: "get"},
		&res,
		&client.DoOpts{Diags: &resp.Diagnostics},
	); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	data.ID = types.StringValue(res.Data.ID)
	data.Name = types.StringValue(res.Data.Name)
	data.CustomerID = types.StringValue(res.Data.Customer.ID)
	data.CluterType = types.StringValue(res.Data.StorageCluster.storageType)
	data.Solution = types.StringValue(res.Data.SolutionType)
	data.Region = types.StringValue(res.Data.Region)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
