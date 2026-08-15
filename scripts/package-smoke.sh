#!/bin/sh
set -eu
umask 077

binary=${1:?usage: package-smoke.sh ABSOLUTE_BINARY}
case "$binary" in /*) ;; *) echo "package-smoke: binary must be absolute" >&2; exit 2 ;; esac
[ -x "$binary" ] || { echo "package-smoke: binary is not executable" >&2; exit 2; }
root=$(mktemp -d /private/tmp/countershape-package-smoke.XXXXXX)
chmod 700 "$root"
server_pid=
cleanup() {
  if [ -n "$server_pid" ]; then kill -TERM "$server_pid" 2>/dev/null || true; wait "$server_pid" 2>/dev/null || true; fi
  rm -rf "$root"
}
trap cleanup EXIT INT TERM HUP

mkdir -m 700 "$root/home" "$root/tmp" "$root/work"
/usr/bin/git -C "$root/work" init -q
HOME="$root/home" TMPDIR="$root/tmp" "$binary" --help >"$root/help.txt"
HOME="$root/home" TMPDIR="$root/tmp" "$binary" version >"$root/version.txt"
HOME="$root/home" TMPDIR="$root/tmp" "$binary" export --output "$root/report.html" --acknowledge-confidentiality-not-established >"$root/export.txt"
[ "$(/usr/bin/stat -f '%Lp' "$root/report.html")" = 600 ]
/usr/bin/grep -F 'CONFIDENTIALITY NOT ESTABLISHED' "$root/report.html" >/dev/null
if HOME="$root/home" TMPDIR="$root/tmp" "$binary" export --output "$root/report.html" --acknowledge-confidentiality-not-established >/dev/null 2>&1; then
  echo "package-smoke: overwrite was accepted" >&2; exit 1
fi

launch="$root/launch.json"
COUNTERSHAPE_STUDIO_TEST_LAUNCH=1 HOME="$root/home" TMPDIR="$root/tmp" "$binary" studio --no-open --seed-state decision-ready --launch-file "$launch" >"$root/studio.out" 2>"$root/studio.err" &
server_pid=$!
i=0
while [ ! -s "$launch" ] && [ "$i" -lt 100 ]; do sleep 0.05; i=$((i+1)); done
[ -s "$launch" ] || { echo "package-smoke: studio did not publish launch authority" >&2; exit 1; }
origin=$(/opt/homebrew/bin/node -e 'const fs=require("fs");const v=JSON.parse(fs.readFileSync(process.argv[1],"utf8"));process.stdout.write(v.origin)' "$launch")
url=$(/opt/homebrew/bin/node -e 'const fs=require("fs");const v=JSON.parse(fs.readFileSync(process.argv[1],"utf8"));process.stdout.write(v.url)' "$launch")
token=${url##*#access_token=}
case "$origin" in http://127.0.0.1:*) ;; *) echo "package-smoke: non-loopback origin" >&2; exit 1 ;; esac
/usr/bin/curl -fsS -H "Authorization: Bearer $token" "$origin/" >"$root/index.html"
/usr/bin/grep -F '/assets/studio.css' "$root/index.html" >/dev/null
/usr/bin/grep -F '/assets/studio.js' "$root/index.html" >/dev/null
kill -TERM "$server_pid"
wait "$server_pid"
server_pid=

if find "$root" -type f -perm +044 -print | grep . >/dev/null; then echo "package-smoke: world/group-readable private output" >&2; exit 1; fi
echo "COUNTERSHAPE_PACKAGE_SMOKE_OK"
