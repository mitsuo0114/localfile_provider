# localfile_provider

## Generating Terraform documentation

This project uses `tfplugindocs` to generate Terraform documentation.
Install the tool and run it from the repository root:

```bash
go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest

# Generate docs in the ./docs directory
tfplugindocs
```

