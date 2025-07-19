package internal

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"os"
	"path/filepath"
)

// Ensure zipResource satisfies the required interfaces
var _ resource.Resource = &zipResource{}
var _ resource.ResourceWithConfigure = &zipResource{}
var _ resource.ResourceWithImportState = &zipResource{}

// zipResource manages zip archives containing a single file.
// Changing the source file or output location/name forces replacement.
type zipResource struct {
	client *FileClient
}

// zipResourceModel holds state data for the zip resource.  ID stores
// the absolute path of the zip file.  SrcFileID is the absolute path
// of the source file.  Name and Location are retained for display.
type zipResourceModel struct {
	ID        types.String `tfsdk:"id"`
	SrcFileID types.String `tfsdk:"src_data_file"`
	Name      types.String `tfsdk:"name"`
	Location  types.String `tfsdk:"location"`
}

// NewZipResource returns a new zip resource instance
func NewZipResource() resource.Resource {
	return &zipResource{}
}

// Metadata sets the resource type name.
func (r *zipResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_onefile_zip"
}

// Schema defines the attributes for the zip resource.  The
// src_data_file attribute should reference the ID of a localfile-txt
// resource (the absolute path to the file).  Name and location
// determine where the zip file is written.  Changes to these
// attributes require recreation.
func (r *zipResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Absolute path to the zip archive on disk.",
				MarkdownDescription: "Absolute path to the zip archive on disk.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"src_data_file": schema.StringAttribute{
				Required:            true,
				Description:         "Absolute path to the source file to include in the zip. Typically references a localfile-txt resource's id.",
				MarkdownDescription: "Absolute path to the source file to include in the zip. Typically references a localfile-txt resource's id.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				Description:         "Name of the zip archive file.",
				MarkdownDescription: "Name of the zip archive file.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"location": schema.StringAttribute{
				Optional:            true,
				Description:         "Subdirectory within the base directory to place the zip archive.",
				MarkdownDescription: "Subdirectory within the base directory to place the zip archive.",
				Default:             stringdefault.StaticString(""),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
		Description:         "Creates a zip archive containing a single source file.",
		MarkdownDescription: "Creates a zip archive containing a single source file.",
	}
}

// Configure stores the provider's FileClient on the resource
func (r *zipResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*FileClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Data Type",
			"The provider data for localfile_onefile_zip must be a *FileClient.",
		)
		return
	}
	r.client = client
}

// Create builds the zip file with the specified source file inside.
func (r *zipResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan zipResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	srcPath := plan.SrcFileID.ValueString()
	name := plan.Name.ValueString()
	loc := ""
	if !plan.Location.IsNull() && !plan.Location.IsUnknown() {
		loc = plan.Location.ValueString()
	}
	// Determine destination zip path
	zipPath, err := r.client.fullPath(loc, name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to determine zip path",
			err.Error(),
		)
		return
	}
	// Determine internal file name inside zip as base name of source
	internalName := filepath.Base(srcPath)
	// Create zip file
	if err := r.client.CreateZipFile(zipPath, srcPath, internalName); err != nil {
		resp.Diagnostics.AddError(
			"Error creating zip archive",
			err.Error(),
		)
		return
	}
	// Log
	ctx = tflog.SetField(ctx, "zip_path", zipPath)
	tflog.Info(ctx, "Created zip archive", map[string]any{"success": true})
	// Set state
	var state zipResourceModel
	state.ID = types.StringValue(zipPath)
	state.SrcFileID = types.StringValue(srcPath)
	state.Name = types.StringValue(name)
	if loc != "" {
		state.Location = types.StringValue(loc)
	} else {
		state.Location = types.StringValue("")
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read ensures the zip file exists.  If it does not, remove state.
func (r *zipResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state zipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	zipPath := state.ID.ValueString()
	if zipPath == "" {
		return
	}
	if _, err := os.Stat(zipPath); err != nil {
		if os.IsNotExist(err) {
			// Zip file no longer exists; remove resource
			resp.State.RemoveResource(ctx)
			tflog.Info(ctx, "Zip file removed from disk, removing from state", map[string]any{"path": zipPath})
			return
		}
		resp.Diagnostics.AddError(
			"Error reading zip file",
			err.Error(),
		)
		return
	}
	// Nothing else to update for read
}

// Update is not implemented because changes to any attribute require
// replacement.  The plan modifiers ensure Terraform recreates the
// resource when src_data_file, name, or location change.
func (r *zipResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// No-op
}

// Delete removes the zip file from disk and clears state.
func (r *zipResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state zipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	zipPath := state.ID.ValueString()
	if err := r.client.Delete(zipPath); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting zip file",
			err.Error(),
		)
		return
	}
	ctx = tflog.SetField(ctx, "zip_path", zipPath)
	tflog.Info(ctx, "Deleted zip archive", map[string]any{"success": true})
	resp.State.RemoveResource(ctx)
}

// ImportState allows importing an existing zip file.  The ID should
// be the absolute path to the zip file.  The source file cannot be
// determined during import and must be set manually afterwards.
func (r *zipResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importID := req.ID
	// Determine name and location relative to base dir
	rel, err := filepath.Rel(r.client.BaseDir, importID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Cannot determine relative path for import ID: %s", err),
		)
		return
	}
	name := filepath.Base(rel)
	loc := filepath.Dir(rel)
	// Build state attributes
	attrs := map[string]attr.Value{}
	attrs["id"] = types.StringValue(importID)
	attrs["name"] = types.StringValue(name)
	if loc == "." {
		attrs["location"] = types.StringValue("")
	} else {
		attrs["location"] = types.StringValue(loc)
	}
	// src_data_file cannot be inferred; leave unknown
	attrs["src_data_file"] = types.StringNull()
	// Set attributes
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), attrs["id"])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), attrs["name"])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("location"), attrs["location"])...)
	// Leave src_data_file null; will require user to specify in config
}
