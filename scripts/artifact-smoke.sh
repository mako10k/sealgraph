#!/bin/sh
set -eu

smoke_archive=${1:?usage: artifact-smoke.sh ARCHIVE VERSION}
smoke_version=${2:?usage: artifact-smoke.sh ARCHIVE VERSION}
smoke_tmp=$(mktemp -d "${TMPDIR:-/tmp}/sealgraph-smoke.XXXXXX")
trap 'rm -rf "$smoke_tmp"' EXIT HUP INT TERM
tar -xzf "$smoke_archive" -C "$smoke_tmp"
smoke_bin="$smoke_tmp/sealgraph_${smoke_version}_linux_amd64/sealgraph"
test "$("$smoke_bin" --version)" = "sealgraph $smoke_version"

mkdir "$smoke_tmp/repository"
cd "$smoke_tmp/repository"
"$smoke_bin" init >/dev/null
"$smoke_bin" add root --root --content root >/dev/null
root_seal=$("$smoke_bin" seal root | awk '{print $3}')
"$smoke_bin" add dependent --content dependent --depend-on root >/dev/null
"$smoke_bin" seal dependent >/dev/null
"$smoke_bin" add root --root --content root-v2 >/dev/null
"$smoke_bin" seal root >/dev/null
test "$("$smoke_bin" stale --frontier --refs-only)" = "dependent"
"$smoke_bin" fsck --format json | grep -q '"schema":"sealgraph/fsck/v1"'

object_path=".sealgraph/objects/$(printf '%s' "$root_seal" | cut -c1-2)/$(printf '%s' "$root_seal" | cut -c3-)"
chmod u+w "$object_path"
printf corrupt > "$object_path"
if "$smoke_bin" fsck >/dev/null 2>&1; then
  echo "artifact-smoke: corrupted object was accepted" >&2
  exit 1
fi
