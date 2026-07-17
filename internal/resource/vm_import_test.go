// Copyright @ Vivicta. All Rights Reserved. 2026
// SPDX-License-Identifier: MPL-2.0

package resource

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Vivicta-SC/terraform-provider-ocp/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"
)

func mustVMGQL(t *testing.T, raw string) *client.VMGQL {
	t.Helper()
	var data client.VMGQL
	require.NoError(t, json.Unmarshal([]byte(raw), &data))
	return &data
}

func emptyList(t *testing.T, objType attr.Type) types.List {
	t.Helper()
	list, diags := types.ListValue(objType, []attr.Value{})
	require.False(t, diags.HasError(), diags)
	return list
}

// A full VM as returned by the GraphQL API: one OS disk (key 2000), one data
// disk, and one NIC. Used to simulate "get" responses across all tests below.
const vmGQLFixture = `{
	"id": "vh-1",
	"customer": {"id": "cust-1"},
	"domain": {"id": "dom-1"},
	"project": {"id": "proj-1"},
	"template": {"id": "tmpl-1"},
	"dataProtectionPolicy": {"id": "dpp-1"},
	"tier": {"id": "tier-1"},
	"hostname": "host1",
	"note": "note1",
	"region": "SWEDEN",
	"cpuCount": 2,
	"coresPerSocket": 1,
	"memorySizeGB": 4,
	"clusterType": "PRIMARY",
	"antivirusType": "NONE",
	"localDiskList": {"edges": [
		{"node": {"id": "disk-os", "key": 2000, "sizeGB": 80}},
		{"node": {"id": "disk-1", "key": 1, "sizeGB": 50}}
	]},
	"networkInterfaceList": [
		{"id": "nic-1", "label": "eth0", "defaultGwIp": "10.0.0.1", "network": {"id": "net-1"},
		 "ipv4Addresses": [{"IP": "10.0.0.5"}], "ipv6Addresses": []}
	],
	"tagList": {"edges": []},
	"ipAddressList": {"edges": [
		{"node": {"id": "ip-1", "IP": "10.0.0.5", "network": {"id": "net-1"}}}
	]}
}`

func TestIntoModelWithoutCreationUnknown_RecoversOSDiskSize(t *testing.T) {
	data := mustVMGQL(t, vmGQLFixture)

	var vm vmResourceModel
	diags := vm.intoModelWithoutCreationUnknown(context.Background(), data)

	require.False(t, diags.HasError(), diags)
	require.Equal(t, types.Int32Value(80), vm.OSDiskSizeGB)
}

func TestIntoModelWithoutCreationUnknown_PreservesPriorOSDiskSizeWhenAbsent(t *testing.T) {
	data := mustVMGQL(t, `{"id": "vh-1", "localDiskList": {"edges": [
		{"node": {"id": "disk-1", "key": 1, "sizeGB": 50}}
	]}}`)

	vm := vmResourceModel{OSDiskSizeGB: types.Int32Value(77)}
	diags := vm.intoModelWithoutCreationUnknown(context.Background(), data)

	require.False(t, diags.HasError(), diags)
	require.Equal(t, types.Int32Value(77), vm.OSDiskSizeGB, "os_disk_size_gb must be preserved when the API doesn't expose it")
}

func TestFromDisksGQL_ImportSeedsDefaultAllocationUnitSize(t *testing.T) {
	data := mustVMGQL(t, vmGQLFixture)

	// Simulate the state right after import: no prior disks to preserve.
	vm := vmResourceModel{Disks: emptyList(t, diskObjectType)}
	diags := vm.fromDisksGQL(context.Background(), data.Disks.GetNodes())
	require.False(t, diags.HasError(), diags)

	var disks []diskModel
	require.False(t, vm.Disks.ElementsAs(context.Background(), &disks, false).HasError())
	require.Len(t, disks, 1, "the OS disk must be excluded from `disks`")

	require.Equal(t, "disk-1", disks[0].ID.ValueString())
	require.Equal(t, int32(50), disks[0].SizeGB.ValueInt32())
	require.Equal(t, types.Int32Value(defaultAllocationUnitSize), disks[0].AllocationUnitSize,
		"allocation_unit_size has no API equivalent; it must fall back to the schema default rather than null")
	require.True(t, disks[0].WinDiskLetter.IsNull(), "win_disk_letter has no default and must stay unset")
}

