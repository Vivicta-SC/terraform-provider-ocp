// Copyright @ Vivicta. All Rights Reserved. 2026
// SPDX-License-Identifier: MPL-2.0

package resource

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func timeoutAttribute(_ context.Context, create string, read string, update string, delete string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional: true,
		Computed: true,
		Default: objectdefault.StaticValue(types.ObjectValueMust(map[string]attr.Type{
			"create": timetypes.GoDurationType{},
			"read":   timetypes.GoDurationType{},
			"update": timetypes.GoDurationType{},
			"delete": timetypes.GoDurationType{},
		}, map[string]attr.Value{
			"create": timetypes.NewGoDurationValueFromStringMust(create),
			"read":   timetypes.NewGoDurationValueFromStringMust(read),
			"update": timetypes.NewGoDurationValueFromStringMust(update),
			"delete": timetypes.NewGoDurationValueFromStringMust(delete),
		})),
		Attributes: map[string]schema.Attribute{
			"create": schema.StringAttribute{
				CustomType: timetypes.GoDurationType{},
				Optional:   true,
				Computed:   true,
				Default:    stringdefault.StaticString(create),
			},
			"read": schema.StringAttribute{
				CustomType: timetypes.GoDurationType{},
				Optional:   true,
				Computed:   true,
				Default:    stringdefault.StaticString(read),
			},
			"update": schema.StringAttribute{
				CustomType: timetypes.GoDurationType{},
				Optional:   true,
				Computed:   true,
				Default:    stringdefault.StaticString(update),
			},
			"delete": schema.StringAttribute{
				CustomType: timetypes.GoDurationType{},
				Optional:   true,
				Computed:   true,
				Default:    stringdefault.StaticString(delete),
			},
		},
	}
}

type timeoutsModel struct {
	Create timetypes.GoDuration `tfsdk:"create"`
	Read   timetypes.GoDuration `tfsdk:"read"`
	Update timetypes.GoDuration `tfsdk:"update"`
	Delete timetypes.GoDuration `tfsdk:"delete"`
}
