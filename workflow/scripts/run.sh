#!/usr/bin/env bash
# Runs the workflow binary from an Alfred UI script action. These get no
# alfred_workflow_bundleid, which the binary needs to find its cache and data.
set -euo pipefail

wfdir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

: "${alfred_workflow_bundleid:=$(plutil -extract bundleid raw "$wfdir/info.plist")}"
export alfred_workflow_bundleid
export OP_NO_UPDATE=true

cd "$wfdir"
exec "$wfdir/alfred-1password-search" "$@"
