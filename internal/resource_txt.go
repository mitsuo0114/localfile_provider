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
	"path/filepath"
)

// Ensure txtResource satisfies required interfaces
var _ resource.Resource = &txtResource{}
var _ resource.ResourceWithConfigure = &txtResource{}
var _ resource.ResourceWithImportState = &txtResource{}

// txtResource manages plain text files within the base directory.  A
// change to the file name or location forces recreation, while
// updates to the content modify the existing file in place.
type txtResource struct {
	client *FileClient
}

// txtResourceModel maps the schema data to Go types.  The ID
// attribute stores the absolute file path.  Name and Location are
// kept for convenience and to detect changes.  Data represents the
// file contents.
type txtResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Location types.String `tfsdk:"location"`
	Data     types.String `tfsdk:"data"`
}

// NewTxtResource returns a new instance of the txt resource
func NewTxtResource() resource.Resource {
	return &txtResource{}
}

// Metadata sets the resource type name.
func (r *txtResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_txt"
}

// Schema defines the attributes for the txt resource.  Name and
// location determine the file path.  Data holds the contents.  The
// ID attribute stores the full absolute path and is computed from the
// configuration.  Changes to name or location trigger replacement
// through plan modifiers.【844297507211234†L343-L365】 demonstrates the
// structured logging used in Configure.
func (r *txtResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Absolute path to the file on disk.",
				MarkdownDescription: "Absolute path to the file on disk.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				Description:         "Name of the file, including extension.",
				MarkdownDescription: "Name of the file, including extension.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"location": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Subdirectory within the base directory to place the file.",
				MarkdownDescription: "Subdirectory within the base directory to place the file.",
				Default:             stringdefault.StaticString(""),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"data": schema.StringAttribute{
				Required:            true,
				Description:         "Contents to write to the file.",
				MarkdownDescription: "Contents to write to the file.",
			},
		},
		Description:         "Creates and manages a text file on the local filesystem.",
		MarkdownDescription: "Creates and manages a text file on the local filesystem.",
	}
}

// Configure stores the provider's FileClient on the resource.
func (r *txtResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*FileClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Data Type",
			"The provider data for localfile_txt must be a *FileClient.",
		)
		return
	}
	r.client = client
}

// Create writes the file to disk and records its path in state.
func (r *txtResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Read plan into model
	var plan txtResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Compute full path
	name := plan.Name.ValueString()
	location := ""
	if !plan.Location.IsNull() && !plan.Location.IsUnknown() {
		location = plan.Location.ValueString()
	}
	fullPath, err := r.client.fullPath(location, name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to determine file path",
			err.Error(),
		)
		return
	}
	// Write file content
	data := plan.Data.ValueString()
	if err := r.client.WriteFile(fullPath, data); err != nil {
		resp.Diagnostics.AddError(
			"Error writing file",
			err.Error(),
		)
		return
	}
	// Log creation
	ctx = tflog.SetField(ctx, "file_path", fullPath)
	tflog.Info(ctx, "Created text file", map[string]any{"success": true})
	// Set state
	var state txtResourceModel
	state.ID = types.StringValue(fullPath)
	state.Name = types.StringValue(name)
	if location != "" {
		state.Location = types.StringValue(location)
	} else {
		state.Location = types.StringValue("")
	}
	state.Data = types.StringValue(data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes state with the contents of the file.  If the file
// does not exist, the resource is removed from state.
func (r *txtResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state txtResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Use the stored ID as the file path
	pathStr := state.ID.ValueString()
	// If path is empty, nothing to do
	if pathStr == "" {
		return
	}
	// Read file
	content, err := r.client.ReadFile(pathStr)
	if err != nil {
		// If file missing, remove state
		resp.State.RemoveResource(ctx)
		tflog.Info(ctx, "File no longer exists, removing from state", map[string]any{"path": pathStr})
		return
	}
	// Update state Data with actual file contents
	state.Data = types.StringValue(content)
	// Keep existing name and location; they are part of state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update modifies the file contents if the data has changed.  Name
// and location changes trigger replacement via plan modifiers and are
// not handled here.
func (r *txtResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan txtResourceModel
	var state txtResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Only update file content if it has changed
	if plan.Data.ValueString() != state.Data.ValueString() {
		pathStr := state.ID.ValueString()
		if err := r.client.WriteFile(pathStr, plan.Data.ValueString()); err != nil {
			resp.Diagnostics.AddError(
				"Error updating file",
				err.Error(),
			)
			return
		}
		// Log update
		ctx = tflog.SetField(ctx, "file_path", pathStr)
		tflog.Info(ctx, "Updated text file contents", map[string]any{"success": true})
	}
	// Update state
	state.Data = types.StringValue(plan.Data.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes the file from disk and clears state.
func (r *txtResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state txtResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	pathStr := state.ID.ValueString()
	if err := r.client.Delete(pathStr); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting file",
			err.Error(),
		)
		return
	}
	ctx = tflog.SetField(ctx, "file_path", pathStr)
	tflog.Info(ctx, "Deleted text file", map[string]any{"success": true})
	// Remove state
	resp.State.RemoveResource(ctx)
}

// ImportState allows users to import an existing file.  The import ID
// should be the absolute path to the file.  The method derives the
// name and location from the path relative to the provider's base
// directory.
func (r *txtResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID is absolute path
	importID := req.ID
	// Derive name and location relative to base directory
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
	// If loc is '.' treat as root
	if loc == "." {
		attrs["location"] = types.StringValue("")
	} else {
		attrs["location"] = types.StringValue(loc)
	}
	// Data will be populated on Read
	attrs["data"] = types.StringNull()
	// Set state
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), attrs["id"])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), attrs["name"])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("location"), attrs["location"])...)
	// Data left null; will be filled by Read
}
