// Renders the WatchCompare app icon (1024×1024, opaque, as App Store Connect requires) with
// CoreGraphics: an emerald ground (the web's brand colour, tailwind emerald-700) and a white watch
// glyph whose hands sit at 10:10. Run: swift ios/scripts/make-icon.swift <output.png> [size]
import CoreGraphics
import Foundation
import ImageIO
import UniformTypeIdentifiers

let out = CommandLine.arguments.count > 1 ? CommandLine.arguments[1] : "AppIcon.png"
let size = CommandLine.arguments.count > 2 ? Int(CommandLine.arguments[2])! : 1024
let s = CGFloat(size)

// Opaque RGB bitmap (no alpha channel in the output).
let ctx = CGContext(data: nil, width: size, height: size, bitsPerComponent: 8, bytesPerRow: 0,
                    space: CGColorSpace(name: CGColorSpace.sRGB)!,
                    bitmapInfo: CGImageAlphaInfo.noneSkipLast.rawValue)!
ctx.setAllowsAntialiasing(true)
ctx.setShouldAntialias(true)
let u = s / 1024 // design units → pixels

// Background: vertical gradient emerald-600 → emerald-800. Apple masks the corners; keep it square.
let top = CGColor(red: 0.02, green: 0.55, blue: 0.40, alpha: 1)      // #058c66
let bottom = CGColor(red: 0.02, green: 0.30, blue: 0.23, alpha: 1)   // #054d3b
let grad = CGGradient(colorsSpace: CGColorSpaceCreateDeviceRGB(), colors: [top, bottom] as CFArray, locations: [0, 1])!
ctx.drawLinearGradient(grad, start: CGPoint(x: 0, y: s), end: CGPoint(x: 0, y: 0), options: [])

// Subtle highlight disc behind the watch.
ctx.setFillColor(CGColor(gray: 1, alpha: 0.06))
ctx.fillEllipse(in: CGRect(x: 132 * u, y: 132 * u, width: 760 * u, height: 760 * u))

let c = CGPoint(x: 512 * u, y: 512 * u)
let white = CGColor(gray: 1, alpha: 1)
let ink = bottom

// Strap lugs (top and bottom bars).
ctx.setFillColor(white)
for y in [792.0, 148.0] {
    let r = CGRect(x: 392 * u, y: (y - 30) * u, width: 240 * u, height: 84 * u)
    ctx.addPath(CGPath(roundedRect: r, cornerWidth: 24 * u, cornerHeight: 24 * u, transform: nil))
}
ctx.fillPath()

// Crown.
let crown = CGRect(x: 812 * u, y: 484 * u, width: 56 * u, height: 56 * u)
ctx.addPath(CGPath(roundedRect: crown, cornerWidth: 12 * u, cornerHeight: 12 * u, transform: nil))
ctx.fillPath()

// Case, bezel, dial.
ctx.setFillColor(white)
ctx.fillEllipse(in: CGRect(x: c.x - 320 * u, y: c.y - 320 * u, width: 640 * u, height: 640 * u))
ctx.setFillColor(ink)
ctx.fillEllipse(in: CGRect(x: c.x - 268 * u, y: c.y - 268 * u, width: 536 * u, height: 536 * u))
ctx.setFillColor(white)
ctx.fillEllipse(in: CGRect(x: c.x - 244 * u, y: c.y - 244 * u, width: 488 * u, height: 488 * u))

// Hour markers: batons at 12/3/6/9, dots elsewhere.
ctx.setFillColor(ink)
for i in 0..<12 {
    let a = CGFloat(i) / 12 * 2 * .pi
    let dir = CGPoint(x: sin(a), y: cos(a))
    if i % 3 == 0 {
        let len: CGFloat = 54 * u, w: CGFloat = 22 * u
        let p0 = CGPoint(x: c.x + dir.x * (214 * u), y: c.y + dir.y * (214 * u))
        let p1 = CGPoint(x: c.x + dir.x * (214 * u - len), y: c.y + dir.y * (214 * u - len))
        ctx.setLineWidth(w); ctx.setLineCap(.round); ctx.setStrokeColor(ink)
        ctx.move(to: p0); ctx.addLine(to: p1); ctx.strokePath()
    } else {
        let p = CGPoint(x: c.x + dir.x * (200 * u), y: c.y + dir.y * (200 * u))
        ctx.fillEllipse(in: CGRect(x: p.x - 12 * u, y: p.y - 12 * u, width: 24 * u, height: 24 * u))
    }
}

// Hands at 10:10 (hour hand toward 10, minute toward 2).
func hand(angleDeg: CGFloat, length: CGFloat, width: CGFloat) {
    let a = angleDeg * .pi / 180
    let dir = CGPoint(x: sin(a), y: cos(a))
    ctx.setLineWidth(width); ctx.setLineCap(.round); ctx.setStrokeColor(ink)
    ctx.move(to: CGPoint(x: c.x - dir.x * 30 * u, y: c.y - dir.y * 30 * u))
    ctx.addLine(to: CGPoint(x: c.x + dir.x * length, y: c.y + dir.y * length))
    ctx.strokePath()
}
hand(angleDeg: -60, length: 130 * u, width: 34 * u)   // hour → 10
hand(angleDeg: 60, length: 186 * u, width: 26 * u)    // minute → 2
ctx.setFillColor(ink)
ctx.fillEllipse(in: CGRect(x: c.x - 26 * u, y: c.y - 26 * u, width: 52 * u, height: 52 * u))
ctx.setFillColor(white)
ctx.fillEllipse(in: CGRect(x: c.x - 10 * u, y: c.y - 10 * u, width: 20 * u, height: 20 * u))

let image = ctx.makeImage()!
let dest = CGImageDestinationCreateWithURL(URL(fileURLWithPath: out) as CFURL, UTType.png.identifier as CFString, 1, nil)!
CGImageDestinationAddImage(dest, image, nil)
guard CGImageDestinationFinalize(dest) else { fatalError("could not write \(out)") }
print("wrote \(out) \(size)x\(size)")
