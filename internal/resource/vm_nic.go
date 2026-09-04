// Copyright @ Vivicta. All Rights Reserved. 2026
// SPDX-License-Identifier: MPL-2.0

package resource

import (
	"context"

	"github.com/Vivicta-SC/terraform-provider-ocp/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// These mirror the schema Defaults below. They're reused when reconstructing a
// NIC that has no matching prior state entry (e.g. on import), since neither
// attribute has an API equivalent to read back - both only describe how the NIC
// was configured at creation time (auto-assigning an IP, using it as the
// default gateway).
const (
	defaultAutoAssignIP        = true
	defaultUseAsDefaultGateway = false
)

func nicsAttribute() schema.ListNestedAttribute {
	// TODO: Make this into Map with Label key?
	return schema.ListNestedAttribute{
		Optional:      true,
		Computed:      true,
		Default:       listdefault.StaticValue(types.ListValueMust(nicObjectType, []attr.Value{})),
		PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()},
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{
					Computed:      true,
					PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				},
				"network_id": schema.StringAttribute{
					Required:      true,
					PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				},
				"label": schema.StringAttribute{
					Computed:      true,
					PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				},
				"default_gateway_ip": schema.StringAttribute{
					Computed:      true,
					PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				},
				"ipv4": schema.ListNestedAttribute{
					Optional:      true,
					Computed:      true,
					PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace(), listplanmodifier.UseStateForUnknown()},
					NestedObject: schema.NestedAttributeObject{
						Attributes: map[string]schema.Attribute{
							"id": schema.StringAttribute{
								Computed: true,
							},
							"ip": schema.StringAttribute{
								Computed: true,
								Optional: true,
							},
						},
					},
				},
				"ipv6": schema.SetAttribute{
					ElementType:   types.StringType,
					Computed:      true,
					PlanModifiers: []planmodifier.Set{setplanmodifier.RequiresReplace(), setplanmodifier.UseStateForUnknown()},
				},
				"auto_assign_ip": schema.BoolAttribute{
					Description:   "Not readable via the OCP API. Preserved as-is on refresh; seeded to its default on import.",
					Optional:      true,
					Computed:      true,
					Default:       booldefault.StaticBool(defaultAutoAssignIP),
					PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace(), boolplanmodifier.UseStateForUnknown()},
				},
				"use_as_default_gateway": schema.BoolAttribute{
					Description:   "Not readable via the OCP API. Preserved as-is on refresh; seeded to its default on import.",
					Optional:      true,
					Computed:      true,
					Default:       booldefault.StaticBool(defaultUseAsDefaultGateway),
					PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace(), boolplanmodifier.UseStateForUnknown()},
				},
			},
		},
	}
}

var ipv4ObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"id": types.StringType,
		"ip": types.StringType,
	},
}

var nicObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"id":                     types.StringType,
		"network_id":             types.StringType,
		"default_gateway_ip":     types.StringType,
		"label":                  types.StringType,
		"ipv4":                   types.ListType{ElemType: ipv4ObjectType},
		"ipv6":                   types.SetType{ElemType: types.StringType},
		"auto_assign_ip":         types.BoolType,
		"use_as_default_gateway": types.BoolType,
	},
}

type ipv4Model struct {
	ID types.String `tfsdk:"id"`
	IP types.String `tfsdk:"ip"`
}

type nicModel struct {
	ID                  types.String `tfsdk:"id"`
	NetworkID           types.String `tfsdk:"network_id"`
	DefaultGatewayIP    types.String `tfsdk:"default_gateway_ip"`
	Label               types.String `tfsdk:"label"`
	IPv4                types.List   `tfsdk:"ipv4"`
	IPv6                types.Set    `tfsdk:"ipv6"`
	AutoAssignIp        types.Bool   `tfsdk:"auto_assign_ip"`
	UseAsDefaultGateway types.Bool   `tfsdk:"use_as_default_gateway"`
}

func (nic *nicModel) intoModel(ctx context.Context, data *client.NICGQL, data_ips []*client.VMIPGQL) diag.Diagnostics {
	var diags diag.Diagnostics

	nic.ID = types.StringValue(data.ID)
	nic.NetworkID = types.StringValue(data.Network.ID)
	nic.Label = types.StringValue(data.Label)
	nic.DefaultGatewayIP = types.StringValue(data.DefaultGatewayIP)

	ipv4 := make([]ipv4Model, 0, len(data.IPv4Addresses))
	for _, ip := range data.IPv4Addresses {
		tfIP := ipv4Model{IP: types.StringValue(ip.IP)}
		for _, vm_ip_obj := range data_ips {
			if vm_ip_obj.IP == ip.IP && data.Network.ID == vm_ip_obj.Network.ID {
				tfIP.ID = types.StringValue(vm_ip_obj.ID)
				break
			}
		}
		ipv4 = append(ipv4, tfIP)
	}

	tfList, diags := types.ListValueFrom(ctx, ipv4ObjectType, ipv4)
	nic.IPv4 = tfList

	gqlIPv6 := make([]string, 0, len(data.IPv6Addresses))
	for _, ip := range data.IPv6Addresses {
		gqlIPv6 = append(gqlIPv6, ip.IP)
	}
	ipv6, diag := types.SetValueFrom(ctx, types.StringType, gqlIPv6)
	diags.Append(diag...)
	nic.IPv6 = ipv6

	return diags
}

func (vm *vmResourceModel) fromNICsGQL(ctx context.Context, data []client.NICGQL, data_ips []*client.VMIPGQL) diag.Diagnostics {
	var diags diag.Diagnostics

	current_nics := make([]nicModel, 0, len(vm.NICS.Elements()))
	diags.Append(vm.NICS.ElementsAs(ctx, &current_nics, false)...)
	if diags.HasError() {
		return diags
	}

	nics := make([]nicModel, 0, len(data))
	// TODO: this does not respect the orders now!
	for _, nicGQL := range data {
		var nic *nicModel
		for _, current_nic := range current_nics {
			// Attempting to match to current state, so that _wo arguments are preserved
			if nicGQL.ID == current_nic.ID.ValueString() || current_nic.NetworkID.ValueString() == nicGQL.Network.ID {
				nic = &current_nic
				break
			}
		}
		if nic == nil {
			// No prior state to preserve _wo arguments from (e.g. this is an
			// import). Neither field is readable from the API, so fall back to
			// the schema defaults rather than leaving them null - that keeps a
			// plan matching the common/default config a no-op instead of an
			// unconditional forced replace.
			nic = &nicModel{
				AutoAssignIp:        types.BoolValue(defaultAutoAssignIP),
				UseAsDefaultGateway: types.BoolValue(defaultUseAsDefaultGateway),
			}
		}
		diags.Append(nic.intoModel(ctx, &nicGQL, data_ips)...)
		nics = append(nics, *nic)
	}
	nicsList, diag := types.ListValueFrom(ctx, nicObjectType, nics)
	diags.Append(diag...)
	vm.NICS = nicsList
	return diags
}
