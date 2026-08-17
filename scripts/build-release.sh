#!/bin/sh
set -eu

release_version=${1:?usage: build-release.sh VERSION DIST_DIR}
release_dist=${2:?usage: build-release.sh VERSION DIST_DIR}
case "$release_version" in
  0.1.0-beta.*) ;;
  *) echo "build-release: expected 0.1.0-beta.N, got $release_version" >&2; exit 2 ;;
esac

release_name="sealgraph_${release_version}_linux_amd64"
release_tmp=$(mktemp -d "${TMPDIR:-/tmp}/sealgraph-release.XXXXXX")
trap 'rm -rf "$release_tmp"' EXIT HUP INT TERM
mkdir -p "$release_tmp/$release_name" "$release_dist"

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs=false -trimpath \
  -ldflags "-s -w -X github.com/mako10k/sealgraph/internal/cli.Version=$release_version" \
  -o "$release_tmp/$release_name/sealgraph" ./cmd/sealgraph
cp LICENSE README.md "$release_tmp/$release_name/"

archive="$release_dist/$release_name.tar.gz"
tar --sort=name --owner=0 --group=0 --numeric-owner --mtime=@0 \
  -C "$release_tmp" -cf - "$release_name" | gzip -n > "$archive"
(cd "$release_dist" && sha256sum "$release_name.tar.gz" > "sealgraph_${release_version}_checksums.txt")
