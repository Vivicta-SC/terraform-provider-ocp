// Copyright @ Vivicta. All Rights Reserved. 2026
// SPDX-License-Identifier: MPL-2.0

package resource

import (
	"context"
	"fmt"

	"github.com/Vivicta-SC/terraform-provider-ocp/internal/client"
	"github.com/Vivicta-SC/terraform-provider-ocp/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.ResourceWithConfigure = &staasVolumeResource{}
var _ resource.ResourceWithImportState = &staasVolumeResource{}

func NewSTAASVolumeResource() resource.Resource { return &staasVolumeResource{} }

type staasVolumeResource struct{ client *client.OCPClient }

func (r *staasVolumeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_staas_volume"
}
func (r *staasVolumeResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*client.OCPClient)
}
func (r *staasVolumeResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "StaaS volume allows creation of NAS storage.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"data_protection_policy_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"tier_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"vserver_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description: "Vserver ID (`ocp_vserver`), Volume will be created in." +
					" Vserver needs to be of `STAAS` type and  it's StorageClusterType" +
					" must be either `PRIMARY` or `DR_BACKUP` (hosting in secodary DC).",
			},
			"protocol": schema.StringAttribute{
				Description:   "Allowed values: `ISCSI` & `NFS`.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.OneOf("ISCSI", "NFS")},
			},
			"nfs_exports": schema.SetNestedAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.Set{setplanmodifier.RequiresReplace()},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ip_id":     schema.StringAttribute{Optional: true},
						"subnet_id": schema.StringAttribute{Optional: true},
					},
				},
			},
			"size_gb": schema.Int32Attribute{
				Required:      true,
				PlanModifiers: []planmodifier.Int32{int32planmodifier.RequiresReplace()},
			},
			"note": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},

			"deletion_primary_retention_days": schema.Int32Attribute{
				Description: "Retention for Volume in primary site after deletion." +
					" Volumes deleted with retention will block project removal",
				Optional: true,
				Computed: true,
				Default:  int32default.StaticInt32(0),
			},
			"timeouts": timeoutAttribute(ctx, "20m", "15s", "20m", "20m"),
		},
	}
}

var nfsExportObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"ip_id":     types.StringType,
		"subnet_id": types.StringType,
	},
}

type nfsExportModel struct {
	IPID     types.String `tfsdk:"ip_id"`
	SubnetID types.String `tfsdk:"subnet_id"`
}

type staasVolumeResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	ProjectID              types.String `tfsdk:"project_id"`
	DataProtectionPolicyID types.String `tfsdk:"data_protection_policy_id"`
	TierID                 types.String `tfsdk:"tier_id"`
	VserverID              types.String `tfsdk:"vserver_id"`
	Protocol               types.String `tfsdk:"protocol"`
	SizeGB                 types.Int32  `tfsdk:"size_gb"`
	Note                   types.String `tfsdk:"note"`
	NfsExports             types.Set    `tfsdk:"nfs_exports"`

	DeletionPrimaryRetentionDays types.Int32   `tfsdk:"deletion_primary_retention_days"`
	Timeouts                     timeoutsModel `tfsdk:"timeouts"`
}

func (s *staasVolumeResourceModel) fromGQL(ctx context.Context, data *client.StaasVolumeGQL) diag.Diagnostics {
	var diags diag.Diagnostics

	s.ID = types.StringValue(data.ID)
	s.Name = types.StringValue(data.Name)
	s.ProjectID = types.StringValue(data.Project.ID)
	s.DataProtectionPolicyID = types.StringValue(data.DataProtectionPolicy.ID)
	s.TierID = types.StringValue(data.Tier.ID)
	s.VserverID = types.StringValue(data.Vserver.ID)
	s.Protocol = types.StringValue(data.Protocol)
	s.Note = types.StringValue(data.Note)

	if len(data.Visibility) <= 0 {
		return diags
	}
	nfsExports := make([]nfsExportModel, 0, len(data.Visibility))
	for _, visibilityGQL := range data.Visibility {
		var export nfsExportModel
		switch visibilityGQL.Typename {
		case "IpAddressNode":
			export.IPID = types.StringValue(visibilityGQL.ID)
		case "SubnetNode":
			export.SubnetID = types.StringValue(visibilityGQL.ID)
		}
		nfsExports = append(nfsExports, export)
	}
	nfsExportsSet, diag := types.SetValueFrom(ctx, nfsExportObjectType, nfsExports)
	diags.Append(diag...)
	s.NfsExports = nfsExportsSet

	return diags
}

