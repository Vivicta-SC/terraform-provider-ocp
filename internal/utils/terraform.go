// Copyright @ Vivicta. All Rights Reserved. 2026
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"context"
	"fmt"

	"github.com/Vivicta-SC/terraform-provider-ocp/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func IsKnown[T attr.Value](plan T) bool {
	return !plan.IsUnknown() && !plan.IsNull()
}

func HasChangedWith[T attr.Value](plan T, state T) bool {
	return !plan.Equal(state) && IsKnown(plan)
}

func GetTFSetFromGQLConnection(ctx context.Context, connection client.ConnectionNodeGQL, diag *diag.Diagnostics) types.Set {
	ids := connection.GetIDs()
	if len(ids) == 100 {
		diag.AddError("client error", "number of IDs possibly more then 100 & pagination is not yet implemented")
	}
	ret, diag_ := types.SetValueFrom(ctx, types.StringType, ids)
	if diag_.HasError() {
		diag.Append(diag_...)
	}
	return ret
}

func FromTFStringSetToGo(ctx context.Context, attr types.Set, diags *diag.Diagnostics) []string {
	ids := make([]string, 0, len(attr.Elements()))
	if diags_ := attr.ElementsAs(ctx, &ids, false); diags_.HasError() {
		diags.Append(diags_...)
	}
	return ids
}

func StringRequiredUnlessKnown(expressions ...path.Expression) validator.String {
	return requiredUnlessKnown{PathExpressions: expressions}
}

var _ validator.String = requiredUnlessKnown{}

type requiredUnlessKnown struct {
	PathExpressions path.Expressions
}

func (v requiredUnlessKnown) Description(ctx context.Context) string {
	return v.MarkdownDescription(ctx)
}
func (v requiredUnlessKnown) MarkdownDescription(_ context.Context) string {
	return fmt.Sprintf("Attribute is required due to any of the following being unknown: %q", v.PathExpressions)
}
func (v requiredUnlessKnown) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}

	expressions := req.PathExpression.MergeExpressions(v.PathExpressions...)
	for _, expression := range expressions {
		matchedPaths, diags := req.Config.PathMatches(ctx, expression)
		resp.Diagnostics.Append(diags...)
		if diags.HasError() {
			continue
		}
		for _, mp := range matchedPaths {
			if mp.Equal(req.Path) {
				continue
			}

			var mpVal attr.Value
			resp.Diagnostics.Append(req.Config.GetAttribute(ctx, mp, &mpVal)...)
			if diags.HasError() {
				continue
			}
			if !IsKnown(mpVal) {
				resp.Diagnostics.Append(validatordiag.InvalidAttributeCombinationDiagnostic(
					req.Path,
					fmt.Sprintf("Attribute %q must be specified when %q is not", req.Path, mp),
				))
			}
		}
	}
}
