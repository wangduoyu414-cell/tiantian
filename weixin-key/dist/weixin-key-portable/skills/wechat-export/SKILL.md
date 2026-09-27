---
name: wechat-export
description: 从本机微信(Weixin 4.1+, Windows)提取并导出本人账号的聊天记录、图片、语音、视频和文件。当用户要求导出/备份微信聊天记录、提取微信数据库、分析聊天数据时使用。仅限用户本人账号、本人电脑。
---

# wechat-export — 本机微信聊天记录导出

## 铁律(不可绕过)

1. **仅限本人账号、本人电脑**。任何他人账号迹象 → 停止。
2. **永不触碰主动 hook/调试/注入路径**,不重启微信,不杀进程。
3. 密钥、passphrase、聊天内容**不进入对话、不提交 git、不上传**。图片密钥只通过环境变量传递。
4. 每个阶段后有验证门;门不过就停下来报告,不糊弄、不降级冒充成功。
5. 用户已有文件**永不覆盖**;输出目录冲突由工具自行另存。

## 决策树

```
开始
 ├─ 工具未构建? → scripts/observe.ps1 前置;需要 Go 1.26.5+ 或用便携包 exe
 ├─ 1. 观测:scripts/observe.ps1
 │     ├─ weixin-not-running → 请用户启动微信并登录,再回来
 │     ├─ multiple-accounts → 列出候选让用户确认,不猜
 │     └─ strategy: unknown-version → 先 material-probe --inspect;
 │        形状兼容可谨慎继续,不兼容则停止并写策略草稿
 ├─ 2. 配置:doctor → init-account(已存在配置则跳过)
 ├─ 3. 材料:material-probe --verify-cache
 │     ├─ 全部命中 → 跳到 5(全程离线,不碰微信)
 │     └─ 未覆盖 → 4. 有界被动取钥(material-probe 实时模式)
 ├─ 4. 取钥后必须 --verify-cache 复核才准继续
 ├─ 5. 图片密钥:imgkey-probe(需微信运行)
 │     ├─ found → 记录到本次环境变量(不写明文文件)
 │     └─ not resident → 请用户在微信里点开几张图片,再扫一次
 └─ 6. 导出:export(带 --img-key)→ 验收(manifest 人话对账)
```

## 各阶段操作

### 0. 准备

便携包时:`dist/` 下已有 exe,无需 Go。**路径约定**:本文命令里的 `scripts/`、`strategies/` 是仓库根相对路径;便携包里 `observe.ps1` 在根目录、`strategies/` 同级。源码时:

```bash
go build -trimpath -o weixin-key.exe ./cmd/weixin-key
go build -trimpath -o material-probe.exe ./cmd/material-probe-diagnostic
go build -trimpath -o imgkey-probe.exe ./cmd/imgkey-probe
```

### 1. 观测(只读,不碰微信)

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/observe.ps1
```

从输出取:`install.dll`+`dll_sha256`、主进程(`is_main=true` 的 pid/created)、
`recommended` 账号目录。**多账号时必须让用户确认选哪个**。

### 2. 配置

```bash
./weixin-key.exe doctor --db-root "<账号根>" --config "<新配置路径>" --out "<导出目录>" --pretty
./weixin-key.exe init-account --db-root "<账号根>" --account "<账号id>" --config "<新配置路径>" --pretty
```

- 配置路径必须在微信数据目录**之外**,且不存在(init-account 拒绝覆盖)。
- 判据:doctor exit 0 且 `source.status=selected-not-authenticated`;
  init-account `metadata_verified=true`。

### 3. 缓存优先

```bash
./material-probe.exe --verify-cache --root "<账号根>" --cache-config "<配置>" --cache-config-sha256 "<当前配置哈希>"
```

- `cache_hits == source_dbs` → 跳第 4 步。配置哈希每次用前重算。

### 4. 有界被动取钥(缓存缺失时)

```bash
./material-probe.exe --module "<dll>" --sha256 "<dll哈希>" --pid <主进程pid> \
  --created "<RFC3339Nano创建时间>" --root "<账号根>" --exe "<Weixin.exe路径>" \
  --primary-db "message\message_0.db" --snapshot-check --scratch "<已存在空目录>" \
  --cache-verified --cache-config "<配置>" --cache-config-sha256 "<配置哈希>"
```

- 判据:`report.status=all-page1-hmac-verified` 且 `cache.update.applied=true`。
- 之后**重算配置哈希**,再跑一次第 3 步验证门。

### 5. 图片密钥

```bash
./imgkey-probe.exe --pid <主进程pid> --dat "<账号根>\msg\attach\<任一目錄>\<月份>\Img\<任一_t.dat>"
```

- 未命中 → **请用户在微信里点开 2-3 张聊天图片**,然后原样重扫。
- 命中后:`$env:WECHAT_CLI_IMGKEY_HEX = "<key>"`(只进环境变量,不落盘)。

### 6. 导出

```bash
WECHAT_CLI_CONFIG="<配置>" ./weixin-key.exe export --out "<导出目录>" \
  --db-root "<账号根>" --account "<账号id>" --pretty
```

增量:同目录重跑即可,工具自动合并(0 new/全 unchanged 是健康的重复导出)。

### 7. 验收(人话对账)

读 manifest.json 后向用户报告:
- 消息总数、会话数、时间范围
- 附件四类计数:decrypted(图片已解密)/ copied(语音视频文件)/ missing(源文件已被微信清理,无法恢复)/ not-extracted(表情)
- `status=partial` 时说明具体哪部分不完整,**不把 partial 说成 complete**

## 失败处理速查

| 现象 | 含义 | 动作 |
|---|---|---|
| observe 无进程 | 微信没开 | 请用户启动并登录 |
| init-account 拒绝 | 配置已存在 | 换路径或复用现有配置,勿删 |
| verify-cache 部分命中 | 密钥轮换/新库 | 重跑第 4 步取钥 |
| 取钥 0 命中 | 版本不兼容 | 停止,写策略草稿,报告 |
| imgkey-probe 未命中 | 密钥未驻留 | 请用户点开图片后重扫 |
| export 缺图 | 微信已清理源文件 | 属正常 missing,如实报告 |

## 策略管理

`strategies/` 按 Weixin.dll SHA256 记录已验证版本的知识。新版本首次成功后,
把模块哈希、静态 RVA、踩过的坑写成新 JSON;**密钥永不写入策略文件**。
