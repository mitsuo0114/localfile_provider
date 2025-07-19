#!/usr/bin/env bash
set -euo pipefail

# Generate Terraform provider documentation.
# Uses explicit provider name to avoid errors from underscores in the
# repository name.

tfplugindocs --provider-name=localfile "$@"
