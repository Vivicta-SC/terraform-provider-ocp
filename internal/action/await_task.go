// Copyright @ Vivicta. All Rights Reserved. 2026
// SPDX-License-Identifier: MPL-2.0

package action

import (
	"context"
	"time"

	"github.com/Vivicta-SC/terraform-provider-ocp/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ action.ActionWithConfigure = &awaitTaskAction{}

func NewAwaitTaskAction() action.Action { return &awaitTaskAction{} }

type awaitTaskAction struct{ client *client.OCPClient }

func (a *awaitTaskAction) Metadata(_ context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_await_task"
}
func (a *awaitTaskAction) Configure(_ context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	a.client = req.ProviderData.(*client.OCPClient)
}
func (a *awaitTaskAction) Schema(_ context.Context, _ action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Action allowing to await any task",
		Attributes: map[string]schema.Attribute{
			"task_id": schema.StringAttribute{Required: true, Description: "ID of task"},
		},
	}
}

const taskActionquery = `
query task($id: GlobalID!) {
  taskExecution(id: $id) {
    id
    state
  }
}
`

func (a *awaitTaskAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var taskID types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("task_id"), &taskID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		var res struct {
			TaskExecution struct {
				ID    string
				State string
			}
		}

		err := a.client.Do(
			ctx,
			client.GQLRequest{
				Query:     taskActionquery,
				Variables: map[string]interface{}{"id": taskID.ValueString()},
			},
			&res,
			&client.DoOpts{Diags: &resp.Diagnostics},
		)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", err.Error())
			return
		}

		switch res.TaskExecution.State {
		case "RUNNING":
			resp.SendProgress(action.InvokeProgressEvent{
				Message: "Awaiting task to finish...",
			})
		case "SUCCESS":
			return

		case "CANCELED", "FAILURE", "INVALID_INPUT":
			resp.Diagnostics.AddError("Task failed", "completely")
			return
		}
		select {
		case <-ctx.Done():
			resp.Diagnostics.AddError("Cancelled", "context cancelled while waiting for task")
			return

		case <-ticker.C:
			continue
		}
	}
}
