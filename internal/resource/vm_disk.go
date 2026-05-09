// Copyright @ Vivicta. All Rights Reserved. 2026
// SPDX-License-Identifier: MPL-2.0

package resource

import (
	"context"

	"github.com/Vivicta-SC/terraform-provider-ocp/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func disksAttribute() schema.ListNestedAttribute {
	// TODO: Make this into Map with Label key?
	return schema.ListNestedAttribute{
		Optional:      true,
		Computed:      true,
		Default:       listdefault.StaticValue(types.ListValueMust(diskObjectType, []attr.Value{})),
		PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()},
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{
					Computed:      true,
					PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				},
				"size_gb": schema.Int32Attribute{
					Required: true,
				},
				"allocation_unit_size": schema.Int32Attribute{
					Optional:      true,
					Computed:      true,
					Default:       int32default.StaticInt32(16384),
					PlanModifiers: []planmodifier.Int32{int32planmodifier.RequiresReplace()},
				},
				"win_disk_letter": schema.StringAttribute{
					Optional:      true,
					PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
					Validators:    []validator.String{stringvalidator.LengthAtMost(1)},
				},
			},
		},
	}
}

var diskObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"id":                   types.StringType,
		"size_gb":              types.Int32Type,
		"allocation_unit_size": types.Int32Type,
		"win_disk_letter":      types.StringType,
	},
}

type diskModel struct {
	ID                 types.String `tfsdk:"id"`
	SizeGB             types.Int32  `tfsdk:"size_gb"`
	AllocationUnitSize types.Int32  `tfsdk:"allocation_unit_size"`
	WinDiskLetter      types.String `tfsdk:"win_disk_letter"`
}

func (disk *diskModel) intoModel(_ context.Context, data *client.DiskGQL) diag.Diagnostics {
	var diags diag.Diagnostics

	disk.ID = types.StringValue(data.ID)
	disk.SizeGB = types.Int32Value(data.SizeGB)

	return diags
}

func (vm *vmResourceModel) fromDisksGQL(ctx context.Context, data []*client.DiskGQL) diag.Diagnostics {
	var diags diag.Diagnostics

	current_disks := make([]diskModel, 0, len(vm.Disks.Elements()))
	diags.Append(vm.Disks.ElementsAs(ctx, &current_disks, false)...)
	if diags.HasError() {
		return diags
	}

	disks := make([]diskModel, 0, len(data))
	matched := make(map[int]struct{}, len(current_disks))
	// TODO: this does not respect the orders now!
	for _, diskGQL := range data {
		// TODO: how to deal with C: disk?
		if diskGQL.Key == 2000 {
			continue
		}

		var disk *diskModel
		for idx, current_nic := range current_disks {
			// Attempting to match to current state, so that _wo arguments are preserved
			// TODO: we need something better here
			if _, ok := matched[idx]; ok {
				continue
			}
			if diskGQL.ID == current_nic.ID.ValueString() || current_nic.SizeGB.ValueInt32() == diskGQL.SizeGB {
				disk = &current_nic
				matched[idx] = struct{}{}
				break
			}
		}
		if disk == nil {
			disk = &diskModel{}
		}
		diags.Append(disk.intoModel(ctx, diskGQL)...)
		disks = append(disks, *disk)
	}
	diskList, diag := types.ListValueFrom(ctx, diskObjectType, disks)
	diags.Append(diag...)
	vm.Disks = diskList

	return diags
}
