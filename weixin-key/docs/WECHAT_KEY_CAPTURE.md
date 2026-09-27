# WeChat 本地库密钥提取与版本对抗指南

> 适用范围：在**自己账号、自己电脑**上提取微信 4.1+ 本地数据库（`xwechat_files/.../db_storage`）的解密能力。  
> 仅用于本地数据导出/备份。请勿用于获取他人数据；hook/注入有违反微信用户协议的风险。

## 密码学地基（已在 4.1.12.x 实测验证）

```
passphrase(32 字节, 每账号一把)
  → enc_key  = PBKDF2-HMAC-SHA512(passphrase, db_salt, 256000, 32)   # 每个库 salt 不同 → enc_key 不同
  → mac_key  = PBKDF2-HMAC-SHA512(enc_key, salt ⊕ 0x3a, 2, dklen=32) # 注意: dklen=32, 不是 64
  → 校验     = HMAC-SHA512(mac_key, page[16:4032] || LE32(1)) == page[4032:4096]
  → 解密     = AES-256-CBC, 页 4096, reserve 80 (IV16 + HMAC64)
```

- 一个账号一把 passphrase，全部库共用；每库独立 salt → 派生出该库专属 enc_key。
- **在账号未轮换密钥、微信未更改加密参数的前提下**，passphrase 可长期离线复用（含之后新增的库）；密钥轮换或加密参数变化会使缓存失效，届时必须重新捕获。缓存 enc_key 每次使用前重新校验，不凭旧记录直接信任。

## 版本更新后的快速重定位（核心能力）

版本升级可能改变代码、锚点和调用约定。`relocate` 输出静态候选，不证明候选是可信捕获位置，也不保证未来版本仍有同样的字符串：

```bash
# 自动发现安装分析候选（不是进程加载身份的证明）
weixin-key relocate --pretty

# 或指定任意一个 Weixin.dll
weixin-key relocate --dll "C:\Program Files\Tencent\Weixin\4.1.12.55\Weixin.dll"
```

输出每个锚点的 `func_entry_rva`（这是代码地址，不是密钥）：

| 锚点 | 含义 | 4.1.12.55 实测入口 |
|------|------|--------------------|
| `MMV1` | codec 配置关联的静态函数候选（捕获语义须另验证） | `0x353bc60` |
| `x'%s'` | key 字符串格式化点（冗余锚点；某些版本命中少） | `0x356ec90` |
| `Config.Cipher` | WCDB `com.Tencent.WCDB.Config.Cipher` 配置入口 | `0x145adb0` |

> 实现：[`internal/wxkey/relocate.go`](../internal/wxkey/relocate.go)，纯 `debug/pe` 静态解析，跨平台、无依赖。  
> 若某版本字符串锚点被加密，存在实验性 KDF 常量锚点（`256000` = `0x3E800`）退化路径；注意它按指令引用目标匹配，命中率取决于该常量的实际编码方式，未实测的版本不能据此宣称可定位。

## 主动采集当前状态：安全隔离，尚未通过验收

2026-09-12 独立审查发现旧调试实现没有完整恢复原 DR/TF、吞掉 Wait/Continue/Detach 错误、未安全管理事件 hFile，并把第一处字符串引用当可信断点。因此当前候选在关闭/启动/附加/加载 wxkey-dll 之前强制返回安全错误；环境开关不能豁免。

