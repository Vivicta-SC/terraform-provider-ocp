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

var _ datasource.DataSourceWithConfigure = &dppDataSource{}

func NewDataProtectionPolicyDataSource() datasource.DataSource {
	return &dppDataSource{}
}

type dppDataSource struct{ client *client.OCPClient }

func (d *dppDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_data_protection_policy"
}
func (d *dppDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*client.OCPClient)
}
func (d *dppDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Represents data protection configuration for primary snapshots, secondary backup & archive." +
			"Applicable to virtual machines and volumes. Corresponds to `DataProtectionPolicyNode` in GraphQL",
		Attributes: map[string]schema.Attribute{
			"id":                     schema.StringAttribute{Optional: true, Computed: true},
			"customer_id":            schema.StringAttribute{Optional: true, Computed: true},
			"note":                   schema.StringAttribute{Optional: true, Computed: true},
			"is_immutable":           schema.BoolAttribute{Optional: true, Computed: true},
			"primary_snapshot_count": schema.Int32Attribute{Optional: true, Computed: true},
			"secondary_backup_count": schema.Int32Attribute{Optional: true, Computed: true},
			"archive_count":          schema.Int32Attribute{Optional: true, Computed: true},
		},
	}
}

type dppDataSourceModel struct {
	ID                   types.String `tfsdk:"id"`
	CustomerID           types.String `tfsdk:"customer_id"`
	Note                 types.String `tfsdk:"note"`
	IsImmutable          types.Bool   `tfsdk:"is_immutable"`
	PrimarySnapshotCount types.Int32  `tfsdk:"primary_snapshot_count"`
	SecondaryBackupCount types.Int32  `tfsdk:"secondary_backup_count"`
	ArchiveCount         types.Int32  `tfsdk:"archive_count"`
}

const dppQuery = `
query get($id: GlobalID, $filters: DataProtectionPolicyFilter, $required: Boolean! = true) {
  data: dataProtectionPolicy(id: $id, filters: $filters, required: $required) {
  	id
	customer { id } 
	note
  }
}
`

func (d *dppDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dppDataSourceModel
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
	if utils.IsKnown(data.Note) {
		filters["note"] = map[string]string{"exact": data.Note.ValueString()}
	}
	if utils.IsKnown(data.IsImmutable) {
		filters["isImmutable"] = map[string]bool{"exact": data.IsImmutable.ValueBool()}
	}
	if utils.IsKnown(data.PrimarySnapshotCount) {
		filters["primarySnapshotCount"] = map[string]int32{"exact": data.PrimarySnapshotCount.ValueInt32()}
	}
	if utils.IsKnown(data.SecondaryBackupCount) {
		filters["secondaryBackupCount"] = map[string]int32{"exact": data.SecondaryBackupCount.ValueInt32()}
	}
	if utils.IsKnown(data.ArchiveCount) {
		filters["archiveCount"] = map[string]int32{"exact": data.ArchiveCount.ValueInt32()}
	}

	var result struct {
		Data struct {
			client.NodeGQL

			Note     string         `json:"name"`
			Customer client.NodeGQL `json:"customer"`
		}
	}
	if err := d.client.Do(
		ctx,
		client.GQLRequest{Query: dppQuery, Variables: map[string]interface{}{"filters": filters}, Operation: "get"},
		&result,
		&client.DoOpts{Diags: &resp.Diagnostics},
	); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	data.ID = types.StringValue(result.Data.ID)
	data.Note = types.StringValue(result.Data.Note)
	data.CustomerID = types.StringValue(result.Data.Customer.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
