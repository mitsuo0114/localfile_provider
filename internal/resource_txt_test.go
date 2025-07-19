package internal

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func setupTxtResource(t *testing.T) (*txtResource, rschema.Schema, string) {
	ctx := context.Background()
	tmp := t.TempDir()
	client := &FileClient{BaseDir: tmp}
	r := &txtResource{}
	r.Configure(ctx, resource.ConfigureRequest{ProviderData: client}, &resource.ConfigureResponse{})

	var schResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schResp)
	return r, schResp.Schema, tmp
}

func TestTxtResourceLifecycle(t *testing.T) {
	ctx := context.Background()
	r, schema, dir := setupTxtResource(t)

	// Create
	planState := tfsdk.State{Schema: schema}
	planState.Set(ctx, txtResourceModel{
		Name: types.StringValue("test.txt"),
		Data: types.StringValue("hello"),
	})
	createReq := resource.CreateRequest{Plan: tfsdk.Plan{Raw: planState.Raw, Schema: schema}}
	createResp := resource.CreateResponse{State: tfsdk.State{Schema: schema}}
	r.Create(ctx, createReq, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("create diag: %v", createResp.Diagnostics)
	}
	var state txtResourceModel
	createResp.State.Get(ctx, &state)
	path := filepath.Join(dir, "test.txt")
	path, _ = filepath.Abs(path)
	if state.ID.ValueString() != path {
		t.Fatalf("expected id %s, got %s", path, state.ID.ValueString())
	}
	b, err := os.ReadFile(path)
	if err != nil || string(b) != "hello" {
		t.Fatalf("file not written correctly")
	}

	// Update
	planState2 := tfsdk.State{Schema: schema}
	planState2.Set(ctx, txtResourceModel{
		Name: types.StringValue("test.txt"),
		Data: types.StringValue("bye"),
	})
	updateReq := resource.UpdateRequest{Plan: tfsdk.Plan{Raw: planState2.Raw, Schema: schema}, State: createResp.State}
	updateResp := resource.UpdateResponse{State: tfsdk.State{Schema: schema}}
	r.Update(ctx, updateReq, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("update diag: %v", updateResp.Diagnostics)
	}
	b, err = os.ReadFile(path)
	if err != nil || string(b) != "bye" {
		t.Fatalf("file not updated")
	}

	// Delete
	delReq := resource.DeleteRequest{State: updateResp.State}
	delResp := resource.DeleteResponse{State: tfsdk.State{Schema: schema}}
	r.Delete(ctx, delReq, &delResp)
	if delResp.Diagnostics.HasError() {
		t.Fatalf("delete diag: %v", delResp.Diagnostics)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("file still exists")
	}
}

func TestTxtResourceImportState(t *testing.T) {
	ctx := context.Background()
	r, schema, dir := setupTxtResource(t)

	// create existing file
	os.MkdirAll(filepath.Join(dir, "sub"), 0o755)
	filePath := filepath.Join(dir, "sub", "import.txt")
	os.WriteFile(filePath, []byte("data"), 0o644)

	impReq := resource.ImportStateRequest{ID: filePath}
	impState := tfsdk.State{Schema: schema}
	// initialize state so SetAttribute has a valid object to modify
	impState.Set(ctx, txtResourceModel{})
	impResp := resource.ImportStateResponse{State: impState}
	r.ImportState(ctx, impReq, &impResp)
	if impResp.Diagnostics.HasError() {
		t.Fatalf("import diag: %v", impResp.Diagnostics)
	}
	var state txtResourceModel
	impResp.State.Get(ctx, &state)
	if state.ID.ValueString() != filePath || state.Name.ValueString() != "import.txt" || state.Location.ValueString() != "sub" {
		t.Fatalf("unexpected import state: %#v", state)
	}
}
