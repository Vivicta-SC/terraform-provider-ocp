// Copyright @ Vivicta. All Rights Reserved. 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type OCPClient struct {
	endpoint string
	token    string
	debug    bool
	http     *http.Client
}

func New(endpoint, token string, verifySsl bool, debug bool) *OCPClient {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: !verifySsl},
	}
	return &OCPClient{
		endpoint: endpoint,
		token:    token,
		debug:    debug,
		http:     &http.Client{Transport: transport},
	}
}

type GQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
	Operation string                 `json:"operationName,omitempty"`
}

type GQLError struct {
	Message string        `json:"message"`
	Path    []interface{} `json:"path"`
}

func (e *GQLError) FormattedPath() string {
	if len(e.Path) == 0 {
		return "?"
	}
	var parts []string
	for _, p := range e.Path {
		parts = append(parts, fmt.Sprint(p))
	}
	return strings.Join(parts, ".")
}

type GQLResponse struct {
	Data       json.RawMessage `json:"data"`
	Errors     []GQLError      `json:"errors"`
	Extensions struct {
		Warnings     []GQLError `json:"warnings"`
		Deprecations []GQLError `json:"deprecations"`
	} `json:"extensions"`
}

type DoOpts struct {
	Warnings     *[]GQLError
	Deprecations *[]GQLError
	Diags        *diag.Diagnostics
}

func (c *OCPClient) Do(ctx context.Context, request GQLRequest, result interface{}, opts ...*DoOpts) error {
	var opt *DoOpts
	if len(opts) == 1 {
		opt = opts[0]
	} else if len(opts) > 1 {
		return fmt.Errorf("invalid options: expected at most 1 DoOpts, got %d", len(opts))
	}

	reqBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to jsonify GQLRequest: %w", err)
	}

	tflog.Debug(ctx, "Sending GQL request", map[string]interface{}{
		"endpoint": c.endpoint,
		"query":    request.Query,
		"vars":     request.Variables,
		"op":       request.Operation,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request to %s: %w", c.endpoint, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", c.token)
	if c.debug {
		req.Header.Set("X-Gql-Warning", "yes")
		req.Header.Set("X-Gql-Deprecated", "yes")
		req.Header.Set("X-Gql-Exceptions", "yes")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	tflog.Debug(ctx, "Received GQL response", map[string]interface{}{
		"status_code": resp.StatusCode,
		"body":        string(body),
	})

	if resp.StatusCode != http.StatusOK {
		bodyPreview := string(body)
		if len(bodyPreview) > 500 {
			bodyPreview = bodyPreview[:500] + "... (truncated)"
		}
		return fmt.Errorf("GQL request failed with status %d: %s", resp.StatusCode, bodyPreview)
	}

	var gqlResp GQLResponse
	if err := json.Unmarshal(body, &gqlResp); err != nil {
		return fmt.Errorf("failed to parse GQL response: %w", err)
	}

	if opt != nil {
		if opt.Warnings != nil {
			*opt.Warnings = gqlResp.Extensions.Warnings
		}
		if opt.Deprecations != nil {
			*opt.Deprecations = gqlResp.Extensions.Deprecations
		}
		if opt.Diags != nil {
			for _, w := range gqlResp.Extensions.Warnings {
				opt.Diags.AddWarning("API Warning", w.Message)
			}
			for _, w := range gqlResp.Extensions.Deprecations {
				opt.Diags.AddWarning("API Deprecation", w.Message)
			}
		}
	}

	if len(gqlResp.Errors) > 0 {
		var errMessages []string
		for _, gqlErr := range gqlResp.Errors {
			errMessages = append(errMessages, fmt.Sprintf("\t[%s] %s", gqlErr.FormattedPath(), gqlErr.Message))
		}
		return fmt.Errorf("GQL request returned %d error(s):\n%s", len(gqlResp.Errors), strings.Join(errMessages, "\n"))
	}

	if err := json.Unmarshal(gqlResp.Data, result); err != nil {
		return fmt.Errorf("failed to unmarshal GQL data into result: %w", err)
	}

	return nil
}

type GQLMutationValidationErrors struct {
	Typename string `json:"__typename"`
	Message  string `json:"message"`
	Errors   []struct {
		Field    string   `json:"field"`
		Messages []string `json:"messages"`
	} `json:"errors"`
}

type GQLMutationError struct {
	Typename string `json:"__typename"`
	Message  string `json:"message"`
}

func (c *OCPClient) DoMutate(ctx context.Context, request GQLRequest, result interface{}, opts ...*DoOpts) error {
	var raw struct {
		Data json.RawMessage `json:"data"`
	}

	if err := c.Do(ctx, request, &raw, opts...); err != nil {
		return err
	}

	var probe struct {
		Typename string `json:"__typename"`
	}
	if err := json.Unmarshal(raw.Data, &probe); err != nil {
		return fmt.Errorf("failed to parse __typename from GQL response: %w", err)
	}

	switch probe.Typename {
	case "":
		return errors.New("GQL response mutation's __typename is empty")
	case "Error":
		var gqlError GQLMutationError
		if err := json.Unmarshal(raw.Data, &gqlError); err != nil {
			return err
		}
		return fmt.Errorf("mutation failed with: %s", gqlError.Message)
	case "ValidationErrors":
		var valErrors GQLMutationValidationErrors
		if err := json.Unmarshal(raw.Data, &valErrors); err != nil {
			return err
		}

		var b strings.Builder
		for _, fe := range valErrors.Errors {
			if fe.Field != "" {
				fmt.Fprintf(&b, "%s: ", fe.Field)
			}
			b.WriteString(strings.Join(fe.Messages, ", "))
			b.WriteString("\n")
		}
		return errors.New(strings.TrimSpace(b.String()))
	default:
		if result == nil {
			return nil
		}
		if err := json.Unmarshal(raw.Data, result); err != nil {
			return fmt.Errorf("failed to unmarshal GQL mutation data into result: %w", err)
		}
		return nil
	}
}

const query = `
query task($id: GlobalID!) {
  taskExecution(id: $id) {
    id
    state
  }
}`

func (c *OCPClient) AwaitTask(ctx context.Context, taskID string, opts ...*DoOpts) error {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		var res struct {
			TaskExecution struct {
				ID    string
				State string
			}
		}
		if err := c.Do(
			ctx,
			GQLRequest{
				Query:     query,
				Variables: map[string]interface{}{"id": taskID},
			},
			&res,
			opts...,
		); err != nil {
			return err
		}

		switch res.TaskExecution.State {
		case "SUCCESS":
			return nil

		case "CANCELED", "FAILURE", "INVALID_INPUT":
			return errors.New("task failed")
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("awaiting task %s timed out: %s", taskID, ctx.Err())
		case <-ticker.C:
			continue
		}
	}
}

