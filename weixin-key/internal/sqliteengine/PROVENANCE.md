# 固定版本本地加密快照引擎

用途仅为读取本机已授权数据库的事务快照，不负责采集材料。运行时不联网、不下载、不上传用户数据。

## 来源及校验

- 官方上游：`https://github.com/utelle/SQLite3MultipleCiphers`，tag `v2.5.1`。
- 公开发布资产：`https://github.com/utelle/SQLite3MultipleCiphers/releases/download/v2.5.1/sqlite3mc-2.5.1-sqlite-3.53.4-win64.zip`。
- 归档 SHA256：`858c4f2e9e262caca06e6c4fafee62d13dd90c21b649b047f17992bf0402981e`。
- `bin/sqlite3mc_x64.dll` SHA256：`5030decc6d914539e3b9b7e28aa4f6de1e7161dac6fd4b21eb02e6754d9b175e`。
- 归档摘要已与该 release 的 GitHub asset digest 核对；没有声称代码签名、可复现原生构建或干净机器安装验收。
- 官方 MIT 通知保存在 `LICENSE.sqlite3mc`，连同 DLL 通过 Go embed 嵌入 Windows amd64 程序。对外分发前仍需完整第三方通知/依赖与安装环境审查；本批没有对外发布。

## 加载与数据流程

运行时解出到当前用户缓存 `weixin-key/sqlite3mc-2.5.1/<DLL SHA256>/`，校验公共 DLL 的摘要。仅搜索 DLL 自身目录和 Windows System32，不搜索当前工作目录。其他平台明确返回不支持，不能将交叉编译成功当作运行能力。

参数固定为 SQLCipher 4：4096 页、SHA512、256000 KDF 迭代、2 次 MAC 派生、MAC 开启、页号 LE、盐掩码 0x3a。输入是经过 Resolver 验证的 raw enc_key；该上游 API 使用 `raw:` + 二进制 key/salt，不是 `x'hex'` 字符串。

上游 `v2.5.1/src/sqlite3mc_vfs.c` 的 `sqlite3mcIsBackupSupported` 要求备份两端页尺寸和保留区兼容。因此采用：

1. 源连接 `READONLY`，固定读取事务并包含其已提交 WAL。
2. 同配置加密目标，分批调用 SQLite backup API；不使用可能丢表的逻辑复制兜底。
3. 关闭目标连接后，用独立纯 Go 实现逐页验证 HMAC 并原子生成明文。
4. 清理作业临时加密库；源主库/WAL 不执行写入、删除或 checkpoint。SQLite 的 SHM 锁定信息不等同于源消息内容。

`mc_legacy_wal=0` 是先加密再计算 WAL 校验和的格式；`=1` 是旧 SQLite3MC 的不同格式，不能因名字含 legacy 而误选。此参数由上述固定上游源码核对，并通过原生生成 WAL 与独立 Go WAL 读取器交叉验证。

## 测试边界

`go test ./internal/sqliteengine -count=1` 覆盖独立纯 Go 加密夹具、原生生成活跃 WAL、事务期间后续提交隔离、源主库/WAL 内容未变、错误密钥、后续页面损坏、取消和旧目标保护。

这些是合成与本机原生 SQLite 集成证据，不是实际微信版本的采集/登录/schema 证据。跨库不声称同一个事务时点。DLL 仅固定依赖与哈希，尚无 Mac 引擎/安装包验收。
