package internal

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestTxtDataSourceRead(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()
	client := &FileClient{BaseDir: tmp}

	// create file
	os.MkdirAll(filepath.Join(tmp, "dir"), 0o755)
	os.WriteFile(filepath.Join(tmp, "dir", "file.txt"), []byte("hello"), 0o644)

	ds := &txtDataSource{}
	ds.Configure(ctx, datasource.ConfigureRequest{ProviderData: client}, &datasource.ConfigureResponse{})

	var schResp datasource.SchemaResponse
	ds.Schema(ctx, datasource.SchemaRequest{}, &schResp)
	schema := schResp.Schema

	// Build config
	cfgState := tfsdk.State{Schema: schema}
	cfgState.Set(ctx, txtDataSourceModel{
		Name:     types.StringValue("file.txt"),
		Location: types.StringValue("dir"),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Raw: cfgState.Raw, Schema: schema}}
	resp := datasource.ReadResponse{State: tfsdk.State{Schema: schema}}
	ds.Read(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}

	var state txtDataSourceModel
	resp.State.Get(ctx, &state)
	expectedID, _ := filepath.Abs(filepath.Join(tmp, "dir", "file.txt"))
	if state.ID.ValueString() != expectedID {
		t.Fatalf("expected ID %s, got %s", expectedID, state.ID.ValueString())
	}
	if state.Data.ValueString() != "hello" {
		t.Fatalf("expected data hello, got %s", state.Data.ValueString())
	}
}

func TestTxtDataSourceMissingName(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()
	client := &FileClient{BaseDir: tmp}

	ds := &txtDataSource{}
	ds.Configure(ctx, datasource.ConfigureRequest{ProviderData: client}, &datasource.ConfigureResponse{})

	var schResp datasource.SchemaResponse
	ds.Schema(ctx, datasource.SchemaRequest{}, &schResp)
	schema := schResp.Schema

	cfgState := tfsdk.State{Schema: schema}
	cfgState.Set(ctx, txtDataSourceModel{Name: types.StringValue("")})

	req := datasource.ReadRequest{Config: tfsdk.Config{Raw: cfgState.Raw, Schema: schema}}
	resp := datasource.ReadResponse{State: tfsdk.State{Schema: schema}}
	ds.Read(ctx, req, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected error for missing name")
	}
}