type node interface {
	GetID() string
}

type NodeGQL struct {
	Typename string `json:"__typename"`
	ID       string `json:"id"`
}

func (s *NodeGQL) GetID() string {
	return s.ID
}

type ConnectionGQL[T node] struct {
	Edges []struct {
		Node T `json:"node"`
	} `json:"edges"`
}

func (s *ConnectionGQL[T]) GetIDs() []string {
	ids := []string{}
	for _, x := range s.Edges {
		ids = append(ids, x.Node.GetID())
	}
	return ids
}

func (s *ConnectionGQL[T]) GetNodes() []T {
	nodes := make([]T, 0, len(s.Edges))
	for _, x := range s.Edges {
		nodes = append(nodes, x.Node)
	}
	return nodes
}

type ConnectionNodeGQL struct{ ConnectionGQL[*NodeGQL] }

const ProjectQuery = `
fragment ProjectFrag on ProjectNode {
  id
  name
  note
  customer { id }
  separationPod { id }
}

query get($id: GlobalID, $filters: ProjectFilter, $required: Boolean! = true) {
  	data: project(id: $id, filters: $filters, required: $required) { ...ProjectFrag }
}
mutation create($input: ProjectCreateInput!) {
	data: projectCreate(input: $input) {
		__typename
		... on ProjectNode { ...ProjectFrag }
		... on ValidationErrors { message errors { field messages } }
		... on Error { message }
	}
}
mutation update($input: ProjectUpdateInput!) {
	data: projectUpdate(input: $input) {
		__typename
		... on ProjectNode { ...ProjectFrag }
		... on ValidationErrors { message errors { field messages } }
		... on Error { message }
	}
}
mutation delete($id: GlobalID!) {
	data: projectDelete(input: {project: $id}) {
		__typename
		... on ProjectNode { id }
		... on ValidationErrors { message errors { field messages } }
		... on Error { message }
	}
}
`