func TestFromDisksGQL_PreservesMatchedDiskExtras(t *testing.T) {
	data := mustVMGQL(t, vmGQLFixture)

	current, diags := types.ListValueFrom(context.Background(), diskObjectType, []diskModel{
		{
			ID:                 types.StringValue("stale-id"), // matched by size, not ID
			SizeGB:             types.Int32Value(50),
			AllocationUnitSize: types.Int32Value(32768),
			WinDiskLetter:      types.StringValue("D"),
		},
	})
	require.False(t, diags.HasError(), diags)

	vm := vmResourceModel{Disks: current}
	diags = vm.fromDisksGQL(context.Background(), data.Disks.GetNodes())
	require.False(t, diags.HasError(), diags)

	var disks []diskModel
	require.False(t, vm.Disks.ElementsAs(context.Background(), &disks, false).HasError())
	require.Len(t, disks, 1)

	require.Equal(t, "disk-1", disks[0].ID.ValueString(), "id must be refreshed from the API")
	require.Equal(t, types.Int32Value(32768), disks[0].AllocationUnitSize, "prior allocation_unit_size must be preserved")
	require.Equal(t, types.StringValue("D"), disks[0].WinDiskLetter, "prior win_disk_letter must be preserved")
}

func TestFromNICsGQL_ImportSeedsDefaultCreationFlags(t *testing.T) {
	data := mustVMGQL(t, vmGQLFixture)

	// Simulate the state right after import: no prior NICs to preserve.
	vm := vmResourceModel{NICS: emptyList(t, nicObjectType)}
	diags := vm.fromNICsGQL(context.Background(), data.NetworkInterfaceList, data.IpAddressList.GetNodes())
	require.False(t, diags.HasError(), diags)

	var nics []nicModel
	require.False(t, vm.NICS.ElementsAs(context.Background(), &nics, false).HasError())
	require.Len(t, nics, 1)

	require.Equal(t, "net-1", nics[0].NetworkID.ValueString())
	require.Equal(t, types.BoolValue(defaultAutoAssignIP), nics[0].AutoAssignIp,
		"auto_assign_ip has no API equivalent; it must fall back to the schema default rather than null")
	require.Equal(t, types.BoolValue(defaultUseAsDefaultGateway), nics[0].UseAsDefaultGateway,
		"use_as_default_gateway has no API equivalent; it must fall back to the schema default rather than null")
}

func TestFromNICsGQL_PreservesMatchedNICExtras(t *testing.T) {
	data := mustVMGQL(t, vmGQLFixture)

	current, diags := types.ListValueFrom(context.Background(), nicObjectType, []nicModel{
		{
			ID:                  types.StringValue("stale-id"), // matched by network_id, not ID
			NetworkID:           types.StringValue("net-1"),
			Label:               types.StringValue("stale-label"),
			DefaultGatewayIP:    types.StringValue("0.0.0.0"),
			IPv4:                emptyList(t, ipv4ObjectType),
			IPv6:                types.SetValueMust(types.StringType, []attr.Value{}),
			AutoAssignIp:        types.BoolValue(false),
			UseAsDefaultGateway: types.BoolValue(true),
		},
	})
	require.False(t, diags.HasError(), diags)

	vm := vmResourceModel{NICS: current}
	diags = vm.fromNICsGQL(context.Background(), data.NetworkInterfaceList, data.IpAddressList.GetNodes())
	require.False(t, diags.HasError(), diags)

	var nics []nicModel
	require.False(t, vm.NICS.ElementsAs(context.Background(), &nics, false).HasError())
	require.Len(t, nics, 1)

	require.Equal(t, "nic-1", nics[0].ID.ValueString(), "id must be refreshed from the API")
	require.Equal(t, "eth0", nics[0].Label.ValueString(), "label must be refreshed from the API")
	require.Equal(t, types.BoolValue(false), nics[0].AutoAssignIp, "prior auto_assign_ip must be preserved")
	require.Equal(t, types.BoolValue(true), nics[0].UseAsDefaultGateway, "prior use_as_default_gateway must be preserved")
}

func TestImportState_SeedsUnrecoverableDefaults(t *testing.T) {
	ctx := context.Background()
	r := &vmResource{}

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError(), schemaResp.Diagnostics)

	state := tfsdk.State{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
	}

	req := resource.ImportStateRequest{ID: "vh-123"}
	resp := &resource.ImportStateResponse{State: state}
	r.ImportState(ctx, req, resp)

	require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)

	var id types.String
	require.False(t, resp.State.GetAttribute(ctx, path.Root("id"), &id).HasError())
	require.Equal(t, "vh-123", id.ValueString())

	var joinToDomain types.Bool
	require.False(t, resp.State.GetAttribute(ctx, path.Root("join_to_domain"), &joinToDomain).HasError())
	require.Equal(t, types.BoolValue(defaultJoinToDomain), joinToDomain)

	var awaitDeletionTask types.Bool
	require.False(t, resp.State.GetAttribute(ctx, path.Root("await_deletion_task"), &awaitDeletionTask).HasError())
	require.Equal(t, types.BoolValue(defaultAwaitDeletionTask), awaitDeletionTask)

	var allowRestart types.Bool
	require.False(t, resp.State.GetAttribute(ctx, path.Root("allow_restart"), &allowRestart).HasError())
	require.Equal(t, types.BoolValue(defaultAllowRestart), allowRestart)
}
