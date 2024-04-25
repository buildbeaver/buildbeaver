#!/bin/bash
set -e
if [ -n "${BB_DEBUG}" ]; then
  set -x
fi

AWS_DEFAULT_REGION=us-west-2
export AWS_DEFAULT_REGION

ENV_SCRIPT_DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
REPO_DIR=$(realpath "${ENV_SCRIPT_DIR}/../../../")
export REPO_DIR

BUILD_DIR="${REPO_DIR}/build/output"
export BUILD_DIR

cd "${REPO_DIR}"

pushd () {
    command pushd "$@" > /dev/null
}

popd () {
    command popd "$@" > /dev/null
}

# check_deps asserts that all of the binaries specified in $1 (as a space seperated string)
# exist in path. If a binary does not exist this will exit with exit code 1.
check_deps() {
  for dep in $1
  do
    if [ ! "$(which "${dep}")" ]; then
      echo "${dep} must be available."
      exit 1
    fi
  done
}
