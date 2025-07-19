package internal

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure txtDataSource satisfies the required interfaces
var _ datasource.DataSource = &txtDataSource{}
var _ datasource.DataSourceWithConfigure = &txtDataSource{}

// txtDataSource reads an existing text file from disk.  The data
// source requires the file name and optionally a subdirectory.  It
// returns the file contents and absolute path.
type txtDataSource struct {
	client *FileClient
}

// txtDataSourceModel maps configuration attributes to their values
// and holds the computed result of the data source.
type txtDataSourceModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Location types.String `tfsdk:"location"`
	Data     types.String `tfsdk:"data"`
}

// NewTxtDataSource returns a new data source instance
func NewTxtDataSource() datasource.DataSource {
	return &txtDataSource{}
}

// Metadata sets the type name for the data source
func (d *txtDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_txt"
}

// Schema defines the input and output attributes for the data source
func (d *txtDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Absolute path to the file on disk.",
				MarkdownDescription: "Absolute path to the file on disk.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				Description:         "Name of the file to read, including extension.",
				MarkdownDescription: "Name of the file to read, including extension.",
			},
			"location": schema.StringAttribute{
				Optional:            true,
				Description:         "Subdirectory within the base directory where the file resides.",
				MarkdownDescription: "Subdirectory within the base directory where the file resides.",
			},
			"data": schema.StringAttribute{
				Computed:            true,
				Description:         "Contents of the file.",
				MarkdownDescription: "Contents of the file.",
			},
		},
		Description:         "Reads an existing text file from the local filesystem.",
		MarkdownDescription: "Reads an existing text file from the local filesystem.",
	}
}

// Configure stores the FileClient on the data source
func (d *txtDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*FileClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Data Type",
			"The provider data for localfile_txt data source must be a *FileClient.",
		)
		return
	}
	d.client = client
}

// Read reads the file specified by name and location and returns its contents
func (d *txtDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config txtDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Compute file path using base directory
	name := config.Name.ValueString()
	if name == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("name"),
			"Missing file name",
			"The name attribute must be provided.",
		)
		return
	}
	location := ""
	if !config.Location.IsNull() && !config.Location.IsUnknown() {
		location = config.Location.ValueString()
	}
	fullPath, err := d.client.fullPath(location, name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid file path",
			err.Error(),
		)
		return
	}
	// Read file
	content, err := d.client.ReadFile(fullPath)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading file",
			fmt.Sprintf("Could not read file %s: %s", fullPath, err),
		)
		return
	}
	// Log read operation
	ctx = tflog.SetField(ctx, "file_path", fullPath)
	tflog.Debug(ctx, "Read text file via data source")
	// Populate state
	var state txtDataSourceModel
	state.ID = types.StringValue(fullPath)
	state.Name = types.StringValue(name)
	if location != "" {
		state.Location = types.StringValue(location)
	} else {
		state.Location = types.StringValue("")
	}
	state.Data = types.StringValue(content)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
