#!/bin/sh
set -eu
umask 077

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd -P)
output=${1:-"$root/dist/release"}
case "$output" in /*) ;; *) output="$root/$output" ;; esac
version=${COUNTERSHAPE_VERSION:-0.9.0-local}
commit=$(/usr/bin/git -C "$root" --no-replace-objects rev-parse --verify 'HEAD^{commit}')
short=$(printf '%s' "$commit" | cut -c1-12)
package="$output/countershape-darwin-arm64"

if [ "$(/usr/bin/uname -s)" != Darwin ] || [ "$(/usr/bin/uname -m)" != arm64 ]; then
  echo "build-release: exact native profile requires Darwin arm64" >&2
  exit 1
fi
if [ -e "$output" ]; then
  echo "build-release: output already exists" >&2
  exit 1
fi
mkdir -m 700 -p "$package"

(cd "$root/web" && /usr/bin/env -i HOME="$HOME" PATH=/opt/homebrew/bin:/usr/bin:/bin NO_COLOR=1 npm run build) >&2
(cd "$root" && /usr/bin/env -i HOME="$HOME" PATH=/opt/homebrew/bin:/usr/bin:/bin GOENV=off GOWORK=off GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off CGO_ENABLED=1 /opt/homebrew/bin/go build -mod=readonly -buildvcs=false -trimpath -ldflags "-buildid= -X main.version=$version -X main.commit=$short" -o "$package/countershape" ./cmd/countershape)
chmod 700 "$package/countershape"
cp "$root/LICENSE" "$package/LICENSE"
chmod 600 "$package/LICENSE"

binary_sha=$(/usr/bin/shasum -a 256 "$package/countershape" | /usr/bin/awk '{print $1}')
binary_bytes=$(/usr/bin/stat -f '%z' "$package/countershape")
cat >"$package/manifest.json" <<EOF
{"schema_version":"countershape/release-manifest/v1","version":"$version","commit":"$commit","platform":"darwin","architecture":"arm64","binary":{"path":"countershape","bytes":$binary_bytes,"sha256":"$binary_sha"},"license":"Apache-2.0","signing":"UNSIGNED_LOCAL_REFERENCE_PACKAGE","distribution":"LOCAL_ONLY_NOT_PUBLISHED"}
EOF
chmod 600 "$package/manifest.json"

if /usr/bin/strings "$package/countershape" | /usr/bin/grep -E "$HOME|\.didrun/objects/|node_modules/|/@vite/client|/@react-refresh" >/dev/null; then
  echo "build-release: forbidden build/private marker found in binary" >&2
  exit 1
fi
echo "$package"
