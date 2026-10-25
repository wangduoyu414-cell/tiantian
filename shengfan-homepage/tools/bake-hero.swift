// 首屏烘图：把《京华胜景图》的静止层画成图片，供首屏直接加载（不用在手机上现画几 MB 的矢量）。
// 用法：先开预览服务（python3 tools/serve.py 4350），再在项目根目录运行
//   swift tools/bake-hero.swift [http://localhost:4350]
// 输出 assets/img/hero/{layout}-{season}-{层}.avif（原尺寸）和 .png（一半尺寸，给旧手机兜底），
// 以及 manifest.json（各层位置、纵深、会动的小块、可点的地标）。
// 改了 assets/js/jh/ 里的画法或首屏构图后，重新跑一遍。
import Cocoa
import WebKit
import ImageIO
import UniformTypeIdentifiers

let base = CommandLine.arguments.count > 1 ? CommandLine.arguments[1] : "http://localhost:4350"
let root = FileManager.default.currentDirectoryPath
let outDir = root + "/assets/img/hero"
try? FileManager.default.createDirectory(atPath: outDir, withIntermediateDirectories: true)
let jobs: [(String, String, Double, Double)] = ["desk", "mobile"].flatMap { l in ["spring", "summer", "autumn", "winter"].map { s in (l, s, l == "desk" ? 2.0 : 2.75, 1.0) } }

let app = NSApplication.shared
app.setActivationPolicy(.accessory)
let win = NSWindow(contentRect: NSRect(x: 0, y: 0, width: 800, height: 600), styleMask: [.borderless], backing: .buffered, defer: false)
win.isReleasedWhenClosed = false
win.alphaValue = 0.01
win.ignoresMouseEvents = true
let cfg = WKWebViewConfiguration(); cfg.websiteDataStore = .nonPersistent()
let wv = WKWebView(frame: NSRect(x: 0, y: 0, width: 800, height: 600), configuration: cfg)
win.contentView = wv
win.orderFrontRegardless()

var manifest: [String: Any] = ["seasons": ["spring", "summer", "autumn", "winter"], "layouts": [String: Any]()]
var layouts: [String: [String: Any]] = [:]
var totals: [String: Int] = [:]

// AVIF 用原尺寸（主力）；PNG 只给不认 AVIF 的旧手机兜底，缩到一半，省体积
func writeImages(_ name: String, _ png: Data) -> [String: Int] {
  var sizes: [String: Int] = [:]
  guard let src = CGImageSourceCreateWithData(png as CFData, nil), let img = CGImageSourceCreateImageAtIndex(src, 0, nil) else { return sizes }
  let avifURL = URL(fileURLWithPath: "\(outDir)/\(name).avif")
  if let dst = CGImageDestinationCreateWithURL(avifURL as CFURL, "public.avif" as CFString, 1, nil) {
    CGImageDestinationAddImage(dst, img, [kCGImageDestinationLossyCompressionQuality: 0.72] as CFDictionary)
    CGImageDestinationFinalize(dst)
    sizes["avif"] = (try? Data(contentsOf: avifURL).count) ?? 0
  }
  let w = max(1, img.width / 2), h = max(1, img.height / 2)
  if let ctx = CGContext(data: nil, width: w, height: h, bitsPerComponent: 8, bytesPerRow: 0, space: CGColorSpaceCreateDeviceRGB(), bitmapInfo: CGImageAlphaInfo.premultipliedLast.rawValue) {
    ctx.interpolationQuality = .high
    ctx.draw(img, in: CGRect(x: 0, y: 0, width: w, height: h))
    if let small = ctx.makeImage() {
      let pngURL = URL(fileURLWithPath: "\(outDir)/\(name).png")
      if let dst = CGImageDestinationCreateWithURL(pngURL as CFURL, "public.png" as CFString, 1, nil) { CGImageDestinationAddImage(dst, small, nil); CGImageDestinationFinalize(dst) }
      sizes["png"] = (try? Data(contentsOf: pngURL).count) ?? 0
    }
  }
  return sizes
}

final class Runner: NSObject, WKNavigationDelegate {
  var i = 0
  func start() { load() }
  func load() {
    guard i < jobs.count else { finish(); return }
    wv.load(URLRequest(url: URL(string: "\(base)/design/v4/bake.html?r=\(Int(Date().timeIntervalSince1970))")!))
  }
  func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
    let (layout, season, scale, px) = jobs[i]
    let t0 = Date()
    webView.callAsyncJavaScript("return await bakeAll(layout, season, scale, px)", arguments: ["layout": layout, "season": season, "scale": scale, "px": px], in: nil, in: .page) { result in
      switch result {
      case .failure(let e): print("失败", layout, season, e); exit(1)
      case .success(let v):
        guard let s = v as? String, let d = s.data(using: .utf8), let o = try? JSONSerialization.jsonObject(with: d) as? [String: Any] else { print("结果不对", layout, season); exit(1) }
        var planesOut: [[String: Any]] = []
        for p in (o["planes"] as? [[String: Any]]) ?? [] {
          let id = p["id"] as! String, url = p["png"] as! String
          let b64 = url.components(separatedBy: ",").last ?? ""
          guard let png = Data(base64Encoded: b64) else { continue }
          let sizes = writeImages("\(layout)-\(season)-\(id)", png)
          totals["\(layout)-\(season)-png", default: 0] += sizes["png"] ?? 0
          totals["\(layout)-\(season)-avif", default: 0] += sizes["avif"] ?? 0
          var q = p; q.removeValue(forKey: "png"); planesOut.append(q)
        }
        if layouts[layout] == nil || season == "summer" {
          var l: [String: Any] = ["W": o["W"]!, "H": o["H"]!, "scale": scale, "planes": planesOut, "hits": o["hits"]!]
          if season == "summer" { l["sprites"] = o["sprites"]! } else if let prev = layouts[layout]?["sprites"] { l["sprites"] = prev } else { l["sprites"] = o["sprites"]! }
          layouts[layout] = l
        }
        manifest["defs"] = o["defs"]
        print(String(format: "%@ %@ %.1fs png %dK avif %dK", layout, season, Date().timeIntervalSince(t0), (totals["\(layout)-\(season)-png"] ?? 0) / 1024, (totals["\(layout)-\(season)-avif"] ?? 0) / 1024))
        self.i += 1; self.load()
      }
    }
  }
  func finish() {
    manifest["layouts"] = layouts
    manifest["version"] = Int(Date().timeIntervalSince1970)
    if let d = try? JSONSerialization.data(withJSONObject: manifest, options: [.sortedKeys]) { try? d.write(to: URL(fileURLWithPath: "\(outDir)/manifest.json")) }
    print("完成：\(outDir)")
    exit(0)
  }
}
let runner = Runner()
wv.navigationDelegate = runner
runner.start()
DispatchQueue.main.asyncAfter(deadline: .now() + 600) { print("超时"); exit(1) }
app.run()
