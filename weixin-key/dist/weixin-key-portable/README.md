# weixin-key

WeChat 4.1+ Windows 本地数据库密钥提取工具（**仅限自己账号、自己电脑**）。

验证已有 SQLCipher 材料、静态分析安装模块，并对支持的本地库进行快照、解密和导出。

> **新电脑／Agent优先交付方向：**先完善无需旧机缓存的本机首次取钥→提取→导出工具链，完整GUI可以后置。`doctor`（只读新机诊断）和 `init-account`（显式不覆盖的账号元数据初始化）之外，2026-09-27 已在第二台独立电脑完成端到端验证（Weixin 4.1.15.13：取钥→缓存→离线复用→全量导出含解密图片）。新机操作以 `skills/wechat-export/SKILL.md` 决策树为准；`scripts/observe.ps1` 一键收集现场参数；版本知识见 `strategies/`。

> 当前开发候选的主动 hook、自动关闭/调试启动、wxkey-dll 已安全隔离：独立审查发现调试清理和版本绑定未达验收条件。开启环境变量也不能绕过。有效缓存和离线导入仍可使用；这不表示无材料的一键导出已经完成。请勿继续使用桌面旧程序试错，准确状态见 `docs/IMPLEMENTATION_STATUS.md`。

> 仅用于本机数据导出与备份。请勿用于获取他人数据。Hook / 注入可能违反微信用户协议。

## 核心能力

| 命令 | 作用 |
|------|------|
| `doctor` | 检查显式新机路径/平台，输出状态与下一动作；不读秘密或DB正文，不证明可导出 |
| `init-account` | Windows新建账号元数据配置，禁止覆盖既有目标；不运行采集、不创建密钥 |
| `relocate` | 对 `Weixin.dll` 做静态分析，输出锚点/函数候选 RVA；不证明指令边界、寄存器约定或捕获安全 |
| `setup` | 优先验证缓存/离线导入并写入受保护配置；被动路由限定所选账号文件持有者，主动路由目前阻断 |
| `verify` | 用 32 字节 passphrase 或 per-DB enc_key 验证本地 DB，统计 match 数（不匹配时退出码为 1） |
| `decrypt` | 纯 Go 离线解密单个 `.db` 为明文 SQLite（page-1 HMAC 先验 key，不落半成品；支持加密 WAL 重放） |
| `export` | 一键导出：快照+解密全部本地库，输出 chat-v1 Markdown 分卷 + JSONL + manifest（缓存有效时全程离线，不碰微信进程）；`--img-key`/环境变量 `WECHAT_CLI_IMGKEY_HEX` 提供图片密钥时 .dat 图片解密为 jpg/png/wxgf |

`material-probe`（`cmd/material-probe-diagnostic`）：有界只读材料诊断与缓存验收，见 `docs/MATERIAL_PROBE.md`。

`imgkey-probe`（`cmd/imgkey-probe`）：只读内存扫描账号全局图片 AES 密钥。微信按需加载该密钥——未命中表示用户还没在当前会话看过图片，点开几张聊天图片后重扫即可。
| `info` | 查看当前 config 是否已有 passphrase / key map（不打印密钥） |

## 快速开始

```bash
# 开发构建使用 go.mod 指定的 Go 1.26.5；或直接 scripts\build-dist.ps1 产出便携包
go build -o weixin-key.exe ./cmd/weixin-key
go build -o material-probe.exe ./cmd/material-probe-diagnostic
go build -o imgkey-probe.exe ./cmd/imgkey-probe

# 新机第一步：只读环境观测（安装/版本/进程/账号目录/策略匹配）
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\observe.ps1

# 微信升级后：输出静态定位候选（不是可直接启用的 hook 点）
./weixin-key.exe relocate --pretty

# 查看已缓存的密钥材料概况（不回显密钥）
./weixin-key.exe info --pretty

# 用已知 passphrase 离线验证某个库（纯 Go page-1 HMAC，无需 WCDB 原生库）
./weixin-key.exe verify --passphrase <64-hex> --db "path\to\message_0.db" --pretty
# 或直接验证 per-DB enc_key
./weixin-key.exe verify --enc-key <64-hex> --db "path\to\message_0.db" --pretty

# 离线解密某个库为明文 SQLite（纯 Go，无需 WCDB 原生库）
# key 来源优先级: --enc-key > --passphrase > WECHAT_CLI_PASSPHRASE_HEX
#   > config 缓存 enc_key（每次重新校验，失效即拒绝） > config passphrase
# setup 成功过一次之后，无需任何 flag、无需微信进程即可解密：
./weixin-key.exe decrypt --db "path\to\message_0.db" --out plain.db --pretty
# --no-config 可显式禁用 config 回退。

# 或验证整个账号目录（自定义数据路径时设置 WECHAT_CLI_DB_ROOT）
# export WECHAT_CLI_DB_ROOT="H:\\weixinhuancun\\xwechat_files\\wxid_xxx"
./weixin-key.exe verify --passphrase <64-hex> --pretty
```