type ProjectGQL struct {
	NodeGQL

	Customer      struct{ ID string }
	SeparationPod struct{ ID string }

	Name string
	Note string
}

const SeparationPodQuery = `
fragment SeparationPodFrag on SeparationPodNode {
	id
	customer { id }
	name
	note
	solutionType

	allowSharedPrimaryCluster
	allowSharedSecondaryCluster

	addNewDataProtectionPolicies
	addNewDedicatedClusters
	addNewDomains
	addNewNetworks
	addNewTiers
	addNewOsDistributions
	addNewPatchingWindows
	addNewWorkflows

	osDistributionList
	dataProtectionPolicyList(first: 100) { edges { node { id } } }
	dedicatedClusterList(first: 100) { edges { node { id } } }
	domainList(first: 100) { edges { node { id } } }
	networkList(first: 100) { edges { node { id } } }
	patchingWindowList(first: 100) { edges { node { id } } }
	tierList(first: 100) { edges { node { id } } }
	workflowList(first: 100) { edges { node { id } } }
}

query get($id: GlobalID, $filters: SeparationPodFilter, $required: Boolean! = true) {
  	data: separationPod(id: $id, filters: $filters, required: $required) { ...SeparationPodFrag }
}
mutation create($input: SeparationPodCreateV2Input!) {
	data: separationPodCreateV2(input: $input) {
		__typename
		... on SeparationPodNode { ...SeparationPodFrag }
		... on ValidationErrors { message errors { field messages } }
		... on Error { message }
	}
}
mutation update($input: SeparationPodUpdateInput!) {
	data: separationPodUpdate(input: $input) {
		__typename
		... on SeparationPodNode { ...SeparationPodFrag }
		... on ValidationErrors { message errors { field messages } }
		... on Error { message }
	}
}
mutation delete($id: GlobalID!) {
	data: separationPodDelete(input: {separationPod: $id}) {
		__typename
		... on SeparationPodNode { id }
		... on ValidationErrors { message errors { field messages } }
		... on Error { message }
	}
}
`

type SeparationPodGQL struct {
	NodeGQL

	Customer struct {
		ID string `json:"id"`
	} `json:"customer"`
	Name     string `json:"name"`
	Note     string `json:"note"`
	Solution string `json:"solutionType"`

	AllowSharedPrimaryCluster   bool `json:"allowSharedPrimaryCluster"`
	AllowSharedSecondaryCluster bool `json:"allowSharedSecondaryCluster"`

	AddNewDataProtectionPolicies bool `json:"addNewDataProtectionPolicies"`
	AddNewDedicatedClusters      bool `json:"addNewDedicatedClusters"`
	AddNewDomains                bool `json:"addNewDomains"`
	AddNewNetworks               bool `json:"addNewNetworks"`
	AddNewOsDistributions        bool `json:"addNewOsDistributions"`
	AddNewPatchingWindows        bool `json:"addNewPatchingWindows"`
	AddNewTiers                  bool `json:"addNewTiers"`
	AddNewWorkflows              bool `json:"addNewWorkflows"`

	OSDistributions        []string          `json:"osDistributionList"`
	DataProtectionPolicies ConnectionNodeGQL `json:"dataProtectionPolicyList"`
	DedicatedClusters      ConnectionNodeGQL `json:"dedicatedClusterList"`
	Domains                ConnectionNodeGQL `json:"domainList"`
	Networks               ConnectionNodeGQL `json:"networkList"`
	PatchingWindows        ConnectionNodeGQL `json:"patchingWindowList"`
	Tiers                  ConnectionNodeGQL `json:"tierList"`
	Workflows              ConnectionNodeGQL `json:"workflowList"`
}

