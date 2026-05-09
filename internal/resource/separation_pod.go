// Copyright @ Vivicta. All Rights Reserved. 2026
// SPDX-License-Identifier: MPL-2.0

package resource

import (
	"context"

	"github.com/Vivicta-SC/terraform-provider-ocp/internal/client"
	"github.com/Vivicta-SC/terraform-provider-ocp/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.ResourceWithConfigure = &separationPodResource{}
var _ resource.ResourceWithImportState = &separationPodResource{}

func NewSeparationPodResource() resource.Resource { return &separationPodResource{} }

type separationPodResource struct{ client *client.OCPClient }

func (r *separationPodResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_separation_pod"
}
func (r *separationPodResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*client.OCPClient)
}
func (r *separationPodResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"name": schema.StringAttribute{Required: true},
			"note": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"solution": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Default:       stringdefault.StaticString("OCP"),
				Validators:    []validator.String{stringvalidator.OneOf("OCP", "TDL", "CAAS")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"allow_shared_primary_cluster": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"allow_shared_secondary_cluster": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"os_distributions": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
			},
			"data_protection_policy_ids": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
			},
			"dedicated_cluster_ids": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
			},
			"domain_ids": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
			},
			"network_ids": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
			},
			"patching_window_ids": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
			},
			"tier_ids": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
			},
			"workflow_ids": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
			},

			// auto-addition of object is not really compatible with TF's IaaC approach - disable
			"add_new_data_protection_policies": schema.BoolAttribute{
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"add_new_dedicated_clusters": schema.BoolAttribute{
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"add_new_domains": schema.BoolAttribute{
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"add_new_networks": schema.BoolAttribute{
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"add_new_os_distributions": schema.BoolAttribute{
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"add_new_patching_windows": schema.BoolAttribute{
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"add_new_tiers": schema.BoolAttribute{
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"add_new_workflows": schema.BoolAttribute{
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
		},
	}
}

type separationPodResourceModel struct {
	ID         types.String `tfsdk:"id"`
	CustomerID types.String `tfsdk:"customer_id"`
	Name       types.String `tfsdk:"name"`
	Note       types.String `tfsdk:"note"`
	Solution   types.String `tfsdk:"solution"`

	AllowSharedPrimaryCluster   types.Bool `tfsdk:"allow_shared_primary_cluster"`
	AllowSharedSecondaryCluster types.Bool `tfsdk:"allow_shared_secondary_cluster"`

	AddNewDataProtectionPolicies types.Bool `tfsdk:"add_new_data_protection_policies"`
	AddNewDedicatedClusters      types.Bool `tfsdk:"add_new_dedicated_clusters"`
	AddNewDomains                types.Bool `tfsdk:"add_new_domains"`
	AddNewNetworks               types.Bool `tfsdk:"add_new_networks"`
	AddNewOsDistributions        types.Bool `tfsdk:"add_new_os_distributions"`
	AddNewPatchingWindows        types.Bool `tfsdk:"add_new_patching_windows"`
	AddNewWorkflows              types.Bool `tfsdk:"add_new_workflows"`
	AddNewTiers                  types.Bool `tfsdk:"add_new_tiers"`

	OSDistributions         types.Set `tfsdk:"os_distributions"`
	DataProtectionPolicyIDs types.Set `tfsdk:"data_protection_policy_ids"`
	DedicatedClusterIDs     types.Set `tfsdk:"dedicated_cluster_ids"`
	DomainIDs               types.Set `tfsdk:"domain_ids"`
	NetworkIDs              types.Set `tfsdk:"network_ids"`
	PatchingWindowIDs       types.Set `tfsdk:"patching_window_ids"`
	TierIDs                 types.Set `tfsdk:"tier_ids"`
	WorkflowIDs             types.Set `tfsdk:"workflow_ids"`
}

func (s *separationPodResourceModel) fromGQL(ctx context.Context, data *client.SeparationPodGQL, diags *diag.Diagnostics) {
	s.ID = types.StringValue(data.ID)
	s.CustomerID = types.StringValue(data.Customer.ID)
	s.Name = types.StringValue(data.Name)
	s.Note = types.StringValue(data.Note)
	s.Solution = types.StringValue(data.Solution)

	s.AllowSharedPrimaryCluster = types.BoolValue(data.AllowSharedPrimaryCluster)
	s.AllowSharedSecondaryCluster = types.BoolValue(data.AllowSharedSecondaryCluster)

	s.AddNewDataProtectionPolicies = types.BoolValue(data.AddNewDataProtectionPolicies)
	s.AddNewDedicatedClusters = types.BoolValue(data.AddNewDedicatedClusters)
	s.AddNewDomains = types.BoolValue(data.AddNewDomains)
	s.AddNewNetworks = types.BoolValue(data.AddNewNetworks)
	s.AddNewTiers = types.BoolValue(data.AddNewTiers)
	s.AddNewOsDistributions = types.BoolValue(data.AddNewOsDistributions)
	s.AddNewPatchingWindows = types.BoolValue(data.AddNewPatchingWindows)
	s.AddNewWorkflows = types.BoolValue(data.AddNewWorkflows)

	var diags_ diag.Diagnostics
	s.OSDistributions, diags_ = types.SetValueFrom(ctx, types.StringType, data.OSDistributions)
	if diags_.HasError() {
		diags.Append(diags_...)
	}
	s.DataProtectionPolicyIDs = utils.GetTFSetFromGQLConnection(ctx, data.DataProtectionPolicies, diags)
	s.DedicatedClusterIDs = utils.GetTFSetFromGQLConnection(ctx, data.DedicatedClusters, diags)
	s.DomainIDs = utils.GetTFSetFromGQLConnection(ctx, data.Domains, diags)
	s.NetworkIDs = utils.GetTFSetFromGQLConnection(ctx, data.Networks, diags)
	s.PatchingWindowIDs = utils.GetTFSetFromGQLConnection(ctx, data.PatchingWindows, diags)
	s.TierIDs = utils.GetTFSetFromGQLConnection(ctx, data.Tiers, diags)
	s.WorkflowIDs = utils.GetTFSetFromGQLConnection(ctx, data.Workflows, diags)
}

func (r *separationPodResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data separationPodResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := map[string]interface{}{
		"name":         data.Name.ValueString(),
		"customer":     data.CustomerID.ValueString(),
		"solutionType": data.Solution.ValueString(),
		"note":         data.Note.ValueString(),

		"addNewDataProtectionPolicies": data.AddNewDataProtectionPolicies.ValueBool(),
		"addNewDedicatedClusters":      data.AddNewDedicatedClusters.ValueBool(),
		"addNewDomains":                data.AddNewDomains.ValueBool(),
		"addNewNetworks":               data.AddNewNetworks.ValueBool(),
		"addNewOsDistributions":        data.AddNewOsDistributions.ValueBool(),
		"addNewPatchingWindows":        data.AddNewPatchingWindows.ValueBool(),
		"addNewTiers":                  data.AddNewTiers.ValueBool(),
		"addNewWorkflows":              data.AddNewWorkflows.ValueBool(),

		"allowSharedPrimaryCluster":   data.AllowSharedPrimaryCluster.ValueBool(),
		"allowSharedSecondaryCluster": data.AllowSharedSecondaryCluster.ValueBool(),

		"osDistributionList":       utils.FromTFStringSetToGo(ctx, data.OSDistributions, &resp.Diagnostics),
		"dataProtectionPolicyList": utils.FromTFStringSetToGo(ctx, data.DataProtectionPolicyIDs, &resp.Diagnostics),
		"dedicatedClusterList":     utils.FromTFStringSetToGo(ctx, data.DedicatedClusterIDs, &resp.Diagnostics),
		"domainList":               utils.FromTFStringSetToGo(ctx, data.DomainIDs, &resp.Diagnostics),
		"networkList":              utils.FromTFStringSetToGo(ctx, data.NetworkIDs, &resp.Diagnostics),
		"patchingWindowList":       utils.FromTFStringSetToGo(ctx, data.PatchingWindowIDs, &resp.Diagnostics),
		"tierList":                 utils.FromTFStringSetToGo(ctx, data.TierIDs, &resp.Diagnostics),
		"workflowList":             utils.FromTFStringSetToGo(ctx, data.WorkflowIDs, &resp.Diagnostics),
	}
	if resp.Diagnostics.HasError() {
		return
	}

	var res client.SeparationPodGQL
	if err := r.client.DoMutate(
		ctx,
		client.GQLRequest{Query: client.SeparationPodQuery, Variables: map[string]interface{}{"input": input}, Operation: "create"},
		&res,
		&client.DoOpts{Diags: &resp.Diagnostics},
	); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	data.fromGQL(ctx, &res, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *separationPodResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data separationPodResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var res struct{ Data client.SeparationPodGQL }
	if err := r.client.Do(
		ctx,
		client.GQLRequest{
			Query:     client.SeparationPodQuery,
			Variables: map[string]interface{}{"id": data.ID.ValueString()},
			Operation: "get",
		},
		&res,
		&client.DoOpts{Diags: &resp.Diagnostics},
	); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	data.fromGQL(ctx, &res.Data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *separationPodResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *separationPodResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state separationPodResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := map[string]interface{}{
		"separationPod": data.ID.ValueString(),
	}

	if utils.HasChangedWith(data.Name, state.Name) {
		input["name"] = data.Name.ValueString()
	}
	if utils.HasChangedWith(data.Note, state.Note) {
		input["note"] = data.Note.ValueString()
	}

	if utils.HasChangedWith(data.AddNewNetworks, state.AddNewNetworks) {
		input["addNewNetworks"] = data.AddNewNetworks.ValueBool()
	}
	if utils.HasChangedWith(data.AddNewOsDistributions, state.AddNewOsDistributions) {
		input["addNewOsDistributions"] = data.AddNewOsDistributions.ValueBool()
	}
	if utils.HasChangedWith(data.AddNewPatchingWindows, state.AddNewPatchingWindows) {
		input["addNewPatchingWindows"] = data.AddNewPatchingWindows.ValueBool()
	}
	if utils.HasChangedWith(data.AddNewDomains, state.AddNewDomains) {
		input["addNewDomains"] = data.AddNewDomains.ValueBool()
	}
	if utils.HasChangedWith(data.AddNewWorkflows, state.AddNewWorkflows) {
		input["addNewWorkflows"] = data.AddNewWorkflows.ValueBool()
	}
	if utils.HasChangedWith(data.AddNewDedicatedClusters, state.AddNewDedicatedClusters) {
		input["addNewDedicatedClusters"] = data.AddNewDedicatedClusters.ValueBool()
	}
	if utils.HasChangedWith(data.AddNewDataProtectionPolicies, state.AddNewDataProtectionPolicies) {
		input["addNewDataProtectionPolicies"] = data.AddNewDataProtectionPolicies.ValueBool()
	}

	if utils.HasChangedWith(data.AllowSharedPrimaryCluster, state.AllowSharedPrimaryCluster) {
		input["allowSharedPrimaryCluster"] = data.AllowSharedPrimaryCluster.ValueBool()
	}
	if utils.HasChangedWith(data.AllowSharedSecondaryCluster, state.AllowSharedSecondaryCluster) {
		input["allowSharedSecondaryCluster"] = data.AllowSharedSecondaryCluster.ValueBool()
	}

	if utils.HasChangedWith(data.OSDistributions, state.OSDistributions) {
		input["osDistributionList"] = utils.FromTFStringSetToGo(ctx, data.OSDistributions, &resp.Diagnostics)
	}
	if utils.HasChangedWith(data.DataProtectionPolicyIDs, state.DataProtectionPolicyIDs) {
		input["dataProtectionPolicyList"] = utils.FromTFStringSetToGo(ctx, data.DataProtectionPolicyIDs, &resp.Diagnostics)
	}
	if utils.HasChangedWith(data.DedicatedClusterIDs, state.DedicatedClusterIDs) {
		input["dedicatedClusterList"] = utils.FromTFStringSetToGo(ctx, data.DedicatedClusterIDs, &resp.Diagnostics)
	}
	if utils.HasChangedWith(data.DomainIDs, state.DomainIDs) {
		input["domainList"] = utils.FromTFStringSetToGo(ctx, data.DomainIDs, &resp.Diagnostics)
	}
	if utils.HasChangedWith(data.NetworkIDs, state.NetworkIDs) {
		input["networkList"] = utils.FromTFStringSetToGo(ctx, data.NetworkIDs, &resp.Diagnostics)
	}
	if utils.HasChangedWith(data.PatchingWindowIDs, state.PatchingWindowIDs) {
		input["patchingWindowList"] = utils.FromTFStringSetToGo(ctx, data.PatchingWindowIDs, &resp.Diagnostics)
	}
	if utils.HasChangedWith(data.TierIDs, state.TierIDs) {
		input["tierList"] = utils.FromTFStringSetToGo(ctx, data.TierIDs, &resp.Diagnostics)
	}
	if utils.HasChangedWith(data.WorkflowIDs, state.WorkflowIDs) {
		input["workflowList"] = utils.FromTFStringSetToGo(ctx, data.WorkflowIDs, &resp.Diagnostics)
	}

	var res client.SeparationPodGQL
	if err := r.client.DoMutate(
		ctx,
		client.GQLRequest{Query: client.SeparationPodQuery, Variables: map[string]interface{}{"input": input}, Operation: "update"},
		&res,
		&client.DoOpts{Diags: &resp.Diagnostics},
	); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	data.fromGQL(ctx, &res, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *separationPodResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data separationPodResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DoMutate(
		ctx,
		client.GQLRequest{
			Query:     client.SeparationPodQuery,
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
