# 贵昌集团 · 晟帆国旅｜个人主页 / 电子名片

纯静态网站：`index.html` + `assets/`，没有构建步骤，手机和微信内置浏览器优先。贵昌集团（政企采购）是主体，晟帆国旅（会务文旅）是"贵昌集团旗下"板块。名片放在页面最后。

当前是 v3。设计语言叫「凭证」：深蓝证书封皮 + 米白纸张 + 烫金。每一章只放一个主视觉，都用真实素材做。

| 章节 | 主视觉 |
|---|---|
| 封面 | 防伪扭索纹、烫金标题、翻牌轮播服务品类 |
| 01 实力 | 翻牌看板 + 点阵中国地图（北京总部连到已服务省市） |
| 02 资质 | 31 张证书随滚动砌成 3D 证书墙；点开是原件查看器，附官方核验入口 |
| 03 业务 | 运单（四大品类实拍）、手机样机（合创慧采真实界面）、登机牌（晟帆五项服务）、集团星图 |
| 04 客户 | 客户层级阶梯、客户铭牌滚动墙、冬奥感谢信、大事记 |
| 05 对接人 | 电子名片：打电话、加微信、存通讯录 |

`design/preview/手机长图-v3.jpg` 是手机端全页效果图。

## 本地预览

```bash
python3 tools/serve.py
```

启动后打开终端里打印的地址（默认 http://localhost:4340/）。同一局域网的手机可以打开打印出来的"手机"地址。

- 这个服务只对外提供 `index.html`、`assets/` 和 `design/concept-v2/`。
- 服务设置了不缓存，改完文件刷新页面就能看到。

## 目录

```
index.html              页面结构（联系信息集中在 #card 的 data-* 属性里）
assets/css/site.css     全部样式
assets/css/fonts.css    本地字体声明（由 tools/fonts.py 生成）
assets/js/site.js       交互与动效：翻牌、点阵地图、证书墙、原件查看器、星图等
assets/js/map-data.js   点阵地图数据（由 PPT 中的地图轮廓采样生成）
assets/fonts/           裁剪后的字体（思源宋体、Barlow Condensed、JetBrains Mono，均为 OFL）
assets/img/cert/        证书图（cert/s/ 是缩略图）
assets/img/letter/      感谢信、表彰信
assets/img/photo/       品类、晟帆服务、公益、十周年等实拍
assets/img/ui/          合创慧采平台界面截图
tools/serve.py          本地 / 局域网预览服务
tools/fonts.py          按 font-text.json 下载裁剪字体
tools/font-text.json    各字体需要包含的字符
tools/collect-serif-chars.js  在浏览器里收集页面用到的宋体字
design/concept-v1/      第一版风格样张
design/concept-v2/      v3 方向样张（资质墙 + 点阵地图）
design/preview/         各版本手机长图与分镜
design/source/          两家 logo 原图
```

## 修改须知

- **联系信息：** 改 `index.html` 里 `#card` 的 `data-*` 属性和卡片上的文字。存通讯录、复制、加微信都从这些属性读取。
- **版本号：** 改了 CSS 或 JS 后，把 `index.html` 里资源链接的 `?v=` 版本号加一，避免访客浏览器读旧缓存。
- **字体：** 改了宋体文字后要重新裁剪字体，否则新字会显示成系统字体。步骤：
  1. 在浏览器控制台运行 `tools/collect-serif-chars.js`。
  2. 把结果填进 `tools/font-text.json`。
  3. 运行 `python3 tools/fonts.py`（需要能访问 Google Fonts）。
- **兼容性：** iPhone 和微信是 WebKit 内核，验收要以 WebKit 为准。翻牌动画没有用 `backface-visibility`，是刻意的：WebKit 会把背面透出来。

## 还是占位的内容

- 个人照片、手机、微信二维码、邮箱、两家公司的职务。
- 20 余个分支机构的城市清单（地图上目前只标了已有服务记录的省市）。
- 证书清晰扫描件。专精特新要续期后的新证书；现在这张取自介绍材料，左下角被遮挡。
- 晟帆的真实项目照片。现在用的是介绍材料里带水印的示意图。
- 客户墙的 logo。现在用的是名称铭牌。
- ICP 备案号。

## 上线前

- 点阵中国地图属于风格化地图，公开发布前要送审拿审图号，或者改用自然资源部标准地图做底图。
- 党政机关只写层级，不列名称、不用标志；部队与涉密单位不展示；带奥运标志的材料不用。
- 冬奥相关表述，以及"24 项资质 / 7 项知识产权"的口径，需要确认后再上线。
