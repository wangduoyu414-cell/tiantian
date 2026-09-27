# 显式只读材料诊断

该入口用于检验一个**静态假设**，不是正式 setup 回退，也不是桌面导出功能。生产主动采集安全门不变。

离线结构核验还支持 `--semantic-check`（必须和 `--verify-cache --snapshot-check` 一起使用）。它在同一私有临时快照上有界统计消息正文的 SQLite 运行时类型、`WCDB_CT_message_content`/`WCDB_CT_source`/`compress_content` 标记、`real_sender_id` 非空情况、`packed_info_data` 非空情况和时间单位；不输出表名、行值、正文、联系人或密钥，也不进行压缩解码。该统计用于决定 parser 适配，不是完整导出成功证明。

## 假设和边界

- 只接受显式完整模块路径及 SHA256；不加载该 DLL。
- 在 AMD64 PE 可执行区中寻找最多四组严格的 59 字节指令形状，并要求完整落入 `.pdata` 所述函数。
- 静态匹配不是材料含义、调用关联或生产捕获 ABI 的证明。
- 真实诊断只观察一个由 PID、创建时间、完整 EXE 和账号文件持有者绑定的进程；前后复核账号文件持有者以及加载模块的路径、映像大小和每个候选代码段。账号关联丢失或无法复核时清零成功计数；这些是开始/结束观察，不声称中途关联始终连续。
- 新对象布局独立于旧 D0：`[ptr, 0, length=32, capacity=47]`。只扫描已提交的私有 PAGE_READWRITE 内存；不调试、不暂停、不启动或关闭微信、不写进程或源库。
- 显式选一个 `db_storage` 相对数据库；先试 raw，再试直接二进制口令和最多四种模块常量 XOR32 还原后的口令。只有该库验证命中才扩展其他库。
- 未命中仅约束这个布局、观察区间、主库及转换集合；不能推断其他库、其他生命周期或整个进程内存中不存在材料。
- 上限：512 MiB 请求读取、65536 次区域查询、65536 个结构、4096 个去重候选、64 个数据库、512 次 PBKDF2。单 worker，无候选持久化、无后台队列。
- CLI 90 秒为协作截止（包括静态准备），不是 OS 调用的硬实时上限；真实执行另由外层监督自有 helper，禁止以微信为超时终止目标。

## 独立入口

构建：

```powershell
go build -trimpath -o material-probe.exe ./cmd/material-probe-diagnostic
```

只读磁盘检查：

```text
material-probe.exe --inspect --module ABSOLUTE_MODULE_PATH --sha256 EXPECTED_SHA256
```

真实受限诊断（所有目标字段必须显式给出，不读环境或配置默认值）：

```text
material-probe.exe --module ABSOLUTE_MODULE_PATH --sha256 EXPECTED_SHA256 --pid OBSERVED_PID --created RFC3339Nano --root ABSOLUTE_ACCOUNT_ROOT --exe ABSOLUTE_EXE_PATH --primary-db message\message_0.db
```

拒绝未知、多余、重复参数和溢出 PID；静态模式拒绝任何真实目标参数。stdout 只含 JSON 计数、固定错误类别及静态模块证据，不含候选、材料、盐或聊天。exit 0 表示诊断完成，不表示解密成功。预算/取消/验证不完整不能算“无密钥”；后复核失败清除成功计数。

## 可选整库验证（不写配置，不保留明文）

`--snapshot-check --scratch ABS_EXISTING_DIRECTORY` 必须一起显式给出；不能与 `--inspect` 混用。默认诊断仍只计数并销毁材料。

新 `WithVerifiedPassiveKeys` 只保留已认证的每库派生密钥，不保留观察材料或口令；只有全部库命中、所有后复核完成且未取消/出错后，才能在同步回调中使用。容器禁止JSON序列化、格式化固定去敏；借入buffer及容器退出时清理。Go及原生接口中的字符串无法承诺完全清零，不能把这些措施说成绝对内存擦除。

整库检查仅针对显式主库：把观察期间已核验的同一源Root保留至同步消费结束，拒绝根目录替换或junction，即便junction指回原目录也拒绝。原Root相对打开文件并与无reparse、禁止替换的只读文件句柄核对后，才交现有resolver再次验证当前源；不禁止微信普通写入。再以原生读事务备份、Go全页认证解密、SQLite完整性检查和 `Msg_` 表行数统计验证可读性。不返回消息正文、表名或密钥。