func (r *staasVolumeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data staasVolumeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	timeout, _diags := data.Timeouts.Create.ValueGoDuration()
	resp.Diagnostics.Append(_diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	var vserverRes struct{ Data client.VserverGQL }
	if err := r.client.Do(
		ctx,
		client.GQLRequest{
			Query:     client.VserverQuery,
			Variables: map[string]interface{}{"id": data.VserverID.ValueString()},
			Operation: "get",
		},
		&vserverRes,
		&client.DoOpts{Diags: &resp.Diagnostics},
	); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	nfsExports := make([]interface{}, 0, len(data.NfsExports.Elements()))
	if len(data.NfsExports.Elements()) > 0 {
		exports := make([]nfsExportModel, 0, len(data.NfsExports.Elements()))
		resp.Diagnostics.Append(data.NfsExports.ElementsAs(ctx, &exports, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		for _, nfs_export := range exports {
			export := map[string]interface{}{}
			if utils.IsKnown(nfs_export.IPID) {
				export["ipAddress"] = nfs_export.IPID.ValueString()
			}
			if utils.IsKnown(nfs_export.SubnetID) {
				export["subnet"] = nfs_export.SubnetID.ValueString()
			}
			nfsExports = append(nfsExports, export)
		}
	}

	input := map[string]interface{}{
		"staasGroupOrStandalone": map[string]interface{}{"standalone": map[string]interface{}{
			"dataProtectionPolicy": data.DataProtectionPolicyID.ValueString(),
			"project":              data.ProjectID.ValueString(),
			"protocol":             data.Protocol.ValueString(),
			"vserver":              data.VserverID.ValueString(),
			"tier":                 data.TierID.ValueString(),
			"region":               vserverRes.Data.Region,
			"hostedInSecondaryDc":  vserverRes.Data.StorageCluster.StorageType != "PRIMARY",
			"nfsExportList":        nfsExports,
		}},
		"sizeGB": data.SizeGB.ValueInt32(),
		"note":   data.Note.ValueString(),
	}

	var res struct {
		TaskExecution struct{ ID string }
		Volume        client.StaasVolumeGQL
	}
	if err := r.client.DoMutate(
		ctx,
		client.GQLRequest{Query: client.StaasVolumeQuery, Variables: map[string]interface{}{"input": input}, Operation: "create"},
		&res,
		&client.DoOpts{Diags: &resp.Diagnostics},
	); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	resp.Diagnostics.Append(data.fromGQL(ctx, &res.Volume)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Info(ctx, fmt.Sprintf("Awaiting staas_volume creation task with timeout: %s", timeout.String()))
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := r.client.AwaitTask(ctx, res.TaskExecution.ID, &client.DoOpts{Diags: &resp.Diagnostics}); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
	}
}

func (r *staasVolumeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data staasVolumeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	timeout, _diags := data.Timeouts.Read.ValueGoDuration()
	resp.Diagnostics.Append(_diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var res struct{ Data client.StaasVolumeGQL }
	if err := r.client.Do(
		ctx,
		client.GQLRequest{
			Query:     client.StaasVolumeQuery,
			Variables: map[string]interface{}{"id": data.ID.ValueString()},
			Operation: "get",
		},
		&res,
		&client.DoOpts{Diags: &resp.Diagnostics},
	); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	resp.Diagnostics.Append(data.fromGQL(ctx, &res.Data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *staasVolumeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *staasVolumeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state staasVolumeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	timeout, _diags := plan.Timeouts.Update.ValueGoDuration()
	resp.Diagnostics.Append(_diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	if utils.HasChangedWith(plan.SizeGB, state.SizeGB) {
		input := map[string]interface{}{
			"volume": plan.ID.ValueString(),
			"sizeGB": plan.SizeGB.ValueInt32(),
		}

		var op string
		if state.Protocol.ValueString() == "ISCSI" {
			op = "resizeISCSI"
		} else {
			op = "resizeNAS"
		}

		var res client.NodeGQL
		if err := r.client.DoMutate(
			ctx,
			client.GQLRequest{Query: client.StaasVolumeQuery, Variables: map[string]interface{}{"input": input}, Operation: op},
			&res,
			&client.DoOpts{Diags: &resp.Diagnostics},
		); err != nil {
			resp.Diagnostics.AddError("Client Error", err.Error())
			return
		}

		tflog.Info(ctx, fmt.Sprintf("Awaiting staas_volume resize task with timeout: %s", timeout.String()))
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		if err := r.client.AwaitTask(ctx, res.ID, &client.DoOpts{Diags: &resp.Diagnostics}); err != nil {
			resp.Diagnostics.AddError("Client Error", err.Error())
		}
	}

	if utils.HasChangedWith(plan.Note, state.Note) {
		input := map[string]interface{}{
			"volume": plan.ID.ValueString(),
			"note":   plan.Note.ValueString(),
		}

		var res client.StaasVolumeGQL
		if err := r.client.DoMutate(
			ctx,
			client.GQLRequest{Query: client.StaasVolumeQuery, Variables: map[string]interface{}{"input": input}, Operation: "update"},
			&res,
			&client.DoOpts{Diags: &resp.Diagnostics},
		); err != nil {
			resp.Diagnostics.AddError("Client Error", err.Error())
			return
		}
	}

	var res struct{ Data client.StaasVolumeGQL }
	if err := r.client.Do(
		ctx,
		client.GQLRequest{
			Query:     client.StaasVolumeQuery,
			Variables: map[string]interface{}{"id": plan.ID.ValueString()},
			Operation: "get",
		},
		&res,
		&client.DoOpts{Diags: &resp.Diagnostics},
	); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	resp.Diagnostics.Append(plan.fromGQL(ctx, &res.Data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *staasVolumeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data staasVolumeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	timeout, _diags := data.Timeouts.Delete.ValueGoDuration()
	resp.Diagnostics.Append(_diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	var res client.NodeGQL
	if err := r.client.DoMutate(
		ctx,
		client.GQLRequest{
			Query:     client.StaasVolumeQuery,
			Variables: map[string]interface{}{"id": data.ID.ValueString(), "retention": data.DeletionPrimaryRetentionDays.ValueInt32()},
			Operation: "delete",
		},
		&res,
		&client.DoOpts{Diags: &resp.Diagnostics},
	); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Awaiting staas_volume deletion task with timeout: %s", timeout.String()))
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := r.client.AwaitTask(ctx, res.ID, &client.DoOpts{Diags: &resp.Diagnostics}); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
	}
}
