// Copyright @ Vivicta. All Rights Reserved. 2026
// SPDX-License-Identifier: MPL-2.0

package resource

import (
	"context"
	"fmt"
	"time"

	"github.com/Vivicta-SC/terraform-provider-ocp/internal/client"
	"github.com/Vivicta-SC/terraform-provider-ocp/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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
func (r *staasVolumeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "StaaS volume allows creation of NAS storage.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"staas_group_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
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
			"protocol": schema.StringAttribute{Computed: true},
			"primary_retention_deletion_days": schema.Int32Attribute{
				Optional: true,
				Computed: true,
				Default:  int32default.StaticInt32(0),
			},
		},
	}
}

type staasVolumeResourceModel struct {
	ID           types.String `tfsdk:"id"`
	StaasGroupID types.String `tfsdk:"staas_group_id"`
	SizeGB       types.Int32  `tfsdk:"size_gb"`
	Note         types.String `tfsdk:"note"`
	Protocol     types.String `tfsdk:"protocol"`

	PrimaryRetentionDeletionDays types.Int32 `tfsdk:"primary_retention_deletion_days"`
}

func (s *staasVolumeResourceModel) fromGQL(data *client.StaasVolumeGQL) {
	s.ID = types.StringValue(data.ID)
	s.StaasGroupID = types.StringValue(data.StaasGroup.ID)
	s.Protocol = types.StringValue(data.Protocal)
	s.Note = types.StringValue(data.Note)
}

func (r *staasVolumeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data staasVolumeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := map[string]interface{}{
		"staasGroupOrStandalone": map[string]string{"staasGroup": data.StaasGroupID.ValueString()},
		"sizeGB":                 data.SizeGB.ValueInt32(),
		"note":                   data.Note.ValueString(),
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

	data.fromGQL(&res.Volume)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	createTimeout := 20 * time.Minute // TODO: from inputs
	tflog.Info(ctx, fmt.Sprintf("Awaiting staas_volume creation task with timeout: %s", createTimeout.String()))
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()
	if err := r.client.AwaitTask(ctx, res.TaskExecution.ID, &client.DoOpts{Diags: &resp.Diagnostics}); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
	}
}

func (r *staasVolumeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data staasVolumeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

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

	data.fromGQL(&res.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *staasVolumeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *staasVolumeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state staasVolumeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if utils.HasChangedWith(data.SizeGB, state.SizeGB) {
		input := map[string]interface{}{
			"volume": data.ID.ValueString(),
			"sizeGB": data.SizeGB.ValueInt32(),
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

		updateTimeout := 20 * time.Minute // TODO: from inputs
		tflog.Info(ctx, fmt.Sprintf("Awaiting staas_volume resize task with timeout: %s", updateTimeout.String()))
		ctx, cancel := context.WithTimeout(ctx, updateTimeout)
		defer cancel()
		if err := r.client.AwaitTask(ctx, res.ID, &client.DoOpts{Diags: &resp.Diagnostics}); err != nil {
			resp.Diagnostics.AddError("Client Error", err.Error())
		}
	}

	if utils.HasChangedWith(data.Note, state.Note) {
		input := map[string]interface{}{
			"volume": data.ID.ValueString(),
			"note":   data.Note.ValueString(),
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

		data.fromGQL(&res)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *staasVolumeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data staasVolumeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var res client.NodeGQL
	if err := r.client.DoMutate(
		ctx,
		client.GQLRequest{
			Query:     client.StaasVolumeQuery,
			Variables: map[string]interface{}{"id": data.ID.ValueString(), "retention": data.PrimaryRetentionDeletionDays.ValueInt32()},
			Operation: "delete",
		},
		&res,
		&client.DoOpts{Diags: &resp.Diagnostics},
	); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	deleteTimeout := 20 * time.Minute // TODO: from inputs
	tflog.Info(ctx, fmt.Sprintf("Awaiting staas_volume deletion task with timeout: %s", deleteTimeout.String()))
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()
	if err := r.client.AwaitTask(ctx, res.ID, &client.DoOpts{Diags: &resp.Diagnostics}); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
	}
}
