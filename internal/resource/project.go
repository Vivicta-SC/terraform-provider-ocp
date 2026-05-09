// Copyright @ Vivicta. All Rights Reserved. 2026
// SPDX-License-Identifier: MPL-2.0

package resource

import (
	"context"

	"github.com/Vivicta-SC/terraform-provider-ocp/internal/client"
	"github.com/Vivicta-SC/terraform-provider-ocp/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.ResourceWithConfigure = &projectResource{}
var _ resource.ResourceWithImportState = &projectResource{}

func NewProjectResource() resource.Resource { return &projectResource{} }

type projectResource struct{ client *client.OCPClient }

func (r *projectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}
func (r *projectResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*client.OCPClient)
}
func (r *projectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"customer_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"separation_pod_id": schema.StringAttribute{
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
		},
	}
}

type projectResourceModel struct {
	ID              types.String `tfsdk:"id"`
	CustomerID      types.String `tfsdk:"customer_id"`
	SeparationPodID types.String `tfsdk:"separation_pod_id"`
	Name            types.String `tfsdk:"name"`
	Note            types.String `tfsdk:"note"`
}

func (s *projectResourceModel) fromGQL(data *client.ProjectGQL) {
	s.ID = types.StringValue(data.ID)
	s.CustomerID = types.StringValue(data.Customer.ID)
	s.SeparationPodID = types.StringValue(data.SeparationPod.ID)
	s.Name = types.StringValue(data.Name)
	s.Note = types.StringValue(data.Note)
}

func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data projectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := map[string]interface{}{
		"customer":      data.CustomerID.ValueString(),
		"separationPod": data.SeparationPodID.ValueString(),
		"name":          data.Name.ValueString(),
		"note":          data.Note.ValueString(),
	}

	var res client.ProjectGQL
	if err := r.client.DoMutate(
		ctx,
		client.GQLRequest{Query: client.ProjectQuery, Variables: map[string]interface{}{"input": input}, Operation: "create"},
		&res,
		&client.DoOpts{Diags: &resp.Diagnostics},
	); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	data.fromGQL(&res)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data projectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var res struct{ Data client.ProjectGQL }
	if err := r.client.Do(
		ctx,
		client.GQLRequest{
			Query:     client.ProjectQuery,
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

func (r *projectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state projectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := map[string]interface{}{
		"project": data.ID.ValueString(),
	}

	if utils.HasChangedWith(data.Name, state.Name) {
		input["name"] = data.Name.ValueString()
	}
	if utils.HasChangedWith(data.Note, state.Note) {
		input["note"] = data.Note.ValueString()
	}

	var res client.ProjectGQL
	if err := r.client.DoMutate(
		ctx,
		client.GQLRequest{Query: client.ProjectQuery, Variables: map[string]interface{}{"input": input}, Operation: "update"},
		&res,
		&client.DoOpts{Diags: &resp.Diagnostics},
	); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	data.fromGQL(&res)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data projectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DoMutate(
		ctx,
		client.GQLRequest{
			Query:     client.ProjectQuery,
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
