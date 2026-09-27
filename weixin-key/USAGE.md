# weixin-key 使用说明

> 本目录是 weixin-key 项目的完整备份(2026-09-27,对应主仓库 commit c7613b6)。
> 主仓库:https://github.com/wangduoyu414-cell/weixin-key

## 这是什么

微信(Weixin 4.1+,Windows)本地聊天记录导出工具。**仅限本人账号、本人电脑**。

能从一台没装过任何开发环境的新电脑上,把本机微信的聊天记录完整导出:
文字、图片(含加密 .dat 解密)、语音、视频、文件,带发送者、方向、会话名、
时间,输出为可读的 Markdown + 结构化 JSONL + 归位好的附件。

已验证版本:4.1.13.65、4.1.15.13(2026-09 官网最新版)。

## 目录说明

```
weixin-key/
├─ dist/weixin-key-portable/   ★ 便携包:三个 exe + 说明,拷走即用,无需安装
├─ skills/wechat-export/       ★ Agent 操作手册(SKILL.md 决策树 + 包装脚本)
├─ strategies/                 版本策略库(按 Weixin.dll 哈希记录已验证版本)
├─ scripts/                    observe.ps1(新机观测)、build-dist.ps1(打包)
├─ cmd/ internal/              Go 源码
└─ docs/                       技术文档(取钥原理、实施状态等)
```

## 新电脑怎么用(三步)

**前提**:新机装了微信 4.1+,已登录本人账号,正常用过(能看聊天记录和图片)。

**第 1 步**:把 `dist/weixin-key-portable` 拷到新机。

**第 2 步**:在便携包目录里跑观测:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File observe.ps1
```

输出现场参数(JSON)。多账号机器会列出候选,先确认要导出哪个。

**第 3 步**:任选一种方式执行导出:

- **有 Agent(推荐)**:让 Agent 读 `skills/wechat-export/SKILL.md`,把第 2 步的
  JSON 给它,按决策树自动完成取钥→验证→导出→验收。
- **手动**:SKILL.md 里每一步都有完整的可直接复制的命令,照着改路径执行。

导出结果在 `--out` 指定的目录:`index.md` 是总目录,`accounts/` 下按会话
分文件夹直接阅读,`data/` 是结构化数据,`attachments/` 是解密好的附件。

## 常见问题

| 情况 | 处理 |
|---|---|
| 取钥 0 命中(微信版本太新) | 把 observe.ps1 输出带回主仓库做版本适配;工具不会假装成功 |
| 图片密钥扫不到 | 在微信里点开几张聊天图片,再扫一次(密钥按需加载进内存) |
| 部分附件 missing | 微信自己清理了源文件(过期/没下载过),任何工具都无法恢复 |
| 大图是 .wxgf 打不开 | 微信私有格式,用 ffmpeg(带 wxgf 支持)转 jpg;缩略图 _t.jpg 都能直接看 |
| 语音 .silk 打不开 | 用 silk-v3-decoder 之类工具转 mp3/wav |

## 安全红线

- 只在**本人账号、本人电脑**上用。
- 密钥、聊天内容不提交、不上传、不分享。本目录不含任何密钥。
- 密钥缓存绑定具体某台电脑的 Windows 用户(DPAPI),不能拷走;换电脑重新取钥即可。

## License

Private. All rights reserved.
