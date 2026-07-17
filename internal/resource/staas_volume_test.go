// Copyright @ Vivicta. All Rights Reserved. 2026
// SPDX-License-Identifier: MPL-2.0

package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Vivicta-SC/terraform-provider-ocp/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func staasVolumeTestSchema(t *testing.T) rschema.Schema {
	t.Helper()
	r := &staasVolumeResource{}
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	require.False(t, resp.Diagnostics.HasError())
	return resp.Schema
}

func baseStaasVolumeModel(id string, sizeGB int32, protocol string) staasVolumeResourceModel {
	return staasVolumeResourceModel{
		ID:                     types.StringValue(id),
		Name:                   types.StringValue("my-volume"),
		ProjectID:              types.StringValue("proj-1"),
		DataProtectionPolicyID: types.StringValue("dpp-1"),
		TierID:                 types.StringValue("tier-1"),
		VserverID:              types.StringValue("vserver-1"),
		Protocol:               types.StringValue(protocol),
		SizeGB:                 types.Int32Value(sizeGB),
		Note:                   types.StringValue(""),
		NfsExports:             types.SetNull(nfsExportObjectType),

		DeletionPrimaryRetentionDays: types.Int32Value(0),
		Timeouts: timeoutsModel{
			Create: timetypes.NewGoDurationValueFromStringMust("20m"),
			Read:   timetypes.NewGoDurationValueFromStringMust("15s"),
			Update: timetypes.NewGoDurationValueFromStringMust("20m"),
			Delete: timetypes.NewGoDurationValueFromStringMust("20m"),
		},
	}
}

// Regression test: the required `size_gb` attribute must be reconstructed from
// the API response, otherwise `terraform import` (and every subsequent Read)
// leaves it null.
func TestStaasVolumeFromGQL_PopulatesSizeGB(t *testing.T) {
	data := &client.StaasVolumeGQL{
		NodeGQL:  client.NodeGQL{ID: "vol-1"},
		Name:     "my-volume",
		Note:     "",
		Protocol: "NFS",
		SizeGB:   500,
	}
	data.Project.ID = "proj-1"
	data.DataProtectionPolicy.ID = "dpp-1"
	data.Tier.ID = "tier-1"
	data.Vserver.ID = "vserver-1"

	var model staasVolumeResourceModel
	diags := model.fromGQL(context.Background(), data)

	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, int32(500), model.SizeGB.ValueInt32())
}

