// swift-tools-version: 6.0
import PackageDescription

// Everything except the @main entry point lives in this package so the models, API client and
// views can be built and unit-tested with `swift build` / `swift test` on macOS (no Xcode needed);
// the iOS app target in ../WatchCompare.xcodeproj is a thin shell around it.
let package = Package(
    name: "WatchCompareKit",
    defaultLocalization: "en",
    platforms: [.iOS(.v17), .macOS(.v14)],
    products: [
        .library(name: "WatchCompareKit", targets: ["WatchCompareKit"]),
    ],
    targets: [
        .target(
            name: "WatchCompareKit",
            resources: [.process("Resources")]
        ),
        .testTarget(
            name: "WatchCompareKitTests",
            dependencies: ["WatchCompareKit"]
        ),
    ]
)
