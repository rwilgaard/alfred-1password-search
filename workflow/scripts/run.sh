#!/usr/bin/env bash
# Runs the workflow binary from an Alfred UI script action, filling in the env awgo needs.
set -euo pipefail

wfdir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

: "${alfred_workflow_bundleid:=$(plutil -extract bundleid raw "$wfdir/info.plist")}"
: "${alfred_workflow_cache:=$HOME/Library/Caches/com.runningwithcrayons.Alfred/Workflow Data/$alfred_workflow_bundleid}"
: "${alfred_workflow_data:=$HOME/Library/Application Support/Alfred/Workflow Data/$alfred_workflow_bundleid}"
export alfred_workflow_bundleid alfred_workflow_cache alfred_workflow_data
export OP_NO_UPDATE=true
mkdir -p "$alfred_workflow_cache" "$alfred_workflow_data"

cd "$wfdir"
exec "$wfdir/alfred-1password-search" "$@"