## 一键导出聊天记录（chat-v1）

```bash
# 已有有效缓存材料时全程离线，不需要微信进程
./weixin-key.exe export --out "D:\Exports\WeChat" --pretty
```

输出布局：

```
<out>/
  index.md                                   # 总览：账号/范围/完整性/会话索引
  manifest.json                              # run_id、每库快照证据、complete|partial|failed
  accounts/<账号>/conversations/<会话>/<yyyy-MM>.md
  data/messages.jsonl                        # 标准数据（逐消息来源可溯）
  data/conversations.json
  data/contacts.json
  .export-state/                             # 发布状态（崩溃对账用）
```

- 一致快照：明文库使用 SQLite 固定读事务和在线备份；Windows amd64 加密库使用随程序内嵌的固定版本引擎，先得到同配置加密快照，再逐页校验解密。读取已提交的 WAL，不执行源库 checkpoint，也不删除源 WAL/SHM。SQLite 读事务可能使用 SHM 锁定信息。
- 不支持在线加密引擎的平台，仅在能由操作系统阻止写入/替换时允许冻结复制，否则明确失败；不能把“先复制主库再复制 WAL”当成一致快照。当前 macOS 加密在线快照尚未实现。
- 发布前保存恢复日志和旧结果备份，manifest、消息和游标以同一次状态提交为准；中断后先恢复再读取历史。一个输出目录绑定一个账号，换账号请选另一个目录。
- 用户已有文件不覆盖；生成物发生冲突时另存，后续增量只读取状态中登记并通过哈希核验的历史文件，不混入用户笔记或其他账号记录。
- 引擎来源、哈希和支持范围见 `internal/sqliteengine/PROVENANCE.md`；真实微信、桌面操作、Mac 和独立 Review 的未验收项见 `docs/IMPLEMENTATION_STATUS.md`，不要把合成测试通过等同于产品全部完成。
- 用户已有文件永不覆盖：与上次发布 hash 不一致的目标另存 `*.conflict-*` 副本。
- Markdown 为 UTF-8/CommonMark，每条消息带时间/发送者/方向/类型与来源定位；消息体中的代码围栏不会破坏文档结构。
- 重跑确定性：相同源产出字节一致的 messages.jsonl 与分卷。

Windows 安装发现优先使用显式 exe，再使用正在运行进程的真实可执行路径，最后只读查询注册表/ProgramFiles 安装根；支持平铺及数值版本目录。多安装、身份缺失或权限错误拒绝猜测。`relocate` 自动发现的 DLL 只是安装分析候选，不保证它就是进程实际加载的模块；可用 `--dll` 显式指定。

`verify` 默认走 **SQLCipher4 page-1 HMAC 离线校验**（`mac_key` dklen=32），不依赖原生 WCDB；自定义缓存目录不在 `Documents\\xwechat_files` 时，请设置 `WECHAT_CLI_DB_ROOT`；passphrase 建议用环境变量 `WECHAT_CLI_PASSPHRASE_HEX` 传入（命令行参数会进进程列表和 shell 历史）。

## 密码学地基（4.1.12.x 实测）

```
passphrase(32B, 每账号一把)
  → enc_key = PBKDF2-HMAC-SHA512(passphrase, db_salt, 256000, 32)
  → mac_key = PBKDF2-HMAC-SHA512(enc_key, salt⊕0x3a, 2, dklen=32)   # 注意 dklen=32 不是 64
  → 校验    = HMAC-SHA512(mac_key, page[16:4032] || LE32(1))
  → 解密    = AES-256-CBC, page=4096, reserve=80
```

- 一账号一把 passphrase；每库 salt 不同 → enc_key 不同。
- **在账号未轮换密钥、微信未更改加密参数的前提下**，passphrase 可长期离线复用（含之后新增的库）。失效（轮换/加密参数变化）时必须重新捕获；缓存 enc_key 每次使用前都会重新校验，不凭旧记录直接信任。

## 推荐工作流（版本更新 / 新机器）

1. 先验证所选账号的缓存或本人已有的离线材料；有效则不操作微信。
2. 缺少材料时先检查支持状态；当前主动路径未通过安全验收，明确停止，不强杀微信、不按字符串候选设置断点。
3. 静态研究应绑定安装模块哈希、版本和架构；候选需另有指令边界、材料约定及生命周期异常清理证据。
4. 将来主动采集恢复后，仍须先完成全部账号/进程/采集准备预检查，正常退出得到确认后才启动，并做本人真机闭环验证。

### 开发候选：取消、恢复与安全退出

主动采集仍被硬隔离，以下恢复逻辑不是启用许可，也不是本人微信验收结果。

