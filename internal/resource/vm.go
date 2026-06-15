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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.ResourceWithConfigure = &vmResource{}
var _ resource.ResourceWithImportState = &vmResource{}

func NewVMResource() resource.Resource { return &vmResource{} }

type vmResource struct{ client *client.OCPClient }

func (r *vmResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm"
}
func (r *vmResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*client.OCPClient)
}
func (r *vmResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this resource to manage OCP VirtualHost.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"customer_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"domain_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			// TODO: solve for "latest" template of Distribution?
			"template_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			// TODO: these are technically in-place replacable - blocking for now
			"data_protection_policy_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"project_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"tier_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},

			"hostname": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"note": schema.StringAttribute{
				Required: true, // TODO: why is this required in OCP? (cant even empty string)
			},
			"region": schema.StringAttribute{
				Description:   "Allowed values: `SWEDEN`, `NORWAY` & `FINLAND`",
				Required:      true,
				Validators:    []validator.String{stringvalidator.OneOf("SWEDEN", "NORWAY", "FINLAND")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"cpu_count": schema.Int32Attribute{
				Required: true,
			},
			"cores_per_socket": schema.Int32Attribute{
				Optional: true,
				Computed: true,
				Default:  int32default.StaticInt32(1),
			},
			"memory_size_gb": schema.Int32Attribute{
				Required: true,
			},
			"os_disk_size_gb": schema.Int32Attribute{
				Optional:      true,
				PlanModifiers: []planmodifier.Int32{int32planmodifier.RequiresReplace(), int32planmodifier.UseStateForUnknown()},
			},
			"antivirus": schema.StringAttribute{
				Description:   "Allowed values: `MCAFEE`, `DEFENDER`, `SYMANTEC`, `FSECURE`, `CORTEX` & `NONE`. Defaults to `NONE`",
				Optional:      true,
				Computed:      true,
				Default:       stringdefault.StaticString("NONE"),
				Validators:    []validator.String{stringvalidator.OneOf("MCAFEE", "DEFENDER", "SYMANTEC", "FSECURE", "CORTEX", "NONE")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"join_to_domain": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"cluster_type": schema.StringAttribute{
				Description:   "Allowed values: `PRIMARY` & `SECONDARY`. Defaults to `PRIMARY`",
				Optional:      true,
				Computed:      true,
				Default:       stringdefault.StaticString("PRIMARY"),
				Validators:    []validator.String{stringvalidator.OneOf("PRIMARY", "SECONDARY")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"tag_ids": schema.SetAttribute{
				ElementType:   types.StringType,
				Optional:      true,
				Computed:      true,
				Default:       setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
				PlanModifiers: []planmodifier.Set{setplanmodifier.RequiresReplace()},
			},
			"disks": disksAttribute(),
			"nics":  nicsAttribute(),

			"creation_task_id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"await_deletion_task": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
				Description: "Set to await VM deletion task, otherwise VM will be considered deleted immediatelly." +
					" Only use this, if potential new VM does not use the same resources - IPs, hostname, etc.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"allow_restart": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(false),
				Description:   "Allow OCP restart of VM during resize (lowering cpu/memory)",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"timeouts": timeoutAttribute(ctx, "20m", "15s", "20m", "20m"),
		},
	}
}

type vmResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	CustomerID             types.String `tfsdk:"customer_id"`
	DomainID               types.String `tfsdk:"domain_id"`
	TemplateID             types.String `tfsdk:"template_id"`
	DataProtectionPolicyID types.String `tfsdk:"data_protection_policy_id"`
	ProjectID              types.String `tfsdk:"project_id"`
	TierID                 types.String `tfsdk:"tier_id"`
	Hostname               types.String `tfsdk:"hostname"`
	Note                   types.String `tfsdk:"note"`
	Region                 types.String `tfsdk:"region"`
	CpuCount               types.Int32  `tfsdk:"cpu_count"`
	CoresPerSocket         types.Int32  `tfsdk:"cores_per_socket"`
	MemorySizeGB           types.Int32  `tfsdk:"memory_size_gb"`
	Antivirus              types.String `tfsdk:"antivirus"`
	ClusterType            types.String `tfsdk:"cluster_type"`
	JoinToDomain           types.Bool   `tfsdk:"join_to_domain"`
	OSDiskSizeGB           types.Int32  `tfsdk:"os_disk_size_gb"`

	Tags  types.Set  `tfsdk:"tag_ids"`
	Disks types.List `tfsdk:"disks"`
	NICS  types.List `tfsdk:"nics"`

	CreationTaskID    types.String  `tfsdk:"creation_task_id"`
	AwaitDeletionTask types.Bool    `tfsdk:"await_deletion_task"`
	AllowRestart      types.Bool    `tfsdk:"allow_restart"`
	Timeouts          timeoutsModel `tfsdk:"timeouts"`
}

func (vm *vmResourceModel) intoModelWithoutCreationUnknown(_ context.Context, data *client.VMGQL) diag.Diagnostics {
	var diags diag.Diagnostics

	vm.ID = types.StringValue(data.ID)
	vm.Antivirus = types.StringValue(data.AntivirusType)
	vm.ClusterType = types.StringValue(data.ClusterType)
	vm.CoresPerSocket = types.Int32Value(data.CoresPerSocket)
	vm.CpuCount = types.Int32Value(data.CpuCount)
	vm.Hostname = types.StringValue(data.Hostname)
	vm.MemorySizeGB = types.Int32Value(data.MemorySizeGB)
	vm.Note = types.StringValue(data.Note)
	vm.Region = types.StringValue(data.Region)

	vm.CustomerID = types.StringValue(data.Customer.ID)
	vm.DataProtectionPolicyID = types.StringValue(data.DataProtectionPolicy.ID)
	vm.DomainID = types.StringValue(data.Domain.ID)
	vm.ProjectID = types.StringValue(data.Project.ID)
	vm.TemplateID = types.StringValue(data.Template.ID)
	vm.TierID = types.StringValue(data.Tier.ID)

	return diags
}

func (vm *vmResourceModel) intoModel(ctx context.Context, data *client.VMGQL) diag.Diagnostics {
	var diags diag.Diagnostics
	diags.Append(vm.intoModelWithoutCreationUnknown(ctx, data)...)

	diags.Append(vm.fromNICsGQL(ctx, data.NetworkInterfaceList, data.IpAddressList.GetNodes())...)
	diags.Append(vm.fromDisksGQL(ctx, data.Disks.GetNodes())...)

	tags, _diags := types.SetValueFrom(ctx, types.StringType, data.Tags.GetIDs())
	vm.Tags = tags
	diags.Append(_diags...)

	return diags
}

func (r *vmResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data vmResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	createTimeout, _diags := data.Timeouts.Create.ValueGoDuration()
	resp.Diagnostics.Append(_diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	input := map[string]interface{}{
		"antivirusType":        data.Antivirus.ValueString(),
		"clusterType":          data.ClusterType.ValueString(),
		"coresPerSocket":       data.CoresPerSocket.ValueInt32(),
		"cpuCount":             data.CpuCount.ValueInt32(),
		"customer":             data.CustomerID.ValueString(),
		"dataProtectionPolicy": data.DataProtectionPolicyID.ValueString(),
		"domain":               data.DomainID.ValueString(),
		"hostname":             data.Hostname.ValueString(),
		"joinToDomain":         data.JoinToDomain.ValueBool(),
		"memorySizeGB":         data.MemorySizeGB.ValueInt32(),
		"note":                 data.Note.ValueString(),
		"project":              data.ProjectID.ValueString(),
		"region":               data.Region.ValueString(),
		"skip424Check":         true,
		"template":             data.TemplateID.ValueString(),
		"tier":                 data.TierID.ValueString(),
		"version":              "V2",
		"tagList":              utils.FromTFStringSetToGo(ctx, data.Tags, &resp.Diagnostics),
	}
	if data.Antivirus.ValueString() != "NONE" {
		input["hasAntivirus"] = true
	}
	if utils.IsKnown(data.OSDiskSizeGB) {
		input["osDiskSizeGB"] = data.OSDiskSizeGB.ValueInt32()
	}
	if len(data.NICS.Elements()) > 0 {
		nics := make([]nicModel, 0, len(data.NICS.Elements()))
		resp.Diagnostics.Append(data.NICS.ElementsAs(ctx, &nics, false)...)

		nicsInput := []interface{}{}
		for _, nic := range nics {
			nicInput := map[string]interface{}{
				"network":             nic.NetworkID.ValueString(),
				"autoAssignIp":        nic.AutoAssignIp.ValueBool(),
				"useAsDefaultGateway": nic.UseAsDefaultGateway.ValueBool(),
			}
			if len(nic.IPv4.Elements()) > 0 {
				ipsTF := make([]ipv4Model, 0, len(nic.IPv4.Elements()))
				resp.Diagnostics.Append(data.Disks.ElementsAs(ctx, &ipsTF, false)...)
				ips := make([]string, 0, len(nic.IPv4.Elements()))
				for _, ip := range ipsTF {
					ips = append(ips, ip.IP.ValueString())
				}
				nicInput["ipList"] = ips
			}
			nicsInput = append(nicsInput, nicInput)
		}
		input["interfaceList"] = nicsInput
	}
	if len(data.Disks.Elements()) > 0 {
		disks := make([]diskModel, 0, len(data.Disks.Elements()))
		resp.Diagnostics.Append(data.Disks.ElementsAs(ctx, &disks, false)...)

		disksInput := []interface{}{}
		for _, disk := range disks {
			var letter string
			if disk.WinDiskLetter.ValueString() != "" {
				letter = disk.WinDiskLetter.ValueString()
			}
			disksInput = append(disksInput, map[string]interface{}{
				"sizeGB":             disk.SizeGB.ValueInt32(),
				"allocationUnitSize": disk.AllocationUnitSize.ValueInt32(),
				"winDiskLetter":      letter,
			})
		}
		input["localDiskList"] = disksInput
	}

	if resp.Diagnostics.HasError() {
		return
	}

	var res struct {
		VirtualHost   client.VMGQL
		TaskExecution struct{ ID string }
	}
	if err := r.client.DoMutate(
		ctx,
		client.GQLRequest{Query: client.VMQuery, Variables: map[string]interface{}{"input": input}, Operation: "create"},
		&res,
		&client.DoOpts{Diags: &resp.Diagnostics},
	); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	resp.Diagnostics.Append(data.intoModelWithoutCreationUnknown(ctx, &res.VirtualHost)...)
	data.CreationTaskID = types.StringValue(res.TaskExecution.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Info(ctx, fmt.Sprintf("Awaiting VM creation task with timeout: %s", createTimeout.String()))
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()
	if err := r.client.AwaitTask(ctx, res.TaskExecution.ID, &client.DoOpts{Diags: &resp.Diagnostics}); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
	}

	// Refresh data, that are available only after creation
	var resRead struct{ Data client.VMGQL }
	if err := r.client.Do(
		ctx,
		client.GQLRequest{
			Query:     client.VMQuery,
			Variables: map[string]interface{}{"id": data.ID.ValueString()},
			Operation: "get",
		},
		&resRead,
		&client.DoOpts{Diags: &resp.Diagnostics},
	); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	resp.Diagnostics.Append(data.intoModel(ctx, &resRead.Data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vmResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data vmResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	timeout, _diags := data.Timeouts.Read.ValueGoDuration()
	resp.Diagnostics.Append(_diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var res struct{ Data client.VMGQL }
	if err := r.client.Do(
		ctx,
		client.GQLRequest{
			Query:     client.VMQuery,
			Variables: map[string]interface{}{"id": data.ID.ValueString()},
			Operation: "get",
		},
		&res,
		&client.DoOpts{Diags: &resp.Diagnostics},
	); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	resp.Diagnostics.Append(data.intoModel(ctx, &res.Data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vmResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *vmResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state vmResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	timeout, _diags := plan.Timeouts.Update.ValueGoDuration()
	resp.Diagnostics.Append(_diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	needsRefresh := false

	cpuChanged := utils.HasChangedWith(plan.CpuCount, state.CpuCount)
	coresChanged := utils.HasChangedWith(plan.CoresPerSocket, state.CoresPerSocket)
	memoryChanged := utils.HasChangedWith(plan.MemorySizeGB, state.MemorySizeGB)
	if cpuChanged || coresChanged || memoryChanged {
		needsRefresh = true
		input := map[string]interface{}{
			"virtualHost":  plan.ID.ValueString(),
			"allowRestart": plan.AllowRestart.ValueBool(),
		}
		if cpuChanged {
			input["cpuCount"] = plan.CpuCount.ValueInt32()
		}
		if coresChanged {
			input["cpuCount"] = plan.CoresPerSocket.ValueInt32()
		}
		if memoryChanged {
			input["memorySizeGB"] = plan.MemorySizeGB.ValueInt32()
		}

		var res struct{ ID string }
		if err := r.client.DoMutate(
			ctx,
			client.GQLRequest{Query: client.VMQuery, Variables: map[string]interface{}{"input": input}, Operation: "resize"},
			&res,
			&client.DoOpts{Diags: &resp.Diagnostics},
		); err != nil {
			resp.Diagnostics.AddError("Client Error", err.Error())
			return
		}

		tflog.Info(ctx, fmt.Sprintf("Awaiting VM resize task with timeout: %s", timeout.String()))
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		if err := r.client.AwaitTask(ctx, res.ID, &client.DoOpts{Diags: &resp.Diagnostics}); err != nil {
			resp.Diagnostics.AddError("Client Error", err.Error())
		}
	}

	if utils.HasChangedWith(plan.Note, state.Note) {
		needsRefresh = true
		input := map[string]interface{}{
			"virtualHost": plan.ID.ValueString(),
			"note":        plan.Note.ValueString(),
		}
		if err := r.client.DoMutate(
			ctx,
			client.GQLRequest{Query: client.VMQuery, Variables: map[string]interface{}{"input": input}, Operation: "update"},
			nil,
			&client.DoOpts{Diags: &resp.Diagnostics},
		); err != nil {
			resp.Diagnostics.AddError("Client Error", err.Error())
			return
		}
	}

	if !needsRefresh {
		return
	}
	var res struct{ Data client.VMGQL }
	if err := r.client.Do(
		ctx,
		client.GQLRequest{
			Query:     client.VMQuery,
			Variables: map[string]interface{}{"id": plan.ID.ValueString()},
			Operation: "get",
		},
		&res,
		&client.DoOpts{Diags: &resp.Diagnostics},
	); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	resp.Diagnostics.Append(plan.intoModel(ctx, &res.Data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *vmResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data vmResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	deleteTimeout, _diags := data.Timeouts.Delete.ValueGoDuration()
	resp.Diagnostics.Append(_diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	var res struct {
		TaskExecution struct {
			ID              string
			VirtualHostList client.ConnectionNodeGQL
		}
	}
	err := r.client.DoMutate(
		ctx,
		client.GQLRequest{
			Query:     client.VMQuery,
			Variables: map[string]interface{}{"id": data.ID.ValueString()},
			Operation: "delete",
		},
		&res,
		&client.DoOpts{Diags: &resp.Diagnostics},
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	if !data.AwaitDeletionTask.ValueBool() {
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Awaiting VM deletion task with timeout: %s", deleteTimeout.String()))
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()
	if err := r.client.AwaitTask(ctx, res.TaskExecution.ID, &client.DoOpts{Diags: &resp.Diagnostics}); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
	}
}
