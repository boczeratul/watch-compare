#!/bin/sh
# Captures App Store screenshots for every supported locale on a 6.9" iPhone simulator by running
# the WatchCompareUITests/ScreenshotTests UI test against the live API.
#
#   ios/scripts/screenshots.sh [output-dir]        # default: ios/Screenshots/<device>
#   DEVICE=ipad ios/scripts/screenshots.sh         # 13" iPad (2064×2752) instead of the 6.9" iPhone
#
# Requires Xcode with an iOS simulator runtime (xcodebuild -downloadPlatform iOS). Set XCODE_DEV to
# use a specific Xcode, e.g. XCODE_DEV=/Applications/Xcode.app/Contents/Developer.
set -eu
cd "$(dirname "$0")/.."
DEVICE="${DEVICE:-iphone}"
OUT="${1:-$PWD/Screenshots/$DEVICE}"
DEV="${XCODE_DEV:-$(xcode-select -p)}"
XCODEBUILD="$DEV/usr/bin/xcodebuild"
SIMCTL="$DEV/usr/bin/simctl"
# Device-type identifiers vary between Xcode releases (…-M4, …-M4-8GB, …-M5), so pick the first
# matching one installed. Override with DEVICE_TYPE=<identifier> if you need a specific model.
pick_type() { "$SIMCTL" list devicetypes | grep -o "com\.apple\.CoreSimulator\.SimDeviceType\.$1[A-Za-z0-9-]*" | head -1; }
case "$DEVICE" in
  iphone) DEVICE_TYPE="${DEVICE_TYPE:-$(pick_type iPhone-17-Pro-Max)}" ;;  # 6.9" → 1320×2868
  ipad)   DEVICE_TYPE="${DEVICE_TYPE:-$(pick_type iPad-Pro-13-inch)}" ;;   # 13"  → 2064×2752
  *) echo "DEVICE must be iphone or ipad" >&2; exit 1 ;;
esac
[ -n "$DEVICE_TYPE" ] || { echo "No simulator device type found for $DEVICE" >&2; exit 1; }
NAME="WatchCompare Screenshots ($DEVICE)"
export DEVELOPER_DIR="$DEV"

RUNTIME=$("$SIMCTL" list runtimes | grep -o 'com\.apple\.CoreSimulator\.SimRuntime\.iOS-[0-9-]*' | sort -V | tail -1)
[ -n "$RUNTIME" ] || { echo "No iOS simulator runtime installed; run: xcodebuild -downloadPlatform iOS" >&2; exit 1; }
UDID=$("$SIMCTL" list devices | grep "^ *$NAME (" | grep -o '[0-9A-F]\{8\}-[0-9A-F-]\{27\}' | head -1 || true)
if [ -z "$UDID" ]; then
  UDID=$("$SIMCTL" create "$NAME" "$DEVICE_TYPE" "$RUNTIME")
fi
"$SIMCTL" boot "$UDID" 2>/dev/null || true
"$SIMCTL" bootstatus "$UDID" -b >/dev/null
# App Store-style status bar: 9:41, full battery and signal.
"$SIMCTL" status_bar "$UDID" override --time 9:41 --batteryState charged --batteryLevel 100 --wifiBars 3 --cellularBars 4 --operatorName ""
"$SIMCTL" ui "$UDID" appearance light

mkdir -p "$OUT"
# LOCALES="zh-Hans de" limits the run to some of them.
for spec in "en en US USD" "zh-Hant zh-Hant TW TWD" "zh-Hans zh-Hans CN CNY" "ja ja JP JPY" "de de DE EUR"; do
  case " ${LOCALES:-en zh-Hant zh-Hans ja de} " in *" ${spec%% *} "*) ;; *) continue ;; esac
  set -- $spec
  locale=$1; language=$2; region=$3; currency=$4
  echo "==> $locale"
  # TEST_RUNNER_* must be environment variables of xcodebuild (not KEY=VALUE arguments, which
  # would become build settings); xcodebuild strips the prefix and passes them to the test runner.
  env TEST_RUNNER_SCREENSHOT_DIR="$OUT" TEST_RUNNER_SCREENSHOT_LOCALE="$locale" \
      TEST_RUNNER_SCREENSHOT_LANGUAGE="$language" TEST_RUNNER_SCREENSHOT_REGION="$region" \
      TEST_RUNNER_SCREENSHOT_CURRENCY="$currency" \
  "$XCODEBUILD" test -project WatchCompare.xcodeproj -scheme WatchCompare -only-testing:WatchCompareUITests \
    -destination "id=$UDID" -derivedDataPath "${DERIVED_DATA:-/tmp/watchcompare-dd}" \
    CODE_SIGNING_ALLOWED=NO -quiet
  test -f "$OUT/$locale/01-home.png" || { echo "no screenshots written for $locale" >&2; exit 1; }
done
"$SIMCTL" status_bar "$UDID" clear
echo "Screenshots written to $OUT"
find "$OUT" -name '*.png' | sort
