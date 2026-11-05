#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "usage: $0 <tray|gui> <universal-binary> <version> <favicon.png> <out.dmg>" >&2
  exit 2
}

[ "$#" -eq 5 ] || usage

edition=$1
binary=$2
version=$3
icon=$4
out=$5

here=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
template="$here/Info.plist.in"
min_os=${MACOS_MIN_VERSION:-13.0}

case "$edition" in
  tray)
    app_name="Internet Monitor"
    executable="internet-monitor"
    bundle_id="io.github.khaymah.internetmonitor"
    ui_element="true"
    ;;
  gui)
    app_name="Internet Monitor Window"
    executable="internet-monitor-gui"
    bundle_id="io.github.khaymah.internetmonitor.window"
    ui_element="false"
    ;;
  *) usage ;;
esac

[ -f "$binary" ] || { echo "missing binary: $binary" >&2; exit 1; }
[ -f "$icon" ] || { echo "missing icon: $icon" >&2; exit 1; }
[ -f "$template" ] || { echo "missing template: $template" >&2; exit 1; }

version=${version#v}
if ! [[ "$version" =~ ^[0-9]+(\.[0-9]+){0,2}([-+][0-9A-Za-z.-]+)?$ ]]; then
  echo "unexpected version string: $version" >&2
  exit 1
fi
build=${version%%[-+]*}

archs=$(lipo -archs "$binary")
for want in arm64 x86_64; do
  case " $archs " in
    *" $want "*) ;;
    *) echo "$binary is missing the $want slice (has: $archs)" >&2; exit 1 ;;
  esac
done

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

iconset="$work/AppIcon.iconset"
mkdir -p "$iconset"
for size in 16 32 128 256 512; do
  double=$((size * 2))
  sips -z "$size" "$size" "$icon" --out "$iconset/icon_${size}x${size}.png" >/dev/null
  sips -z "$double" "$double" "$icon" --out "$iconset/icon_${size}x${size}@2x.png" >/dev/null
done
iconutil -c icns "$iconset" -o "$work/AppIcon.icns"

stage="$work/stage"
app="$stage/$app_name.app"
mkdir -p "$app/Contents/MacOS" "$app/Contents/Resources"
cp "$binary" "$app/Contents/MacOS/$executable"
chmod 755 "$app/Contents/MacOS/$executable"
cp "$work/AppIcon.icns" "$app/Contents/Resources/AppIcon.icns"

sed \
  -e "s|@NAME@|$app_name|g" \
  -e "s|@EXECUTABLE@|$executable|g" \
  -e "s|@BUNDLE_ID@|$bundle_id|g" \
  -e "s|@VERSION@|$version|g" \
  -e "s|@BUILD@|$build|g" \
  -e "s|@MIN_OS@|$min_os|g" \
  -e "s|@UI_ELEMENT@|$ui_element|g" \
  "$template" > "$app/Contents/Info.plist"
plutil -lint "$app/Contents/Info.plist"

codesign --force --deep -s - "$app"
codesign --verify --deep --strict "$app"

ln -s /Applications "$stage/Applications"

mkdir -p "$(dirname "$out")"
rm -f "$out"
hdiutil create -volname "$app_name" -srcfolder "$stage" -ov -format UDZO "$out"
hdiutil verify "$out" >/dev/null

echo "built $out ($app_name $version, $archs)"
