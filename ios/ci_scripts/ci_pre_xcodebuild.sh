#!/bin/sh
# Xcode Cloud runs this before xcodebuild (it looks for ci_scripts/ next to the .xcodeproj).
# Every TestFlight upload needs a unique build number, so stamp CURRENT_PROJECT_VERSION with the
# monotonically increasing Xcode Cloud build number. MARKETING_VERSION (1.0, 1.1…) stays manual.
set -eu
if [ -n "${CI_BUILD_NUMBER:-}" ]; then
  cd "$CI_PRIMARY_REPOSITORY_PATH/ios"
  xcrun agvtool new-version -all "$CI_BUILD_NUMBER"
  echo "Build number set to $CI_BUILD_NUMBER"
fi