- 恢复失败时保留同一个调试 OS 线程、原上下文及句柄；全部修改过的线程读回确认恢复前，不释放目标执行。
- GUI 显示“恢复未完成”和“重试恢复”；关闭主窗口会请求取消，必要时保留安全退出窗口，直到清理和工作线程退出。重试恢复不等于开始新采集。
- CLI `setup` 的 Ctrl+C 请求取消；恢复等待期间再按 Ctrl+C 请求一次有界恢复重试。错误写 stderr，不把恢复失败藏在“已取消”或成功 JSON 中。
- 准备路径先在工具自己的隐藏、惰性子进程上建立同线程 KillOnExit(false)，成功后才允许进入账号关闭/目标启动；子进程退出和双失败已做专用测试，实际微信 profile/登录流程仍未验收。
- 不承诺能拦住任务管理器强制结束、关闭终端、系统关机或进程崩溃。恢复提示出现时不要强制结束工具。GUI 实际窗口交互及 Mac 对应能力另行验收。

详细对抗指南见 [`docs/WECHAT_KEY_CAPTURE.md`](docs/WECHAT_KEY_CAPTURE.md)。

### 4.1.12.55 实测 relocate 结果（示例）

| 锚点 | 含义 | `func_entry_rva` |
|------|------|------------------|
| `MMV1` | codec 配置（主 hook） | `0x353bc60` |
| `x'%s'` | key 格式化（冗余） | `0x356ec90` |
| `Config.Cipher` | WCDB Cipher 配置入口 | `0x145adb0` |

RVA 是静态代码候选，不是密钥。升级后重新 `relocate` 仅是研究起点，不能替代版本适配验收。

## 配置落盘

默认路径仍为 `~/.config/wxcli/config.json`。路径兼容不等于 schema 5 可被旧版 wxkey / wechat-cli 读取。

- 可用环境变量覆盖：`WECHAT_CLI_CONFIG`
- Schema 2：per-salt `enc_key` map  
- Schema 3：账号级 `passphrase` + KDF 名，运行时按需派生 enc_key  
- Schema 4：`key_entries` 为每把 enc_key 记录来源/账号/KDF/验证时间。
- Schema 5（Windows）：公开元数据与 `protected_secrets` 分开；密钥、passphrase、图片材料由当前用户 DPAPI 保护，不使用机器级保护。写文件前限制到当前用户私有 DACL；保护结果先解保护核对再原子替换。
- 旧 schema 只读加载不会迁移；首次写入前将原文件完整字节保护到 `<config>.schema4-backup.protected`。备份或保护失败不覆盖旧配置；已有备份只有实际 DACL 符合当前用户私有要求、且能解出有效旧配置时才复用，不覆盖或自动修改其权限。此前已经存在的明文 `.schema3-backup` **不会自动删除**，需用户自行决定其保管/清理。
- 保护后的配置和备份需要原 Windows 用户的 DPAPI 环境，不能直接拷贝给另一用户/机器使用；当前没有自动回退成明文或面向用户的备份恢复命令。
- 非 Windows 平台仍可读旧明文配置，但新秘密持久化明确失败；Mac Keychain 尚未实现，不能将 CLI 交叉构建当作 Mac 可用证明。

**密钥材料是机密：永不 commit、不上传、不分享。**

## 仓库结构

```
cmd/weixin-key/          CLI 入口
cmd/material-probe-diagnostic/  有界只读材料诊断/缓存验收
cmd/imgkey-probe/        图片密钥只读内存扫描
cmd/weixin-key-gui/      GUI（未验收）
internal/wxkey/          捕获、relocate、Route D 扫描、校验
internal/wcdb/           SQLCipher/WCDB 打开与 KDF
internal/export/         快照、解析（Name2Id/zstd/会话命名）、附件提取、v4 .dat 解码、发布
internal/config/         本地 key map / passphrase 配置（DPAPI）
internal/safefile/       原子写盘
native/                  可选 native capture 原型（C++ / PowerShell）
docs/                    捕获与对抗文档
scripts/                 observe.ps1（新机观测）、build-dist.ps1（便携包）
strategies/              按 Weixin.dll SHA256 索引的版本策略库（永不放密钥）
skills/wechat-export/    Agent 操作决策树与带验证门的包装脚本
```

## 测试

```bash
go test ./...
# 重点：
# - TestSQLCipherVerifierAcceptsCorrectKey  守护 mac_key dklen=32
# - TestRelocateAgainstRealWeixinDLL        本机装微信时跑真 DLL（否则 skip）
```

## 安全与合规

- 仅在**自己账号、自己机器**上使用。
- 不提交 `PASS_PHRASE*`、`SALT_KEY_MAP*`、`VERIFIED_KEYS*`、lab captures、真实 `.db`。
- 不传播密钥；不把本工具用于未授权访问。

## License

Private. All rights reserved.