const VMQuery = `
fragment VirtualHostFrag on VirtualHostNode {
  id
  customer { id }
  domain { id }
  project { id }
  template { id }
  dataProtectionPolicy { id }
  tier { id }
  hostname
  note
  region
  cpuCount
  coresPerSocket
  memorySizeGB
  antivirusType
  clusterType
  localDiskList(first: 100) { edges { node {
    id
	key
    sizeGB
  } } }
  ipAddressList(first: 100) { edges { node {
    id 
    ip 
    network { id }
  } } }
  networkInterfaceList {
    id
    label
    defaultGwIp
    network { id }
    ipv4Addresses { ip }
    ipv6Addresses { ip }
  }
  tagList(first: 100) { edges { node {
    id
  	name
    content
  } } }
}
query get($id: GlobalID, $filters: VirtualHostFilter, $required: Boolean! = true) {
  data: virtualHost(id: $id, filters: $filters, required: $required) { ...VirtualHostFrag }
}
mutation create($input: VirtualHostCreateInput!) {
  data: virtualHostCreate(input: $input) {
    __typename
    ... on VirtualHostCreated {
      virtualHost { ...VirtualHostFrag }
      taskExecution { id }
    }
    ... on ValidationErrors { message errors { field messages } }
    ... on Error { message }
  }
}
mutation update($input: VirtualHostUpdateInput!) {
  data: virtualHostUpdate(input: $input) {
    __typename
    ... on VirtualHostNode { id }
    ... on ValidationErrors { message errors { field messages } }
    ... on Error { message }
  }
}
mutation resize($input: VirtualHostResizeInput!) {
  data: virtualHostResize(input: $input) {
    __typename
    ... on TaskExecutionNode { id }
    ... on ValidationErrors { message errors { field messages } }
    ... on Error { message }
  }
}
mutation delete($id: GlobalID!) {
  data: virtualHostDelete(input: {virtualHost: $id, gracePeriod: 0}) {
    __typename
    ... on VirtualHostDeleteResult { taskExecution { id virtualHostList(first:2) { edges {node { id } } } } }
    ... on ValidationErrors { message errors { field messages } }
    ... on Error { message }
  }
}
`

type DiskGQL struct {
	NodeGQL

	Key    int32
	SizeGB int32
}

type ConnectionDiskGQL struct{ ConnectionGQL[*DiskGQL] }

type NICGQL struct {
	NodeGQL // Does not really implement Node interface

	Network          NodeGQL               `json:"network"`
	Label            string                `json:"label"`
	DefaultGatewayIP string                `json:"defaultGwIp"`
	IPv4Addresses    []struct{ IP string } `json:"ipv4Addresses"`
	IPv6Addresses    []struct{ IP string } `json:"ipv6Addresses"`
}

type VMIPGQL struct {
	NodeGQL

	IP      string
	Network NodeGQL `json:"network"`
}

type ConnectionIPGQL struct{ ConnectionGQL[*VMIPGQL] }

type VMGQL struct {
	NodeGQL

	Customer             struct{ ID string }
	Domain               struct{ ID string }
	Project              struct{ ID string }
	Template             struct{ ID string }
	DataProtectionPolicy struct{ ID string }
	Tier                 struct{ ID string }
	Hostname             string
	Note                 string
	Region               string
	CpuCount             int32
	CoresPerSocket       int32
	MemorySizeGB         int32
	ClusterType          string
	AntivirusType        string

	Disks                ConnectionDiskGQL `json:"localDiskList"`
	NetworkInterfaceList []NICGQL          `json:"networkInterfaceList"`
	Tags                 ConnectionNodeGQL `json:"tagList"`
	IpAddressList        ConnectionIPGQL   `json:"ipAddressList"`
}

const StaasGroupQuery = `
fragment StaasGroupFrag on StaasGroupNode {
  id
  name
  note
  protocol
  project { id }
  dataProtectionPolicy { id }
  tier { id }
  vserver { id }
  visibility {
    __typename
    ... on IpAddressNode { id }
    ... on SubnetNode { id }
    ... on StaasGroupiScsiVisibility {
      network { id }
      virtualHost { id }
    }
  }
}

query get($id: GlobalID, $filters: StaasGroupFilter, $required: Boolean! = true) {
  data: staasGroup(id: $id, filters: $filters, required: $required) { ...StaasGroupFrag }
}
mutation create($input: StaasGroupCreateInput!) {
  data: staasGroupCreate(input: $input) {
    __typename
    ... on StaasGroupNode{ ...StaasGroupFrag }
    ... on ValidationErrors { message errors { field messages } }
    ... on Error { message }
  }
}
mutation update($input: StaasGroupUpdateInput!) {
  data: staasGroupUpdate(input: $input) {
    __typename
    ... on StaasGroupNode { ...StaasGroupFrag }
    ... on ValidationErrors { message errors { field messages } }
    ... on Error { message }
  }
}
mutation addNFSExport($input: StaasGroupAddNfsExportInput!) {
  data: staasGroupAddNfsExport(input: $input) {
    __typename
    ... on TaskExecutionNode { id }
    ... on ValidationErrors { message errors { field messages } }
		... on Error { message }
  }
}
mutation delete($id: GlobalID!) {
  data: staasGroupDelete(input: {staasGroup: $id}) {
    __typename
    ... on StaasGroupNode { id }
    ... on ValidationErrors { message errors { field messages } }
    ... on Error { message }
  }
}
`

