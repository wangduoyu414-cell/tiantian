#!/usr/bin/env python3
# 局域网预览：只对外提供 index.html、assets/ 和样张目录下的文件，design/ 其余部分、tools/ 等一律 404。
# 用法：python3 tools/serve.py [端口]，默认 4340。手机和电脑连同一个 Wi-Fi（或电脑连着手机热点），
# 打开启动时打印的"手机"地址即可。
import http.server
import os
import subprocess
import sys
from functools import partial

ROOT = os.path.dirname(os.path.dirname(os.path.realpath(__file__)))
INDEX = os.path.join(ROOT, 'index.html')
ALLOWED = tuple(os.path.join(ROOT, *d.split('/')) + os.sep for d in ('assets', 'design/concept-v2'))
PORT = int(sys.argv[1]) if len(sys.argv) > 1 else 4340


class SiteOnly(http.server.SimpleHTTPRequestHandler):
    extensions_map = {**http.server.SimpleHTTPRequestHandler.extensions_map, '.woff2': 'font/woff2'}

    def send_head(self):
        # translate_path 已经把 ../ 和 %2e%2e 规整掉，这里按真实路径判断是否在白名单内
        p = os.path.realpath(self.translate_path(self.path))
        if os.path.isdir(p):
            p = os.path.join(p, 'index.html')
        if p != INDEX and not (p.startswith(ALLOWED) and os.path.isfile(p)):
            self.send_error(404)
            return None
        return super().send_head()

    def end_headers(self):
        # 微信内置浏览器缓存很重；不让它缓存，改完文件手机上刷新就能看到
        self.send_header('Cache-Control', 'no-store')
        super().end_headers()


def lan_ips():
    # 只取 en* 网卡（Wi-Fi / 有线）的 IPv4。代理软件开 TUN 模式时，默认出口会变成 198.18.x 这类虚拟网卡，手机连不上
    try:
        out = subprocess.run(['ifconfig'], capture_output=True, text=True).stdout
    except OSError:
        out = ''
    ips, iface = [], ''
    for line in out.splitlines():
        if line and not line[0].isspace():
            iface = line.split(':', 1)[0]
        elif line.strip().startswith('inet ') and iface.startswith('en'):
            ip = line.split()[1]
            if not ip.startswith(('127.', '169.254.')):
                ips.append(ip)
    return ips or ['127.0.0.1']


if __name__ == '__main__':
    server = http.server.ThreadingHTTPServer(('0.0.0.0', PORT), partial(SiteOnly, directory=ROOT))
    print(f'本机：http://localhost:{PORT}/')
    for ip in lan_ips():
        print(f'手机：http://{ip}:{PORT}/')
    sys.stdout.flush()
    server.serve_forever()
