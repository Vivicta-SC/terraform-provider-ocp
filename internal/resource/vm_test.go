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
)

func TestVMCreateWithNICIPv4(t *testing.T) {
	var capturedInput map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var gqlReq client.GQLRequest
		if err := json.NewDecoder(r.Body).Decode(&gqlReq); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if gqlReq.Operation == "create" {
			capturedInput = gqlReq.Variables["input"].(map[string]interface{})
			
			// Respond with a mocked created VirtualHost
			resp := map[string]interface{}{
				"data": map[string]interface{}{
					"data": map[string]interface{}{
						"__typename": "VirtualHostCreated",
						"virtualHost": map[string]interface{}{
							"id": "vm-123",
							"name": "test-vm",
							"customer": map[string]interface{}{"id": "cust-123"},
							"dataProtectionPolicy": map[string]interface{}{"id": "dpp-123"},
							"domain": map[string]interface{}{"id": "dom-123"},
							"project": map[string]interface{}{"id": "proj-123"},
							"template": map[string]interface{}{"id": "temp-123"},
							"tier": map[string]interface{}{"id": "tier-123"},
							"tags": map[string]interface{}{"getNodes": []interface{}{}},
							"networkInterfaceList": []interface{}{
								map[string]interface{}{
									"id": "nic-123",
									"label": "nic-0",
									"network": map[string]interface{}{"id": "net-123"},
									"ipAddressList": []interface{}{
										map[string]interface{}{
											"id": "ip-123",
											"ipAddress": "192.168.1.50",
										},
									},
								},
							},
							"ipAddressList": map[string]interface{}{
								"getNodes": []interface{}{
									map[string]interface{}{
										"id": "ip-123",
										"ipAddress": "192.168.1.50",
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
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Await task query response
		if gqlReq.Variables["id"] != nil {
			resp := map[string]interface{}{
				"data": map[string]interface{}{
					"taskExecution": map[string]interface{}{
						"id": "task-123",
						"state": "SUCCESS",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Read VM query response
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"data": map[string]interface{}{
					"id": "vm-123",
					"name": "test-vm",
					"customer": map[string]interface{}{"id": "cust-123"},
					"dataProtectionPolicy": map[string]interface{}{"id": "dpp-123"},
					"domain": map[string]interface{}{"id": "dom-123"},
					"project": map[string]interface{}{"id": "proj-123"},
					"template": map[string]interface{}{"id": "temp-123"},
					"tier": map[string]interface{}{"id": "tier-123"},
					"tags": map[string]interface{}{"getNodes": []interface{}{}},
					"networkInterfaceList": []interface{}{
						map[string]interface{}{
							"id": "nic-123",
							"label": "nic-0",
							"network": map[string]interface{}{"id": "net-123"},
							"ipAddressList": []interface{}{
								map[string]interface{}{
									"id": "ip-123",
									"ipAddress": "192.168.1.50",
								},
							},
						},
					},
					"ipAddressList": map[string]interface{}{
						"getNodes": []interface{}{
							map[string]interface{}{
								"id": "ip-123",
								"ipAddress": "192.168.1.50",
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
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	ocpClient := client.New(server.URL, "dummy-token", false, false)
	r := &vmResource{client: ocpClient}

	ctx := context.Background()

	// Prepare mock schema/plan
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	schemaObj := schemaResp.Schema
	
	ipv4Val, _ := types.ObjectValue(ipv4ObjectType.AttrTypes, map[string]attr.Value{
		"id": types.StringNull(),
		"ip": types.StringValue("192.168.1.50"),
	})
	ipv4List, _ := types.ListValue(ipv4ObjectType, []attr.Value{ipv4Val})

	nicVal, _ := types.ObjectValue(nicObjectType.AttrTypes, map[string]attr.Value{
		"id":                     types.StringNull(),
		"network_id":             types.StringValue("net-123"),
		"default_gateway_ip":     types.StringNull(),
		"label":                  types.StringNull(),
		"ipv4":                   ipv4List,
		"ipv6":                   types.SetNull(types.StringType),
		"auto_assign_ip":         types.BoolValue(false),
		"use_as_default_gateway": types.BoolValue(true),
	})
	nicsList, _ := types.ListValue(nicObjectType, []attr.Value{nicVal})

	model := vmResourceModel{
		CustomerID:           types.StringValue("cust-123"),
		DataProtectionPolicyID: types.StringValue("dpp-123"),
		DomainID:             types.StringValue("dom-123"),
		ProjectID:            types.StringValue("proj-123"),
		TemplateID:           types.StringValue("temp-123"),
		TierID:               types.StringValue("tier-123"),
		Hostname:             types.StringValue("test-vm"),
		Antivirus:            types.StringValue("NONE"),
		ClusterType:          types.StringValue("STANDARD"),
		CoresPerSocket:       types.Int32Value(1),
		CpuCount:             types.Int32Value(1),
		MemorySizeGB:         types.Int32Value(4),
		NICS:                 nicsList,
		Disks:                types.ListNull(diskObjectType),
		Tags:                 types.SetNull(types.StringType),
		Timeouts: timeoutsModel{
			Create: timetypes.NewGoDurationValue(30 * time.Minute),
			Read:   timetypes.NewGoDurationValue(30 * time.Minute),
			Update: timetypes.NewGoDurationValue(30 * time.Minute),
			Delete: timetypes.NewGoDurationValue(30 * time.Minute),
		},
	}

	plan := tfsdk.Plan{
		Schema: schemaObj,
	}
	// Note: We bypass full plan.Get/Set diagnostics if we can build the model directly and call resource logic.
	// Since Create takes resource.CreateRequest which embeds the plan, we simulate this.
	// However, we want to see if the interfaceList[].ipList gets populated correctly.

	req := resource.CreateRequest{
		Plan: plan,
	}
	_ = req.Plan.Set(ctx, &model)

	resp := &resource.CreateResponse{
		State: tfsdk.State{
			Schema: schemaObj,
		},
	}
	r.Create(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("VM create resource diagnostics returned errors: %v", resp.Diagnostics)
	}

	if capturedInput == nil {
		t.Fatalf("Expected create request to be sent to GQL server, but got none")
	}

	interfaceList, ok := capturedInput["interfaceList"].([]interface{})
	if !ok || len(interfaceList) == 0 {
		t.Fatalf("Expected interfaceList in GQL request, got: %v", capturedInput["interfaceList"])
	}

	nic0 := interfaceList[0].(map[string]interface{})
	ipList, ok := nic0["ipList"].([]string)
	if !ok {
		// Sometimes Go decodes json as []interface{} but here it is constructed in VM creation as []string
		ipListGen, ok2 := nic0["ipList"].([]interface{})
		if ok2 {
			for _, ip := range ipListGen {
				ipList = append(ipList, ip.(string))
			}
		} else {
			t.Fatalf("Expected ipList to be a slice, got: %T", nic0["ipList"])
		}
	}

	if len(ipList) != 1 || ipList[0] != "192.168.1.50" {
		t.Errorf("Expected IP list to contain '192.168.1.50', got: %v", ipList)
	}
}
