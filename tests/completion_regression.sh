#!/usr/bin/env bash

set -euo pipefail

bin_path="${1:-./bin/sealgraph}"
source_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
completion_script="$source_root/completions/sealgraph.bash"

top_level="$($bin_path __completion --bash '')"
grep -qx '__sealgraph_completion_mode=plain' <<<"$top_level"
grep -qx 'compare' <<<"$top_level"
if grep -Eq '^(diff|rm)$' <<<"$top_level"; then
    printf 'Git-shaped vocabulary leaked into completion\n' >&2
    exit 1
fi

SEALGRAPH_COMPLETION_BIN="$bin_path"
export SEALGRAPH_COMPLETION_BIN
source "$completion_script"
COMP_WORDS=(sealgraph comp)
COMP_CWORD=1
_sealgraph_complete
if [[ ! " ${COMPREPLY[*]} " =~ " compare " ]]; then
    printf 'Bash wrapper did not complete compare: %s\n' "${COMPREPLY[*]}" >&2
    exit 1
fi