type VserverGQL struct {
	NodeGQL

	Name           string
	Customer       NodeGQL
	StorageCluster struct{ StorageType string }
	Region         string
	SolutionType   string
}

const VserverQuery = `
fragment VserverFrag on VserverNode {
  id
  name
  customer { id }
  storageCluster { storageType }
  region
  solutionType
}

query get($id: GlobalID, $filters: VserverFilter, $required: Boolean! = true) {
  data: vserver(id: $id, filters: $filters, required: $required) { ...VserverFrag }
}
`

type StaasGroupGQL struct {
	NodeGQL

	Project              NodeGQL
	DataProtectionPolicy NodeGQL
	Tier                 NodeGQL
	Vserver              NodeGQL
	Name                 string
	Note                 string
	Protocol             string
}

const StaasVolumeQuery = `
fragment StaasVolumeFrag on VolumeNode {
  id
  name
  note
  protocol
  project { id }
  dataProtectionPolicy { id }
  tier { id }
  vserver { id }
  visibility {
    __typename
    ... on IpAddressNode { id }
    ... on SubnetNode { id }
    ... on iScsiVisibility {
      network { id }
      virtualHost { id }
    }
  }
}

query get($id: GlobalID, $filters: VolumeFilter, $required: Boolean! = true) {
  	data: volume(id: $id, filters: $filters, required: $required) { ...StaasVolumeFrag }
}
mutation create($input: VolumeCreateStaasV2Input!) {
	data: volumeCreateStaasV2(input: $input) {
		__typename
		... on VolumeCreateStaasV2Result{ 
      taskExecution { id } 
      volume { ...StaasVolumeFrag }
    }
		... on ValidationErrors { message errors { field messages } }
		... on Error { message }
	}
}
mutation update($input: VolumeUpdateInput!) {
	data: volumeUpdate(input: $input) {
		__typename
		... on VolumeNode { ...StaasVolumeFrag }
		... on ValidationErrors { message errors { field messages } }
		... on Error { message }
	}
}
mutation resizeISCSI($input: VolumeResizeIscsiInput!) {
	data: volumeResizeIscsi(input: $input) {
		__typename
		... on TaskExecutionNode { id }
		... on ValidationErrors { message errors { field messages } }
		... on Error { message }
	}
}
mutation resizeNAS($input: VolumeResizeNasInput!) {
	data: volumeResizeNas(input: $input) {
		__typename
		... on TaskExecutionNode { id }
		... on ValidationErrors { message errors { field messages } }
		... on Error { message }
	}
}
mutation delete($id: GlobalID!, $retention: Int) {
	data: volumeDelete(input: {volume: $id, retention: $retention}) {
		__typename
		... on TaskExecutionNode{ id }
		... on ValidationErrors { message errors { field messages } }
		... on Error { message }
	}
}
`

type StaasVolumeGQL struct {
	NodeGQL
	Name                 string
	Note                 string
	Protocol             string
	Project              NodeGQL
	DataProtectionPolicy NodeGQL
	Tier                 NodeGQL
	Vserver              NodeGQL

	Visibility []struct {
		Typename string `json:"__typename"`
		ID       string
	}
}

const IPQuery = `
fragment IPFrag on IpAddressNode {
  id
  ip
  network { id }
}

query get($id: GlobalID, $filters: IpAddressFilter, $required: Boolean! = true) {
  data: ipAddress(id: $id, filters: $filters, required: $required) { ...IPFrag }
}
`
