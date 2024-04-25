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
DEPENDENCIES=("grep" "awk" "aws" "yarn")

###############################################################################
# Default configuration
###############################################################################

export AWS_DEFAULT_REGION=us-west-2

###############################################################################
# Option parsing
###############################################################################

print_usage () {
  echo "deploy-frontend.sh - Deploys the BuildBeaver frontend"
  echo "deploy-frontend.sh [environment]"
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

APP_DNS_VAR="dns_app_subdomain"
APP_DNS=$(grep "${APP_DNS_VAR}" "${VARS_FILE}" | awk '{print $3}' | tr -d '"')
if [ -z "$APP_DNS" ]; then
  echo "The vars file must contain a '${APP_DNS_VAR}' key"
  exit 1
fi

CF_DISTRIBUTION=$(aws cloudfront list-distributions --query 'DistributionList.Items[?not_null(Origins.Items[?starts_with(DomainName,`'"${APP_DNS}"'`) == `true`])].Id' --output text)
if [ -z "$CF_DISTRIBUTION" ]; then
  echo "Couldn't locate the CloudFront distribution for APP_DNS ${APP_DNS}"
  exit 1
fi

FRONTEND_BUILD_PATH="${REPO_DIR}/build/output/frontend"

echo "Installing dependencies..."
pushd "${REPO_DIR}/frontend"
  yarn install
popd

echo "Building frontend..."
pushd "${REPO_DIR}/frontend"
  REACT_APP_BB_API_ENDPOINT="https://${APP_DNS}/api/v1" yarn build
popd

echo ""
echo "Deploying frontend from ${FRONTEND_BUILD_PATH} to ${APP_DNS}"
aws s3 cp "${FRONTEND_BUILD_PATH}/" "s3://${APP_DNS}/" --recursive

echo ""
echo "Invalidating CloudFront distribution ${CF_DISTRIBUTION}"
# See https://github.com/aws/aws-cli/issues/4973 for why AWS_PAGER="" is needed
AWS_PAGER="" aws cloudfront create-invalidation --distribution-id "${CF_DISTRIBUTION}" --paths "/*"