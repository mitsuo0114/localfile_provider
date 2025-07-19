# localfile provider

## Generating Terraform documentation

This project uses `tfplugindocs` to generate Terraform documentation.
Install the tool and run it from the repository root.
Because the repository directory contains an underscore, `tfplugindocs`
cannot automatically determine the provider name. Run the helper script to
invoke the command with the correct name:

```bash
go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest

# Generate docs in the ./docs directory
scripts/generate-docs.sh
```

