#!/bin/bash
set -e
if [ -n "${BB_DEBUG}" ]; then
  set -x
fi

/usr/local/bin/docker-entrypoint.sh generate \
  -i "backend/server/api/rest/openapi/$1" \
  -g "$2" \
  --additional-properties disallowAdditionalPropertiesIfNotPresent=false \
  --additional-properties packageName=client \
  --additional-properties isGoSubmodule=true \
  --additional-properties modelPropertyNaming=snake_case \
  -o "$3"

if [ "$2" = "go" ]; then
  rm "$3"/go.{mod,sum}
  rm -rf "$3/test" # TODO figure out how to generate tests with correct go package path
fi