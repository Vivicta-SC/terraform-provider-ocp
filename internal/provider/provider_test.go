// Copyright @ Vivicta. All Rights Reserved. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/Vivicta-SC/terraform-provider-ocp/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
)

func TestProviderConfigure_TLSVerification(t *testing.T) {
	// OCP_TOKEN is required by Configure
	t.Setenv("OCP_TOKEN", "dummy-token")

	tests := []struct {
		name         string
		config       map[string]tftypes.Value
		envVerifySSL string
		expectVerify bool
		expectErr    bool
	}{
		{
			name: "Default is true when no config and no env is set",
			config: map[string]tftypes.Value{
				"endpoint":   tftypes.NewValue(tftypes.String, nil),
				"verify_ssl": tftypes.NewValue(tftypes.Bool, nil),
				"debug":      tftypes.NewValue(tftypes.Bool, nil),
			},
			envVerifySSL: "",
			expectVerify: true,
		},
		{
			name: "Env false overrides default to false (InsecureSkipVerify: true)",
			config: map[string]tftypes.Value{
				"endpoint":   tftypes.NewValue(tftypes.String, nil),
				"verify_ssl": tftypes.NewValue(tftypes.Bool, nil),
				"debug":      tftypes.NewValue(tftypes.Bool, nil),
			},
			envVerifySSL: "false",
			expectVerify: false,
		},
		{
			name: "Env true overrides default to true (InsecureSkipVerify: false)",
			config: map[string]tftypes.Value{
				"endpoint":   tftypes.NewValue(tftypes.String, nil),
				"verify_ssl": tftypes.NewValue(tftypes.Bool, nil),
				"debug":      tftypes.NewValue(tftypes.Bool, nil),
			},
			envVerifySSL: "true",
			expectVerify: true,
		},
		{
			name: "Config false overrides env true",
			config: map[string]tftypes.Value{
				"endpoint":   tftypes.NewValue(tftypes.String, nil),
				"verify_ssl": tftypes.NewValue(tftypes.Bool, false),
				"debug":      tftypes.NewValue(tftypes.Bool, nil),
			},
			envVerifySSL: "true",
			expectVerify: false,
		},
		{
			name: "Config true overrides env false",
			config: map[string]tftypes.Value{
				"endpoint":   tftypes.NewValue(tftypes.String, nil),
				"verify_ssl": tftypes.NewValue(tftypes.Bool, true),
				"debug":      tftypes.NewValue(tftypes.Bool, nil),
			},
			envVerifySSL: "false",
			expectVerify: true,
		},
		{
			name: "Invalid env value raises diagnostic error",
			config: map[string]tftypes.Value{
				"endpoint":   tftypes.NewValue(tftypes.String, nil),
				"verify_ssl": tftypes.NewValue(tftypes.Bool, nil),
				"debug":      tftypes.NewValue(tftypes.Bool, nil),
			},
			envVerifySSL: "invalid-boolean",
			expectErr:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("OCP_VERIFY_SSL", tc.envVerifySSL)

			p := &ocpProvider{version: "test"}
			var schemaResp provider.SchemaResponse
			p.Schema(context.Background(), provider.SchemaRequest{}, &schemaResp)

			var resp provider.ConfigureResponse

			// Create a proper ConfigureRequest with the test config
			req := provider.ConfigureRequest{
				Config: tfsdk.Config{
					Raw: tftypes.NewValue(
						tftypes.Object{
							AttributeTypes: map[string]tftypes.Type{
								"endpoint":   tftypes.String,
								"verify_ssl": tftypes.Bool,
								"debug":      tftypes.Bool,
							},
						},
						tc.config,
					),
					Schema: schemaResp.Schema,
				},
			}

			p.Configure(context.Background(), req, &resp)

			if tc.expectErr {
				assert.True(t, resp.Diagnostics.HasError(), "Expected configuration diagnostic error")
				assert.Nil(t, resp.DataSourceData, "Expected client setup to be aborted on diagnostic error")
			} else {
				assert.False(t, resp.Diagnostics.HasError(), "Unexpected diagnostic error: %v", resp.Diagnostics)
				assert.NotNil(t, resp.DataSourceData, "Expected client to be populated")

				cli, ok := resp.DataSourceData.(*client.OCPClient)
				if assert.True(t, ok, "Expected client of type *client.OCPClient") {
					assert.Equal(t, !tc.expectVerify, cli.InsecureSkipVerify(), "InsecureSkipVerify should match expected verify_ssl value")
				}
			}
		})
	}
}