// Regression test for the import path: right after
// `resource.ImportStatePassthroughID`, only `id` is known in state and
// `size_gb` is null. Read must fill it in from the API, otherwise the
// first plan after import always shows a diff / forces replacement.
func TestStaasVolumeRead_RestoresSizeGBAfterImport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			OperationName string `json:"operationName"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "get", body.OperationName)

		// The query aliases its result field as `data: volume(...)`, so the
		// GQL result sits one level under the top-level GraphQL "data" key.
		fmt.Fprint(w, `{"data":{"data":{
			"__typename":"VolumeNode","id":"vol-1","name":"my-volume","note":"","protocol":"NFS","sizeGB":250,
			"project":{"id":"proj-1"},"dataProtectionPolicy":{"id":"dpp-1"},"tier":{"id":"tier-1"},"vserver":{"id":"vserver-1"},
			"visibility":[]
		}}}`)
	}))
	defer server.Close()

	r := &staasVolumeResource{client: client.New(server.URL, "token", true, false)}
	sch := staasVolumeTestSchema(t)
	ctx := context.Background()

	importedModel := baseStaasVolumeModel("vol-1", 0, "NFS")
	importedModel.SizeGB = types.Int32Null()
	importedModel.Name = types.StringNull()
	importedModel.ProjectID = types.StringNull()
	importedModel.DataProtectionPolicyID = types.StringNull()
	importedModel.TierID = types.StringNull()
	importedModel.VserverID = types.StringNull()
	importedModel.Protocol = types.StringNull()

	state := tfsdk.State{Schema: sch}
	require.False(t, state.Set(ctx, &importedModel).HasError())

	resp := &resource.ReadResponse{State: tfsdk.State{Schema: sch}}
	r.Read(ctx, resource.ReadRequest{State: state}, resp)
	require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

	var out staasVolumeResourceModel
	require.False(t, resp.State.Get(ctx, &out).HasError())
	assert.Equal(t, int32(250), out.SizeGB.ValueInt32())
}

// Regression test for the schema: since protocol-specific resize mutations
// (resizeISCSI/resizeNAS) exist and are used from Update, size_gb must not
// force replacement - otherwise Terraform never reaches that code path.
func TestStaasVolumeSchema_SizeGBAllowsInPlaceResize(t *testing.T) {
	sch := staasVolumeTestSchema(t)

	attr, ok := sch.Attributes["size_gb"].(rschema.Int32Attribute)
	require.True(t, ok, "size_gb must be an Int32Attribute")
	assert.True(t, attr.Required)
	assert.Empty(t, attr.PlanModifiers, "size_gb must not RequiresReplace now that in-place resize is supported")
}

// mockStaasResizeServer answers the "get" query and the given resize
// operation, and always resolves the task-await polling query immediately.
func mockStaasResizeServer(t *testing.T, resizeOp string, wantSizeGB int32, protocol string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Query         string                 `json:"query"`
			Variables     map[string]interface{} `json:"variables"`
			OperationName string                 `json:"operationName"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))

		switch {
		case strings.Contains(body.Query, "taskExecution(id:"):
			fmt.Fprint(w, `{"data":{"taskExecution":{"id":"task-1","state":"SUCCESS"}}}`)

		case body.OperationName == resizeOp:
			input, _ := body.Variables["input"].(map[string]interface{})
			assert.Equal(t, "vol-1", input["volume"])
			assert.Equal(t, float64(wantSizeGB), input["sizeGB"])
			// Mutation result is also aliased as `data: volumeResize...(...)`.
			fmt.Fprint(w, `{"data":{"data":{"__typename":"TaskExecutionNode","id":"task-1"}}}`)

		case body.OperationName == "get":
			fmt.Fprintf(w, `{"data":{"data":{
				"__typename":"VolumeNode","id":"vol-1","name":"my-volume","note":"","protocol":%q,"sizeGB":%d,
				"project":{"id":"proj-1"},"dataProtectionPolicy":{"id":"dpp-1"},"tier":{"id":"tier-1"},"vserver":{"id":"vserver-1"},
				"visibility":[]
			}}}`, protocol, wantSizeGB)

		default:
			t.Fatalf("unexpected operation %q", body.OperationName)
		}
	}))
}

// Regression / apply-path test: growing size_gb in place must call the
// protocol-specific resize mutation (not replacement) and land the new size
// in state.
func TestStaasVolumeUpdate_ResizesInPlace(t *testing.T) {
	tests := []struct {
		protocol string
		wantOp   string
	}{
		{protocol: "NFS", wantOp: "resizeNAS"},
		{protocol: "ISCSI", wantOp: "resizeISCSI"},
	}

	for _, tc := range tests {
		t.Run(tc.protocol, func(t *testing.T) {
			server := mockStaasResizeServer(t, tc.wantOp, 200, tc.protocol)
			defer server.Close()

			r := &staasVolumeResource{client: client.New(server.URL, "token", true, false)}
			sch := staasVolumeTestSchema(t)
			ctx := context.Background()

			state := tfsdk.State{Schema: sch}
			stateModel := baseStaasVolumeModel("vol-1", 100, tc.protocol)
			require.False(t, state.Set(ctx, &stateModel).HasError())

			plan := tfsdk.Plan{Schema: sch}
			planModel := baseStaasVolumeModel("vol-1", 200, tc.protocol)
			require.False(t, plan.Set(ctx, &planModel).HasError())

			resp := &resource.UpdateResponse{State: tfsdk.State{Schema: sch}}
			r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, resp)
			require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

			var out staasVolumeResourceModel
			require.False(t, resp.State.Get(ctx, &out).HasError())
			assert.Equal(t, int32(200), out.SizeGB.ValueInt32())
		})
	}
}
