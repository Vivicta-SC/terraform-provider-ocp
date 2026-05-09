// Copyright @ Vivicta. All Rights Reserved. 2026
// SPDX-License-Identifier: MPL-2.0

package resource

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func vmConfigAttribute(_ context.Context) schema.SingleNestedAttribute {
	defaultTimeouts := map[string]attr.Value{
		"create": timetypes.NewGoDurationValueFromStringMust("20m"),
		"read":   timetypes.NewGoDurationValueFromStringMust("15s"),
		"update": timetypes.NewGoDurationValueFromStringMust("20m"),
		"delete": timetypes.NewGoDurationValueFromStringMust("20m"),
	}
	return schema.SingleNestedAttribute{
		Optional: true,
		Computed: true,
		Default: objectdefault.StaticValue(types.ObjectValueMust(vmConfigType.AttrTypes, map[string]attr.Value{
			"allow_restart": types.BoolValue(false),
			// "await_creation_task": types.BoolValue(true),
			"await_deletion_task": types.BoolValue(true),
			"timeouts":            types.ObjectValueMust(vmTimeoutType.AttrTypes, defaultTimeouts),
		})),
		Attributes: map[string]schema.Attribute{
			// "await_creation_task": schema.BoolAttribute{
			// 	Optional:      true,
			// 	Computed:      true,
			// 	Default:       booldefault.StaticBool(true),
			// 	Description:   "VM will be considered created immediatelly without verifying the success of deletion task. State will not be corrent and will have to be updated manually",
			// 	PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			// },
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
			"timeouts": schema.SingleNestedAttribute{
				Optional: true,
				Computed: true,
				Default:  objectdefault.StaticValue(types.ObjectValueMust(vmTimeoutType.AttrTypes, defaultTimeouts)),
				Attributes: map[string]schema.Attribute{
					"create": schema.StringAttribute{
						CustomType: timetypes.GoDurationType{},
						Optional:   true,
						Computed:   true,
						Default:    stringdefault.StaticString("20m"),
					},
					"read": schema.StringAttribute{
						CustomType: timetypes.GoDurationType{},
						Optional:   true,
						Computed:   true,
						Default:    stringdefault.StaticString("15s"),
					},
					"update": schema.StringAttribute{
						CustomType: timetypes.GoDurationType{},
						Optional:   true,
						Computed:   true,
						Default:    stringdefault.StaticString("20m"),
					},
					"delete": schema.StringAttribute{
						CustomType: timetypes.GoDurationType{},
						Optional:   true,
						Computed:   true,
						Default:    stringdefault.StaticString("20m"),
					},
				},
			},
		},
	}
}

var vmTimeoutType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"create": timetypes.GoDurationType{},
		"read":   timetypes.GoDurationType{},
		"update": timetypes.GoDurationType{},
		"delete": timetypes.GoDurationType{},
	},
}

var vmConfigType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"allow_restart": types.BoolType,
		// "await_creation_task": types.BoolType,
		"await_deletion_task": types.BoolType,
		"timeouts":            vmTimeoutType,
	},
}

type vmTimeoutModel struct {
	Create timetypes.GoDuration `tfsdk:"create"`
	Read   timetypes.GoDuration `tfsdk:"read"`
	Update timetypes.GoDuration `tfsdk:"update"`
	Delete timetypes.GoDuration `tfsdk:"delete"`
}

type vmConfigModel struct {
	AllowRestart types.Bool `tfsdk:"allow_restart"`
	// AwaitCreationTask types.Bool     `tfsdk:"await_creation_task"`
	AwaitDeletionTask types.Bool     `tfsdk:"await_deletion_task"`
	Timeouts          vmTimeoutModel `tfsdk:"timeouts"`
}
