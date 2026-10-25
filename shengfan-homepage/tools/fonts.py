"""按 tools/font-text.json 里列出的字符，从 Google Fonts 下载裁剪过的 woff2，生成 assets/css/fonts.css。

用法：python3 tools/fonts.py
（需要能访问 fonts.googleapis.com；生成的字体文件放在 assets/fonts/，上线后不再依赖 Google。）
"""
import json
import pathlib
import re
import urllib.parse
import urllib.request

ROOT = pathlib.Path(__file__).resolve().parent.parent
UA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36"
spec = json.loads((ROOT / "tools/font-text.json").read_text())


def get(url):
    req = urllib.request.Request(url, headers={"User-Agent": UA})
    with urllib.request.urlopen(req, timeout=30) as r:
        return r.read()


faces = []
for family, weights in spec.items():
    if family.startswith("_"):
        continue
    slug = family.lower().replace(" ", "-")
    for weight, text in weights.items():
        q = urllib.parse.urlencode({"family": f"{family}:wght@{weight}", "text": text, "display": "swap"})
        css = get(f"https://fonts.googleapis.com/css2?{q}").decode()
        m = re.search(r"url\((https://[^)]+)\)\s*format\('woff2'\)", css)
        if not m:
            raise SystemExit(f"没拿到 woff2：{family} {weight}\n{css[:300]}")
        data = get(m.group(1))
        name = f"{slug}-{weight}.woff2"
        (ROOT / "assets/fonts" / name).write_bytes(data)
        print(f"{name:32s} {len(text):4d} 字  {len(data) / 1024:6.1f} KB")
        faces.append(
            f'@font-face {{ font-family: "{family}"; font-style: normal; font-weight: {weight}; '
            f'font-display: swap; src: url(../fonts/{name}) format("woff2"); }}'
        )

(ROOT / "assets/css/fonts.css").write_text(
    "/* 由 tools/fonts.py 生成：只含页面用到的字符，字体文件放在自己服务器上。 */\n" + "\n".join(faces) + "\n"
)
print("已写入 assets/css/fonts.css")
