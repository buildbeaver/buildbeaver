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
VERSION=$("${SCRIPT_DIR}"/version-info.sh)
DEPENDENCIES=("grep" "awk" "aws")

###############################################################################
# Default configuration
###############################################################################
DOCKER_REGISTRY="733436759586.dkr.ecr.us-west-2.amazonaws.com"
export AWS_DEFAULT_REGION=us-west-2

###############################################################################
# Option parsing
###############################################################################
print_usage () {
  echo "deploy-backend.sh - Deploys the BuildBeaver backend"
  echo "deploy-backend.sh [environment]"
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

BB_SERVER_CONTAINER_REPO_VAR="bb_server_container_repo"
BB_SERVER_CONTAINER_REPO=$(grep "${BB_SERVER_CONTAINER_REPO_VAR}" "${VARS_FILE}" | awk '{print $3}' | tr -d '"')
if [ -z "${RESOURCE_PREFIX}" ]; then
  echo "The vars file must contain a '${BB_SERVER_CONTAINER_REPO}' key"
  exit 1
fi

RESOURCE_NAME="${RESOURCE_PREFIX}${ENVIRONMENT}"

echo "Logging in to registry"
aws ecr get-login-password | docker login --username AWS --password-stdin ${DOCKER_REGISTRY}

echo ""
echo "Pulling bb-server:${VERSION}"
docker pull "${BB_SERVER_CONTAINER_REPO}:${VERSION}"

echo ""
echo "Tagging bb-server:${VERSION} with bb-server:${RESOURCE_NAME}-latest"
docker tag "${BB_SERVER_CONTAINER_REPO}:${VERSION}" "${BB_SERVER_CONTAINER_REPO}:${RESOURCE_NAME}-latest"

echo ""
echo "Pushing bb-server:${RESOURCE_NAME}-latest"
docker push "${BB_SERVER_CONTAINER_REPO}:${RESOURCE_NAME}-latest"

echo ""
echo "Triggering new deployment of ${RESOURCE_NAME} backend"
out=$(aws ecs update-service --cluster "${RESOURCE_NAME}" --service "${RESOURCE_NAME}" --force-new-deployment)
if [ -n "${BB_DEBUG}" ]; then
  echo "$out"
fi

echo ""
echo "Done"
