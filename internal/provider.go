package internal

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"os"
	"path/filepath"
)

// ProviderTypeName is the Terraform provider type name.
const ProviderTypeName = "localfile"

// Compile-time assertion to ensure provider implementation satisfies
// required interfaces.
var _ provider.Provider = &localfileProvider{}

// localfileProvider implements the Terraform provider interface.  It
// holds the provider version, which may be injected during build.
type localfileProvider struct {
	version string
}

// NewProvider returns a new provider instance with the given version.
// The version should be set during provider compilation for proper
// reporting in Terraform.
func NewProvider(version string) provider.Provider {
	return &localfileProvider{version: version}
}

// providerModel defines the configuration schema for the provider.
// It contains a single attribute for the base directory used by
// resources and data sources.
type providerModel struct {
	BaseDir types.String `tfsdk:"base_dir"`
}

// Metadata sets the provider type name and version.
func (p *localfileProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = ProviderTypeName
	resp.Version = p.version
}

// Schema defines the provider-level configuration schema.  The
// provider requires a base_dir parameter specifying where files will
// be created and read from.  It is marked as required and must be a
// valid directory path on the machine running Terraform.
func (p *localfileProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"base_dir": schema.StringAttribute{
				Required:    true,
				Description: "Base directory for all file operations. Must be an existing directory.",
			},
		},
		Description:         "The localfile provider manages simple text files and zip archives within a designated base directory.",
		MarkdownDescription: "The localfile provider manages simple text files and zip archives within a designated base directory.",
	}
}

// Configure validates the provider configuration and prepares a
// FileClient instance for use by resources and data sources.  It
// performs basic validation of the base directory and logs
// configuration operations using tflog.  If configuration fails,
// diagnostics are appended and returned to Terraform.
func (p *localfileProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Load configuration into model
	var config providerModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Ensure base_dir is known
	if config.BaseDir.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("base_dir"),
			"Unknown base_dir",
			"The provider cannot be configured because base_dir is unknown. Set base_dir in the provider configuration.",
		)
		return
	}
	// Validate base_dir value
	baseDir := config.BaseDir.ValueString()
	if baseDir == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("base_dir"),
			"Missing base_dir",
			"The base_dir must be specified for the localfile provider.",
		)
		return
	}
	// Resolve absolute path and ensure it exists
	absDir, err := filepath.Abs(baseDir)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("base_dir"),
			"Invalid base_dir",
			fmt.Sprintf("Cannot resolve base_dir: %s", err),
		)
		return
	}
	// Ensure directory exists
	info, err := os.Stat(absDir)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("base_dir"),
			"Invalid base_dir",
			fmt.Sprintf("Base directory does not exist: %s", err),
		)
		return
	}
	if !info.IsDir() {
		resp.Diagnostics.AddAttributeError(
			path.Root("base_dir"),
			"Invalid base_dir",
			"The base_dir must be a directory.",
		)
		return
	}
	// Log configuration details using tflog.  Set structured fields
	// for the base directory to aid in debugging.  Note that there is
	// no sensitive information to mask here.  These lines are based
	// on the HashiCorp logging tutorial, which recommends using
	// tflog.SetField and tflog.Debug/Info around client creation【844297507211234†L343-L365】.
	ctx = tflog.SetField(ctx, "local_file_base_dir", absDir)
	tflog.Debug(ctx, "Configuring localfile provider")
	// Initialize client
	client := &FileClient{BaseDir: absDir}
	// Expose client to resources and data sources
	resp.DataSourceData = client
	resp.ResourceData = client
	tflog.Info(ctx, "Configured localfile provider", map[string]any{"success": true})
}

// Resources returns the list of resource implementations supported by
// this provider.  Each entry is a factory function that returns a
// new resource instance.  The names correspond to the resource
// type names without the provider prefix.
func (p *localfileProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewTxtResource,
		NewZipResource,
	}
}

// DataSources returns the list of data source implementations
// supported by this provider.  Each entry is a factory function
// returning a new data source instance.
func (p *localfileProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewTxtDataSource,
	}
}
