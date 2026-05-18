// Copyright @ Vivicta. All Rights Reserved. 2026
// SPDX-License-Identifier: MPL-2.0

package resource

import (
	"context"
	"fmt"
	"time"

	"github.com/Vivicta-SC/terraform-provider-ocp/internal/client"
	"github.com/Vivicta-SC/terraform-provider-ocp/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.ResourceWithConfigure = &staasGroupResource{}
var _ resource.ResourceWithImportState = &staasGroupResource{}

func NewSTAASGroupResource() resource.Resource { return &staasGroupResource{} }

type staasGroupResource struct{ client *client.OCPClient }

func (r *staasGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_staas_group"
}
func (r *staasGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*client.OCPClient)
}
func (r *staasGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Groups individual StaaS volumes together so they share export policy, data protection policy, tier & QOS",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
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
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"note": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"protocol": schema.StringAttribute{
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

type staasGroupResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	ProjectID              types.String `tfsdk:"project_id"`
	DataProtectionPolicyID types.String `tfsdk:"data_protection_policy_id"`
	TierID                 types.String `tfsdk:"tier_id"`
	VserverID              types.String `tfsdk:"vserver_id"`
	Name                   types.String `tfsdk:"name"`
	Note                   types.String `tfsdk:"note"`
	Protocol               types.String `tfsdk:"protocol"`

	NfsExports types.Set `tfsdk:"nfs_exports"`
}

func (s *staasGroupResourceModel) fromGQLWithoutMappings(data *client.StaasGroupGQL) {
	s.ID = types.StringValue(data.ID)
	s.ProjectID = types.StringValue(data.Project.ID)
	s.DataProtectionPolicyID = types.StringValue(data.DataProtectionPolicy.ID)
	s.TierID = types.StringValue(data.Tier.ID)
	s.VserverID = types.StringValue(data.Vserver.ID)
	s.Name = types.StringValue(data.Name)
	s.Note = types.StringValue(data.Note)
	s.Protocol = types.StringValue(data.Protocol)

}

func (s *staasGroupResourceModel) fromGQL(data *client.StaasGroupGQL) {
	s.fromGQLWithoutMappings(data)

}

func (r *staasGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data staasGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := map[string]interface{}{
		"dataProtectionPolicy": data.DataProtectionPolicyID.ValueString(),
		"name":                 data.Name.ValueString(),
		"note":                 data.Note.ValueString(),
		"project":              data.ProjectID.ValueString(),
		"protocol":             data.Protocol.ValueString(),
		"tier":                 data.TierID.ValueString(),
		"vserver":              data.VserverID.ValueString(),
	}

	var resCreate client.StaasGroupGQL
	if err := r.client.DoMutate(
		ctx,
		client.GQLRequest{Query: client.StaasGroupQuery, Variables: map[string]interface{}{"input": input}, Operation: "create"},
		&resCreate,
		&client.DoOpts{Diags: &resp.Diagnostics},
	); err != nil {
		resp.Diagnostics.AddError("Client Error - Creating", err.Error())
		return
	}

	data.fromGQLWithoutMappings(&resCreate)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	if len(data.NfsExports.Elements()) > 0 {
		exports := make([]nfsExportModel, 0, len(data.NfsExports.Elements()))
		resp.Diagnostics.Append(data.NfsExports.ElementsAs(ctx, &exports, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		for _, nfs_export := range exports {
			input := map[string]interface{}{"staasGroup": resCreate.ID}
			if utils.IsKnown(nfs_export.IPID) {
				input["ipAddress"] = nfs_export.IPID.ValueString()
			}
			if utils.IsKnown(nfs_export.SubnetID) {
				input["subnet"] = nfs_export.SubnetID.ValueString()
			}

			var res client.NodeGQL
			if err := r.client.DoMutate(
				ctx,
				client.GQLRequest{Query: client.StaasGroupQuery, Variables: map[string]interface{}{"input": input}, Operation: "addNFSExport"},
				&res,
				&client.DoOpts{Diags: &resp.Diagnostics},
			); err != nil {
				resp.Diagnostics.AddError("Client Error - Adding NFS Export", err.Error())
				return
			}

			updateTimeout := 20 * time.Minute
			tflog.Info(ctx, fmt.Sprintf("Awaiting StaaS NFS add export with timeout: %s", updateTimeout.String()))
			ctx, cancel := context.WithTimeout(ctx, updateTimeout)
			defer cancel()
			if err := r.client.AwaitTask(ctx, res.ID, &client.DoOpts{Diags: &resp.Diagnostics}); err != nil {
				resp.Diagnostics.AddError("Client Error - Awaiting NFS Export Task", err.Error())
			}
		}
	}

	var res struct{ Data client.StaasGroupGQL }
	if err := r.client.Do(
		ctx,
		client.GQLRequest{
			Query:     client.StaasGroupQuery,
			Variables: map[string]interface{}{"id": resCreate.ID},
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

func (r *staasGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data staasGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var res struct{ Data client.StaasGroupGQL }
	if err := r.client.Do(
		ctx,
		client.GQLRequest{
			Query:     client.StaasGroupQuery,
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

func (r *staasGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *staasGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state staasGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := map[string]interface{}{
		"staasGroup": data.ID.ValueString(),
	}

	if utils.HasChangedWith(data.Name, state.Name) {
		input["name"] = data.Name.ValueString()
	}
	if utils.HasChangedWith(data.Note, state.Note) {
		input["note"] = data.Note.ValueString()
	}

	var res client.StaasGroupGQL
	if err := r.client.DoMutate(
		ctx,
		client.GQLRequest{Query: client.StaasGroupQuery, Variables: map[string]interface{}{"input": input}, Operation: "update"},
		&res,
		&client.DoOpts{Diags: &resp.Diagnostics},
	); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	data.fromGQL(&res)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *staasGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data staasGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DoMutate(
		ctx,
		client.GQLRequest{
			Query:     client.StaasGroupQuery,
			Variables: map[string]interface{}{"id": data.ID.ValueString()},
			Operation: "delete",
		},
		nil,
		&client.DoOpts{Diags: &resp.Diagnostics},
	); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
}
