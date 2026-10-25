// 用 WKWebView 按指定视口尺寸渲染页面并截图（不依赖预览面板宽度）
// 用法：websnap <url> <outPrefix> <width> <height> <js-step>...
//   每个 js-step 是一段 JS（比如 "scrollTo(0,1200)"），执行后等 SNAP_WAIT 秒（默认 2.2）再截图，存为 outPrefix-<序号>.png，
//   并打印 "saved <文件> <宽>x<高> js=<这段 JS 的返回值>"。
// 窗口几乎透明、放在桌面层、不接鼠标；并关掉"被遮挡就停渲染"，这样被别的窗口盖住时动画时钟照样走。
import Cocoa
import WebKit

let a = CommandLine.arguments
guard a.count >= 5 else { print("usage: websnap url outPrefix width height [js...]"); exit(2) }
let url = URL(string: a[1])!, out = a[2]
let W = CGFloat(Double(a[3])!), H = CGFloat(Double(a[4])!)
let steps = Array(a.dropFirst(5))
let waitSec = Double(ProcessInfo.processInfo.environment["SNAP_WAIT"] ?? "2.2") ?? 2.2

let app = NSApplication.shared
app.setActivationPolicy(.accessory)
let win = NSWindow(contentRect: NSRect(x: 0, y: 0, width: W, height: H), styleMask: [.borderless], backing: .buffered, defer: false)
win.isReleasedWhenClosed = false
win.alphaValue = 0.01
win.ignoresMouseEvents = true
win.level = .init(rawValue: Int(CGWindowLevelForKey(.desktopWindow)))
let cfg = WKWebViewConfiguration(); cfg.websiteDataStore = .nonPersistent()
let wv = WKWebView(frame: NSRect(x: 0, y: 0, width: W, height: H), configuration: cfg)
// 私有开关：不因窗口被遮挡而暂停渲染（没有这个开关的系统上忽略）
if wv.responds(to: NSSelectorFromString("_setWindowOcclusionDetectionEnabled:")) { wv.setValue(false, forKey: "windowOcclusionDetectionEnabled") }
win.contentView = wv
win.orderFrontRegardless()

final class Runner: NSObject, WKNavigationDelegate {
  var i = 0
  func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
    DispatchQueue.main.asyncAfter(deadline: .now() + 1.5) { self.next() }
  }
  func webView(_ webView: WKWebView, didFail navigation: WKNavigation!, withError error: Error) { print("load failed", error); exit(1) }
  func webView(_ webView: WKWebView, didFailProvisionalNavigation navigation: WKNavigation!, withError error: Error) { print("load failed", error); exit(1) }
  func next() {
    guard i < max(steps.count, 1) else { print("done"); exit(0) }
    let js = steps.isEmpty ? "''" : steps[i]
    wv.evaluateJavaScript(js) { res, err in
      let r = err != nil ? "ERR \(err!.localizedDescription)" : (res.map { "\($0)" } ?? "")
      DispatchQueue.main.asyncAfter(deadline: .now() + waitSec) {
        let sc = WKSnapshotConfiguration(); sc.afterScreenUpdates = true
        wv.takeSnapshot(with: sc) { img, e in
          let name = String(format: "%@-%02d.png", out, self.i)
          if let img = img, let tiff = img.tiffRepresentation, let rep = NSBitmapImageRep(data: tiff), let png = rep.representation(using: .png, properties: [:]) {
            try? png.write(to: URL(fileURLWithPath: name))
            print("saved \(name) \(rep.pixelsWide)x\(rep.pixelsHigh) js=\(r)")
          } else { print("snapshot failed \(e?.localizedDescription ?? "")") }
          self.i += 1
          if self.i >= max(steps.count, 1) { print("done"); exit(0) }
          self.next()
        }
      }
    }
  }
}
let runner = Runner()
wv.navigationDelegate = runner
wv.load(URLRequest(url: url))
DispatchQueue.main.asyncAfter(deadline: .now() + 180) { print("timeout"); exit(1) }
app.run()
