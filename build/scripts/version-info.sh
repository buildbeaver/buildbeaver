#!/bin/bash
set -e
if [ -n "${BB_DEBUG}" ]; then
  set -x
fi

if [ ! "$(which git)" ]; then
  echo "git must be available."
  exit 1
fi

git config --global --add safe.directory $(pwd)
GIT_DESCRIBE="$(git describe --long --tags --always)"
if [[ ! $GIT_DESCRIBE =~ ^v?[0-9+].[0-9+].[0-9+].*$ ]]; then
  echo "Unknown tag format."
  exit 1
fi

# Parse version info from git describe
GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
NORMALIZED_BRANCH=${GIT_BRANCH//-/_}
NORMALIZED_BRANCH=${NORMALIZED_BRANCH//\//_}
VERSION_RAW=$(echo "$GIT_DESCRIBE" | awk '{split($0,a,"-"); print a[1]}')
VERSION_MAJOR=$(echo "$VERSION_RAW" | awk '{split($0,a,"."); print a[1]}')
VERSION_MINOR=$(echo "$VERSION_RAW" | awk '{split($0,a,"."); print a[2]}')
VERSION_PATCH=$(echo "$VERSION_RAW" | awk '{split($0,a,"."); print a[3]}')
VERSION_BUILD=$(echo "$GIT_DESCRIBE" | awk '{split($0,a,"-"); print a[2]}')
VERSION="${VERSION_MAJOR}.${VERSION_MINOR}.${VERSION_PATCH}"

# Grab the git short SHA
GIT_SHA_SHORT=$(git rev-parse --short=12 HEAD)

# Our version string should include the commit count if it is not 0
if [ "${VERSION_BUILD}" != "0" ]; then
  VERSION="${VERSION}.${VERSION_BUILD}"
fi

# Include the branch name unless this is master, head, or a release branch
if [ "${GIT_BRANCH}" != "master" ] && [ "${GIT_BRANCH}" != "HEAD" ] && [[ "${GIT_BRANCH}" != release* ]]; then
  VERSION="${VERSION}.${NORMALIZED_BRANCH}"
fi

# Let callers specify which part they're interested in
case "$1" in
'')
    echo "${VERSION}"
    ;;
'version')
    echo "${VERSION}"
    ;;
'major')
    echo  "${VERSION_MAJOR}"
    ;;
'minor')
    echo  "${VERSION_MINOR}"
    ;;
'patch')
    echo  "${VERSION_PATCH}"
    ;;
'build')
    echo  "${VERSION_BUILD}"
    ;;
'branch')
    echo  "${NORMALIZED_BRANCH}"
    ;;
'sha-short')
    echo  "${GIT_SHA_SHORT}"
    ;;
*)
    echo "Unsupported version part: ${1}"
    exit 1
    ;;
esac