- 当前机器安装模块 4.1.13.65 的静态 `x'%s'` 候选并非受支持捕获 profile，不对真实微信设置断点。
- 账号控制器先收集并固定全部目标身份、检查 exe 一致/其他实例/捕获准备，再请求正常关闭；预检查失败不关闭任何窗口。关闭多个进程不是天然原子操作，途中退出失败也不会强杀。
- 非重启策略也只接受所选 DB 关联的进程，保留句柄并在每个路由前检查是否退出；无关联证据明确失败。
- debug-r2 已合并调试循环，补上正常/部分可恢复路径的原 DR/TF 恢复、事件错误传播、已接收事件 hFile 所有权及先清理目标后 join。首次读取 context 失败、随后恢复读取成功时的异常重新分类，已获独立有限 PASS。
- 恢复 API 失败、detach 失败的两个 P1 已增加保留 owner/句柄、恢复读回和显式重试实现。具体候选的独立结论以唯一实施状态文档为准，不能仅凭合成通过宣称全量验收。
- KillOnExit 前置准备已改为同线程先创建工具自己的隐藏惰性 bootstrap，设置成功并回收后才进入账号关闭/目标 attach/launch。自建未来 debug 子进程在实际 owner OS 线程退出后存活并脱离、恢复创建暂停一次后 exit0；设置失败＋detach 失败不会返回准备完成。这些不是本人微信生存/捕获或完整 GUI 的验收证据。
- 持续恢复失败时 GUI 保留安全退出和恢复重试入口，CLI `setup` 用 Ctrl+C 取消、恢复等待时再次 Ctrl+C 请求一次重试。普通取消/关窗不能代替实际清理；不承诺拦住强制结束进程、终端关闭或操作系统关机。
- 4.1.13.65 的单个候选已用本地有界解码核对指令边界和两处字段 getter；这不是 live 模块身份或材料类型证明。没有可信 startupAddr/捕获 profile，不得启用。
- 完整生命周期修复、版本绑定语义证据、独立复审和本人登录闭环仍须完成。暂时隔离不等于这些实现已经做好，准确候选与证据见 `D:\weixinpojie\weixin-key\docs\IMPLEMENTATION_STATUS.md`。

有效缓存或本人已授权离线材料可继续用于以下路径：

```bash
weixin-key setup --pretty   # 写入 ~/.config/wxcli/config.json，不向 stdout 回显密钥
weixin-key info --pretty
# 离线 page-1 HMAC 校验，无需 WCDB 原生库；自定义数据目录时设 WECHAT_CLI_DB_ROOT
weixin-key verify --passphrase <64-hex> [--db path\to.db] --pretty
# 离线解密为明文 SQLite（纯 Go；key 来源: --enc-key/--passphrase/WECHAT_CLI_PASSPHRASE_HEX/config 缓存/config passphrase；--no-config 禁用 config 回退）
weixin-key decrypt --db path\to\message_0.db --out plain.db --pretty
```

> 实验室脚本与敏感产物（**勿提交 git**）应隔离存放，不进入本仓库。

## 内置路径说明

- **passphrase 落库**：Windows schema 5 使用当前用户 DPAPI 保护，旧 schema 仍可读，首次写入前创建受保护备份；旧版明文读取器不兼容新格式。非 Windows 新秘密持久化尚未实现。详见 README“配置落盘”。
- **Route D（堆内常驻 enc_key 扫描）**：[`internal/wxkey/bruteforce_windows.go`](../internal/wxkey/bruteforce_windows.go)。mac_key dklen 已从 64 修为 32（此前必假阴性）；**修好后 Route D 在 4.1.12.x 是否仍有效（enc_key 是否常驻）尚未全面实测**。
- **已证伪、勿再当主路径**：全内存 passphrase × 256k 盲扫（算不动）、锁死旧版 RVA 的注入（版一更就废）、不重登硬挂（时机已过）。

## 长期对抗原则

1. **不追每个版本的 RVA**，追「版本自适应定位」（relocate）这个能力本身。
2. **复用但重新验证**：缓存始终重新验证，密钥轮换、加密配置变化或账号变化都可能要求新材料，不承诺“一次永久”。
3. **授权与隔离**：仅限本人本机；不上传秘密或聊天，不强杀、不绕过权限。静态候选、合成测试、独立审查与真实捕获证据分别记录。

## 验证与回归

```bash
go test ./...
# 重点包：
go test ./internal/wxkey/   # mac_key dklen 黄金回归 + 真实 DLL relocate（本机装微信时）
go test ./internal/wcdb/    # KDF / 打开路径
go test ./internal/config/  # 配置落盘安全
```

- `TestSQLCipherVerifierAcceptsCorrectKey`：守护 dklen=32，防止退回 64 导致全员假阴性。
- `TestRelocateAgainstRealWeixinDLL`：本机装有微信时，对真实 DLL 验证三锚点可定位（CI 无微信自动 skip）。
