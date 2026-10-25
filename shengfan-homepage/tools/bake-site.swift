// 各章烘图：把红色教育、团建、会务研学的主画面，会务的隔扇门，导览图底图，各章场景小图画成图片，
// 网页上直接加载，不用在手机上现画几百 KB 的矢量。
// 用法：先开预览服务（python3 tools/serve.py 4350），再在项目根目录运行
//   swift tools/bake-site.swift [http://localhost:4350] [只烘某几项，逗号分隔]
// 输出 assets/img/site/{名}.avif，外加 .jpg（不透明的图）或 .png（透明的小图标），给不认 AVIF 的旧手机兜底；
// 以及 assets/img/site/manifest.json（每张图的尺寸、画面里可点的地方和题签位置）。
// 改了 assets/js/jh/ 里的画法或构图后，重新跑一遍。
import Cocoa
import WebKit
import ImageIO

let base = CommandLine.arguments.count > 1 ? CommandLine.arguments[1] : "http://localhost:4350"
let only = CommandLine.arguments.count > 2 ? Set(CommandLine.arguments[2].split(separator: ",").map(String.init)) : nil
let root = FileManager.default.currentDirectoryPath
let outDir = root + "/assets/img/site"
try? FileManager.default.createDirectory(atPath: outDir, withIntermediateDirectories: true)
let manifestURL = URL(fileURLWithPath: "\(outDir)/manifest.json")
var items: [String: Any] = (try? JSONSerialization.jsonObject(with: Data(contentsOf: manifestURL)) as? [String: Any])?["items"] as? [String: Any] ?? [:]

let app = NSApplication.shared
app.setActivationPolicy(.accessory)
let win = NSWindow(contentRect: NSRect(x: 0, y: 0, width: 800, height: 600), styleMask: [.borderless], backing: .buffered, defer: false)
win.isReleasedWhenClosed = false
win.alphaValue = 0.01
win.ignoresMouseEvents = true
let cfg = WKWebViewConfiguration(); cfg.websiteDataStore = .nonPersistent()
let wv = WKWebView(frame: NSRect(x: 0, y: 0, width: 800, height: 600), configuration: cfg)
if wv.responds(to: NSSelectorFromString("_setWindowOcclusionDetectionEnabled:")) { wv.setValue(false, forKey: "windowOcclusionDetectionEnabled") }
win.contentView = wv
win.orderFrontRegardless()

func write(_ img: CGImage, _ path: String, _ type: String, _ q: Double?) -> Int {
  let url = URL(fileURLWithPath: path)
  guard let dst = CGImageDestinationCreateWithURL(url as CFURL, type as CFString, 1, nil) else { return 0 }
  CGImageDestinationAddImage(dst, img, q.map { [kCGImageDestinationLossyCompressionQuality: $0] as CFDictionary })
  CGImageDestinationFinalize(dst)
  return (try? Data(contentsOf: url).count) ?? 0
}

final class Runner: NSObject, WKNavigationDelegate {
  var names: [String] = [], i = 0, total = 0
  func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
    webView.evaluateJavaScript("listItems()") { v, _ in
      self.names = ((v as? [String]) ?? []).filter { only == nil || only!.contains($0) }
      self.next()
    }
  }
  func next() {
    guard i < names.count else { finish(); return }
    let name = names[i], t0 = Date()
    wv.callAsyncJavaScript("return await bakeItem(name)", arguments: ["name": name], in: nil, in: .page) { result in
      guard case .success(let v) = result, let s = v as? String, let d = s.data(using: .utf8), let o = try? JSONSerialization.jsonObject(with: d) as? [String: Any],
            let url = o["png"] as? String, let png = Data(base64Encoded: url.components(separatedBy: ",").last ?? ""),
            let src = CGImageSourceCreateWithData(png as CFData, nil), let img = CGImageSourceCreateImageAtIndex(src, 0, nil) else { print("失败", name, result); exit(1) }
      let alpha = (o["alpha"] as? Bool) ?? false
      let a = write(img, "\(outDir)/\(name).avif", "public.avif", 0.72)
      let b = alpha ? write(img, "\(outDir)/\(name).png", "public.png", nil) : write(img, "\(outDir)/\(name).jpg", "public.jpeg", 0.8)
      var entry: [String: Any] = ["w": o["w"]!, "h": o["h"]!, "alpha": alpha]
      if let m = o["meta"], !(m is NSNull) { entry["meta"] = m }
      items[name] = entry
      self.total += a
      print(String(format: "%-16@ %5dx%-5d %.1fs  avif %4dK  %@ %4dK", name, img.width, img.height, Date().timeIntervalSince(t0), a / 1024, alpha ? "png" : "jpg", b / 1024))
      self.i += 1; self.next()
    }
  }
  func finish() {
    let manifest: [String: Any] = ["version": Int(Date().timeIntervalSince1970), "items": items]
    if let d = try? JSONSerialization.data(withJSONObject: manifest, options: [.sortedKeys]) { try? d.write(to: manifestURL) }
    print("完成：\(names.count) 张，avif 共 \(total / 1024)K → \(outDir)")
    exit(0)
  }
}
let runner = Runner()
wv.navigationDelegate = runner
wv.load(URLRequest(url: URL(string: "\(base)/design/v4/bake-site.html?r=\(Int(Date().timeIntervalSince1970))")!))
DispatchQueue.main.asyncAfter(deadline: .now() + 900) { print("超时"); exit(1) }
app.run()
