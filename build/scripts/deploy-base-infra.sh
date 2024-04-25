#!/bin/bash
set -e
if [ -n "${BB_DEBUG}" ]; then
  set -x
fi

###############################################################################
# Setup
###############################################################################
SCRIPT_DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
source "${SCRIPT_DIR}/lib/env.sh"
TERRAFORM_OUTPUT="${REPO_DIR}/build/output/terraform"
TERRAFORM_BASE="${REPO_DIR}/build/terraform/base"
check_deps "terraform aws"

###############################################################################
# Default configuration
###############################################################################
export AWS_DEFAULT_REGION=us-west-2

###############################################################################
# Option parsing
###############################################################################
print_usage () {
  echo "deploy-base-infra.sh - Deploys BuildBeaver's base infra"
  echo "deploy-base-infra.sh"
  echo ""
}

###############################################################################
# Execution
###############################################################################
# we need a temporary directory for the environment state
TERRAFORM_STATE_DIR="${TERRAFORM_OUTPUT}/state/base"
rm -rf "${TERRAFORM_STATE_DIR}"
mkdir -p "${TERRAFORM_STATE_DIR}"

echo "Bringing infrastructure up..."
pushd "${TERRAFORM_STATE_DIR}"

terraform init -input=false -from-module="${TERRAFORM_BASE}" -backend-config="key=base"
terraform get
terraform plan -out=tfplan -input=false
terraform apply -auto-approve -input=false tfplan

popd

echo "Base deploy complete."
