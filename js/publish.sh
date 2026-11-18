#!/bin/sh
# Publish the npm package at the version in js/package.json, once that version is tagged:
#
#   js/publish.sh v1.2.3 [--dry-run]
#
# Refuses a version that doesn't match package.json, runs the tests on a fresh build, then
# publishes. CI runs it on a tag (.github/workflows/npm.yml); run by hand, npm asks for 2FA.
set -eu
VERSION=${1:?usage: js/publish.sh vX.Y.Z [--dry-run]}
shift
cd "$(dirname "$0")"
HAVE=v$(node -p "require('./package.json').version")
[ "$HAVE" = "$VERSION" ] || { echo "package.json is $HAVE, not $VERSION" >&2; exit 1; }
npm ci
npm test
npm publish "$@"
