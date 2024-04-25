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
check_deps "docker aws"

###############################################################################
# Default configuration
###############################################################################
DOCKER_REGISTRY="733436759586.dkr.ecr.us-west-2.amazonaws.com"
PUSH=false

###############################################################################
# Option parsing
###############################################################################
print_usage () {
  echo "Builds a Docker image and and optionally pushes it to the registry"
  echo ""
  echo "build-docker.sh [opts] [docker_image_name]"
  echo "-p              Push to the registry"
  echo "-t              A custom tag to apply to the built image. Defaults to the version string derived from git"
}

if [ -z "$1" ]; then
  echo "The name of the Docker image must be the first argument"
  print_usage
  exit 1
fi

while getopts "pt:" opt; do
  case $opt in
    p) PUSH="true" ;;
    t) TAG="${OPTARG}" ;;
    \?)
      echo "Invalid option: -$OPTARG" >&2
      print_usage
      exit 1
      ;;
  esac
done
shift "$(( OPTIND - 1 ))"

BUILD_ROOT="${REPO_DIR}/build"
DOCKER_IMAGE="${1}"
DOCKER_FILE="${BUILD_ROOT}/docker/${DOCKER_IMAGE}/Dockerfile"
DOCKER_REPO="${DOCKER_REGISTRY}/${DOCKER_IMAGE}"

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
if [ ! -f "$DOCKER_FILE" ]; then
    echo "Unable to find Dockerfile at ${DOCKER_FILE}"
    exit 1
fi

if [ ! "$TAG" ]; then
    TAG="${VERSION}"
fi

pushd "${BUILD_ROOT}"
  if "$PUSH"; then
    echo "Building Docker image: '${DOCKER_IMAGE}' with tag '${DOCKER_REPO}:${TAG}'"
    docker build -t "${DOCKER_REPO}:${TAG}" -f "${DOCKER_FILE}" .
    echo ""
    echo "Logging in to registry"
    aws ecr get-login-password | docker login --username AWS --password-stdin ${DOCKER_REGISTRY}
    echo ""
    echo "Pushing to registry"
    docker push "${DOCKER_REPO}:${TAG}"
  else
    echo "Building Docker image: '${DOCKER_IMAGE}' with tag '${DOCKER_IMAGE}:${TAG}'"
    docker build -t "${DOCKER_IMAGE}:${TAG}" -f "${DOCKER_FILE}" .
  fi
popd
