#!/bin/bash
set -e
if [ -n "${BB_DEBUG}" ]; then
  set -x
fi

SCRIPT_DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
. "${SCRIPT_DIR}/../lib/go-env.sh"
check_deps "go"

# Specify our ldflags for injecting our version information into our binaries.
PKG="github.com/buildbeaver/buildbeaver"
VERSION_INFO=$(${SCRIPT_DIR}/../version-info.sh)
GIT_SHA_SHORT=$(${SCRIPT_DIR}/../version-info.sh sha-short)
VERSION_VAR="-X '${PKG}/common/version.VERSION=${VERSION_INFO}' -X '${PKG}/common/version.GITCOMMIT=${GIT_SHA_SHORT}'"
GO_LDFLAGS="-ldflags=${VERSION_VAR}"

for cmd_dir in backend/*/cmd/*; do
  bin_name="$(basename "${cmd_dir}")"
  bin_out="${GOBIN}/${bin_name}"
  pushd "${cmd_dir}"
    echo "Building: ${bin_name} > ${bin_out}"
    go build "${GO_LDFLAGS}" -mod=vendor -o "${bin_out}" .
  popd
done