临时目录通过Windows创建参数原子赋予当前用户专用、对子项继承的DACL，从创建时即私有；并通过句柄读回核验。不先宽松创建再收紧权限，因为已获授权的句柄不会被事后ACL变更撤销。只删除本次精确临时文件及空目录，清理失败返回错误。不得将scratch放进源目录。正常运行不保留明文；强杀helper/系统异常可能留下该专用目录内的文件，应按运行证据恢复清理，不能宣称强杀也保证删除。

扫描仍使用原有90秒/512KDF上限；快照模式CLI整体8分钟协作截止，外层只能监督/终止自有helper。该模式不是正式导出，也不写DPAPI缓存或替换桌面程序。

## 验证与尚未集成

合成测试覆盖精确指令、畸形 PE/错误 hash、跨块对象、读/查询/对象/候选/KDF 上限、取消、模块映射和代码变更、主库未命中不扩散。

原生独立测试使用项目固定的 SQLite3MC 引擎，以包含 NUL 与高位字节的 32 字节口令造两个不同盐的加密库；Go 还原及页一认证后，原生读事务备份、Go 全页解密、独立 SQLite 完整性检查和查询通过。错误转换、类型混用和损坏页被拒绝。这是合成证据，不是微信真机证据。

2026-09-13T03:15:55Z，r2弃置型诊断真实命中22/22库页一HMAC。r4于2026-09-13T04:11:48Z进一步完成主库127119页全页认证、SQLite完整性及512张消息表702852行计数；临时明文删除，未保存密钥或正式聊天导出。消息行数不等于标准消息解析/去重成功。

## 显式受保护缓存及离线验收

持久化需在真实参数及整库快照参数之外，另外给出：

```text
--cache-verified --cache-config ABS_EXISTING_CONFIG --cache-config-sha256 CURRENT_CONFIG_SHA256
```

只有整库认证、SQLite完整性、私有DACL与明文清理全部通过后才调用缓存。当前所有源库沿原Root再认证；配置账号/root不能切换，配置旧hash必须一致，与旧写入者共用跨进程锁但遇忙立即失败。仅保存已认证的每salt派生raw key、账号/profile/来源，不保存还原口令。使用当前用户DPAPI、一次性受保护备份、原子写入及读回/DACL验证。既有秘密字段保留，不清空旧可用缓存。

`cache.update.applied` 表示原子写入已经返回成功；即使之后读回或外层检查失败也保留true。强杀/输出丢失时实际写入状态可能未知，须检查配置hash/DPAPI读回及备份，不能用旧hash盲重试。锁只协调合作写入者，前后hash不是对任意敌对并发替换的事务保证。旧schema4 metadata备份仅在同账号可读时允许复用；旧迁移路径的验证不放松。

采集helper退出后，使用独立无进程模式：

```text
material-probe.exe --verify-cache --root ABS_ACCOUNT_ROOT --cache-config ABS_CONFIG --cache-config-sha256 CURRENT_CONFIG_SHA256
```

该模式不接受PID/模块/快照/写入参数，不使用环境秘密、不访问微信进程、不写配置或聊天文件。显式保护配置须匹配hash/账号；现有resolver复核每个已发现加密库，成功要求全部typed缓存命中、源页前后稳定及零KDF派生。输出只有计数和状态。上限64库、90秒协作截止，不能把页一验证包装成每库整库/解析成功。

2026-09-13T05:16:26Z，r6真实完成22库/22密钥受保护保存，采集helper退出后另起无进程helper纯缓存22/22命中、零KDF、配置不变；主库快照临时明文删除。详见实施状态。正式导出及桌面集成仍未完成。

## 离线私有快照及结构检查

已有有效缓存后，不需要再次访问微信进程：

```text
material-probe.exe --verify-cache --root ABS_ACCOUNT_ROOT --cache-config ABS_CONFIG --cache-config-sha256 CURRENT_CONFIG_SHA256 --snapshot-check --scratch ABS_EXISTING_DIRECTORY --primary-db message\message_0.db --schema-check
```

沿同一已验证Root同步消费已认证typed key，复用原生快照、Go认证、SQLite检查和原子私有目录/清理，不写配置。`--schema-check`只能与离线快照一起使用；通过table_xinfo计入生成/隐藏列，只返回消息表列名和声明类型的分组指纹及表数量，不返回表身份、默认值、表达式或行内容，预算为最多4096表、每表128列、16种结构。该结构证据不代表发送者/压缩/富消息已经正确解析。

r8于2026-09-13T05:44:43Z实际通过22/22纯缓存与主库私有快照/结构检查；512消息表属于同一个17列shape，临时明文删除、配置/备份不变、helper已退出。`real_sender_id`、`compress_content`、`packed_info_data`与两列WCDB_CT标记为后续解析的重要真实结构证据，尚未验证其内容语义。完整绑定见实施状态；本模式不产生正式聊天导出。
