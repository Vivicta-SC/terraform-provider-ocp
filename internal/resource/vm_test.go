// Copyright @ Vivicta. All Rights Reserved. 2026
// SPDX-License-Identifier: MPL-2.0

package resource

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Vivicta-SC/terraform-provider-ocp/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestVMCreate_SendsExplicitNICIPv4(t *testing.T) {
	var capturedInput map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var gqlReq client.GQLRequest
		if err := json.NewDecoder(r.Body).Decode(&gqlReq); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if gqlReq.Operation == "create" {
			capturedInput = gqlReq.Variables["input"].(map[string]interface{})
			resp := map[string]interface{}{
				"data": map[string]interface{}{
					"data": map[string]interface{}{
						"__typename": "VirtualHostCreated",
						"virtualHost": map[string]interface{}{
							"id":                   "vm-123",
							"name":                 "test-vm",
							"customer":             map[string]interface{}{"id": "cust-123"},
							"dataProtectionPolicy": map[string]interface{}{"id": "dpp-123"},
							"domain":               map[string]interface{}{"id": "dom-123"},
							"project":              map[string]interface{}{"id": "proj-123"},
							"template":             map[string]interface{}{"id": "temp-123"},
							"tier":                 map[string]interface{}{"id": "tier-123"},
							"tags":                 map[string]interface{}{"getNodes": []interface{}{}},
							"networkInterfaceList": []interface{}{
								map[string]interface{}{
									"id":            "nic-123",
									"label":         "nic-0",
									"network":       map[string]interface{}{"id": "net-123"},
									"ipv4Addresses": []interface{}{map[string]interface{}{"IP": "192.168.1.50"}},
									"ipv6Addresses": []interface{}{},
								},
							},
							"ipAddressList": map[string]interface{}{
								"getNodes": []interface{}{
									map[string]interface{}{
										"id":      "ip-123",
										"IP":      "192.168.1.50",
										"network": map[string]interface{}{"id": "net-123"},
									},
								},
							},
							"disks": map[string]interface{}{
								"getNodes": []interface{}{},
							},
						},
						"taskExecution": map[string]interface{}{
							"id": "task-123",
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if gqlReq.Variables["id"] != nil {
			resp := map[string]interface{}{
				"data": map[string]interface{}{
					"taskExecution": map[string]interface{}{
						"id":    "task-123",
						"state": "SUCCESS",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		// Read / Get response
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"data": map[string]interface{}{
					"id":                   "vm-123",
					"name":                 "test-vm",
					"customer":             map[string]interface{}{"id": "cust-123"},
					"dataProtectionPolicy": map[string]interface{}{"id": "dpp-123"},
					"domain":               map[string]interface{}{"id": "dom-123"},
					"project":              map[string]interface{}{"id": "proj-123"},
					"template":             map[string]interface{}{"id": "temp-123"},
					"tier":                 map[string]interface{}{"id": "tier-123"},
					"tags":                 map[string]interface{}{"getNodes": []interface{}{}},
					"networkInterfaceList": []interface{}{
						map[string]interface{}{
							"id":            "nic-123",
							"label":         "nic-0",
							"network":       map[string]interface{}{"id": "net-123"},
							"ipv4Addresses": []interface{}{map[string]interface{}{"IP": "192.168.1.50"}},
							"ipv6Addresses": []interface{}{},
						},
					},
					"ipAddressList": map[string]interface{}{
						"getNodes": []interface{}{
							map[string]interface{}{
								"id":      "ip-123",
								"IP":      "192.168.1.50",
								"network": map[string]interface{}{"id": "net-123"},
							},
						},
					},
					"disks": map[string]interface{}{
						"getNodes": []interface{}{},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	ocpClient := client.New(server.URL, "dummy-token", false, false)
	r := &vmResource{client: ocpClient}

	ctx := context.Background()

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	schemaObj := schemaResp.Schema

	ipv4Val, diags := types.ObjectValue(ipv4ObjectType.AttrTypes, map[string]attr.Value{
		"id": types.StringNull(),
		"ip": types.StringValue("192.168.1.50"),
	})
	require.False(t, diags.HasError(), diags)
	ipv4List, diags := types.ListValue(ipv4ObjectType, []attr.Value{ipv4Val})
	require.False(t, diags.HasError(), diags)

	nicVal, diags := types.ObjectValue(nicObjectType.AttrTypes, map[string]attr.Value{
		"id":                     types.StringNull(),
		"network_id":             types.StringValue("net-123"),
		"default_gateway_ip":     types.StringNull(),
		"label":                  types.StringNull(),
		"ipv4":                   ipv4List,
		"ipv6":                   types.SetNull(types.StringType),
		"auto_assign_ip":         types.BoolValue(false),
		"use_as_default_gateway": types.BoolValue(true),
	})
	require.False(t, diags.HasError(), diags)
	nicsList, diags := types.ListValue(nicObjectType, []attr.Value{nicVal})
	require.False(t, diags.HasError(), diags)

	model := vmResourceModel{
		CustomerID:             types.StringValue("cust-123"),
		DataProtectionPolicyID: types.StringValue("dpp-123"),
		DomainID:               types.StringValue("dom-123"),
		ProjectID:              types.StringValue("proj-123"),
		TemplateID:             types.StringValue("temp-123"),
		TierID:                 types.StringValue("tier-123"),
		Hostname:               types.StringValue("test-vm"),
		Antivirus:              types.StringValue("NONE"),
		ClusterType:            types.StringValue("STANDARD"),
		CoresPerSocket:         types.Int32Value(1),
		CpuCount:               types.Int32Value(1),
		MemorySizeGB:           types.Int32Value(4),
		NICS:                   nicsList,
		Disks:                  types.ListNull(diskObjectType),
		Tags:                   types.SetNull(types.StringType),
		Timeouts: timeoutsModel{
			Create: timetypes.NewGoDurationValue(30 * time.Minute),
			Read:   timetypes.NewGoDurationValue(30 * time.Minute),
			Update: timetypes.NewGoDurationValue(30 * time.Minute),
			Delete: timetypes.NewGoDurationValue(30 * time.Minute),
		},
	}

	plan := tfsdk.Plan{Schema: schemaObj}
	req := resource.CreateRequest{Plan: plan}
	_ = req.Plan.Set(ctx, &model)

	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaObj},
	}
	r.Create(ctx, req, resp)

	require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
	require.NotNil(t, capturedInput)

	interfaceList, ok := capturedInput["interfaceList"].([]interface{})
	require.True(t, ok)
	require.Len(t, interfaceList, 1)

	nic0 := interfaceList[0].(map[string]interface{})
	ipListVal, ok := nic0["ipList"]
	require.True(t, ok)

	ipListSlice, ok := ipListVal.([]interface{})
	if ok {
		require.Len(t, ipListSlice, 1)
		require.Equal(t, "192.168.1.50", ipListSlice[0].(string))
	} else {
		ipListStringSlice, ok := ipListVal.([]string)
		require.True(t, ok)
		require.Len(t, ipListStringSlice, 1)
		require.Equal(t, "192.168.1.50", ipListStringSlice[0])
	}
}

func TestVMUpdate_SendsCoresPerSocketOnResize(t *testing.T) {
	var capturedInput map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var gqlReq client.GQLRequest
		if err := json.NewDecoder(r.Body).Decode(&gqlReq); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if gqlReq.Operation == "resize" {
			capturedInput = gqlReq.Variables["input"].(map[string]interface{})
			resp := map[string]interface{}{
				"data": map[string]interface{}{
					"data": map[string]interface{}{
						"__typename": "TaskExecutionNode",
						"id":         "task-resize-123",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if gqlReq.Variables["id"] != nil {
			resp := map[string]interface{}{
				"data": map[string]interface{}{
					"taskExecution": map[string]interface{}{
						"id":    "task-resize-123",
						"state": "SUCCESS",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		// Read / Get response
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"data": map[string]interface{}{
					"id":                   "vm-123",
					"name":                 "test-vm",
					"customer":             map[string]interface{}{"id": "cust-123"},
					"dataProtectionPolicy": map[string]interface{}{"id": "dpp-123"},
					"domain":               map[string]interface{}{"id": "dom-123"},
					"project":              map[string]interface{}{"id": "proj-123"},
					"template":             map[string]interface{}{"id": "temp-123"},
					"tier":                 map[string]interface{}{"id": "tier-123"},
					"tags":                 map[string]interface{}{"getNodes": []interface{}{}},
					"networkInterfaceList": []interface{}{},
					"ipAddressList": map[string]interface{}{
						"getNodes": []interface{}{},
					},
					"disks": map[string]interface{}{
						"getNodes": []interface{}{},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	ocpClient := client.New(server.URL, "dummy-token", false, false)
	r := &vmResource{client: ocpClient}

	ctx := context.Background()

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	schemaObj := schemaResp.Schema

	// Initial State: 1 CPU, 1 Core per socket
	stateModel := vmResourceModel{
		ID:                     types.StringValue("vm-123"),
		CustomerID:             types.StringValue("cust-123"),
		DataProtectionPolicyID: types.StringValue("dpp-123"),
		DomainID:               types.StringValue("dom-123"),
		ProjectID:              types.StringValue("proj-123"),
		TemplateID:             types.StringValue("temp-123"),
		TierID:                 types.StringValue("tier-123"),
		Hostname:               types.StringValue("test-vm"),
		Antivirus:              types.StringValue("NONE"),
		ClusterType:            types.StringValue("STANDARD"),
		CoresPerSocket:         types.Int32Value(1),
		CpuCount:               types.Int32Value(1),
		MemorySizeGB:           types.Int32Value(4),
		NICS:                   types.ListNull(nicObjectType),
		Disks:                  types.ListNull(diskObjectType),
		Tags:                   types.SetNull(types.StringType),
		Timeouts: timeoutsModel{
			Create: timetypes.NewGoDurationValue(30 * time.Minute),
			Read:   timetypes.NewGoDurationValue(30 * time.Minute),
			Update: timetypes.NewGoDurationValue(30 * time.Minute),
			Delete: timetypes.NewGoDurationValue(30 * time.Minute),
		},
	}

	// Case 1: Simultaneous change (both CPU count and Cores per socket change)
	planModelSimultaneous := stateModel
	planModelSimultaneous.CpuCount = types.Int32Value(4)
	planModelSimultaneous.CoresPerSocket = types.Int32Value(2)

	plan := tfsdk.Plan{Schema: schemaObj}
	_ = plan.Set(ctx, &planModelSimultaneous)
	state := tfsdk.State{Schema: schemaObj}
	_ = state.Set(ctx, &stateModel)

	req := resource.UpdateRequest{
		Plan:  plan,
		State: state,
	}
	resp := &resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaObj},
	}

	r.Update(ctx, req, resp)

	require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
	require.NotNil(t, capturedInput)
	require.Equal(t, float64(4), capturedInput["cpuCount"])
	require.Equal(t, float64(2), capturedInput["coresPerSocket"])

	// Case 2: Only Cores per socket changes
	capturedInput = nil
	planModelCoresOnly := stateModel
	planModelCoresOnly.CoresPerSocket = types.Int32Value(2)

	_ = plan.Set(ctx, &planModelCoresOnly)
	req = resource.UpdateRequest{
		Plan:  plan,
		State: state,
	}
	resp = &resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaObj},
	}

	r.Update(ctx, req, resp)

	require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
	require.NotNil(t, capturedInput)
	require.Nil(t, capturedInput["cpuCount"])
	require.Equal(t, float64(2), capturedInput["coresPerSocket"])
}
