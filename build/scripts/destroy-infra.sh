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
TERRAFORM_BASE="${REPO_DIR}/build/terraform/environment"
DEPENDENCIES=("terraform" "aws")

###############################################################################
# Default configuration
###############################################################################
export AWS_DEFAULT_REGION=us-west-2

###############################################################################
# Option parsing
###############################################################################
print_usage () {
  echo "destroy-infra.sh - Destroys infrastructure"
  echo "destroy-infra.sh [environment]"
  echo ""
}

if [ -z "$1" ]; then
  echo "The name of the environment directory must be the first argument"
  print_usage
  exit 1
fi

for dep in "${DEPENDENCIES[@]}"
do
  if [ ! "$(which "${dep}")" ]; then
    echo "${dep} must be available."
    exit 1
  fi
done

###############################################################################
# Execution
###############################################################################
VARS_FILE="${REPO_DIR}/build/environments/${1}/vars.tfvars"
if [ ! -f "$VARS_FILE" ]; then
    echo "Unable to find vars file at ${VARS_FILE}"
    exit 1
fi

ENVIRONMENT_VAR="environment"
ENVIRONMENT=$(grep "${ENVIRONMENT_VAR}" "${VARS_FILE}" | awk '{print $3}' | tr -d '"')
if [ -z "${ENVIRONMENT}" ]; then
  echo "The vars file must contain an '${ENVIRONMENT_VAR}' key"
  exit 1
fi

RESOURCE_PREFIX_VAR="resource_prefix"
RESOURCE_PREFIX=$(grep "${RESOURCE_PREFIX_VAR}" "${VARS_FILE}" | awk '{print $3}' | tr -d '"')
if [ -z "${RESOURCE_PREFIX}" ]; then
  echo "The vars file must contain a '${RESOURCE_PREFIX_VAR}' key"
  exit 1
fi

if [[ "${ENVIRONMENT}" == "prod" || "${ENVIRONMENT}" == "production" ]]; then
  echo "Refusing to destroy production (are you crazy!?)"
  exit 1
fi

echo ""
echo "!!!!!!!!!!!!!!!!!!!!!!!"
echo "        WARNING        "
echo "!!!!!!!!!!!!!!!!!!!!!!!"
echo ""
echo "You are DESTROYING all infrastructure for: ${RESOURCE_PREFIX}${ENVIRONMENT}"
echo "Push ctrl+c to exit"
echo ""
echo "Continuing after 10 seconds"
echo ""
sleep 10

# we need a temporary directory for the environment state
TERRAFORM_STATE_DIR="${TERRAFORM_OUTPUT}/state/${ENVIRONMENT}"
rm -rf "${TERRAFORM_STATE_DIR}"
mkdir -p "${TERRAFORM_STATE_DIR}"

echo "Destroying infrastructure..."
pushd "${TERRAFORM_STATE_DIR}"
  terraform init -input=false -from-module="${TERRAFORM_BASE}" -backend-config="key=${RESOURCE_PREFIX}${ENVIRONMENT}"
  terraform get
  terraform destroy -auto-approve -input=false -lock=false -var-file="$VARS_FILE"
popd

echo "Destroy complete."
