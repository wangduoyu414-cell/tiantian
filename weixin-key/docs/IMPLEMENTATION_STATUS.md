# 实施状态（唯一当前记录）

## 2026-09-27 更新：第二台电脑端到端已跑通

在第二台独立电脑（E:\QIMIStudio 工作机，Weixin **4.1.15.13**，此前未验证的版本）上完成全链路：

- **取钥**：有界被动采集 20/20 库页一 HMAC 命中，DPAPI 缓存写入；离线 verify-cache 20/20、零派生。
- **解析语义已补齐**（此前 partial 的三大缺口关闭）：`Msg_<md5(username)>` 表名规则、每库独立 Name2Id 解析 real_sender_id→发送者/方向、`WCDB_CT=4` 为无字典 zstd（已解码，0 失败）。导出 23,385 条消息 / 165 会话，方向 unknown 仅剩系统占位行。
- **附件落地**：语音（Silk，media 库 VoiceInfo）、文件（msg/file 明文）、视频（msg/video 明文 mp4）已按消息归位；图片 `.dat` 为 v2 容器（AES-128-ECB 头区 + 尾部 XOR），账号全局图片密钥经 imgkey-probe 只读内存扫描获得（微信按需加载，需用户先点开图片），5,049 张全部解密，已知明文对逐字节一致。
- **工程化**：`scripts/observe.ps1`（新机只读观测）、`strategies/`（按 DLL SHA256 的版本策略库）、`skills/wechat-export/`（agent 决策树+带门脚本）、`scripts/build-dist.ps1`（便携包，无需 Go）。
- **未完成项更新**：表情提取未做；wxgf→jpg 与 silk→wav 转码未做；missing 附件为微信侧已清理的源文件（不可恢复，非缺陷）；macOS 全部未做；GUI 未验收。

> 本文档中所有 `D:\weixinpojie\...` 路径为开发机内部证据目录，**不在仓库内**，其他机器上不存在；相关结论以本段和 `strategies/` 为准。

## 历史记录（2026-09-13 及之前）

## 最新交付范围调整：新电脑／Agent优先

- 用户要求最终在新电脑或陌生电脑上也能对本人账号取钥、提取、导出；接受在Codex/WorkBuddy这类本机Agent环境中使用功能和说明，完整GUI自动链路可后置。新机无缓存必须是独立验收场景，不能由旧机缓存复用替代。
- 本批新增 `doctor` 只读路径/平台诊断，以及 `init-account` 显式路径、不覆盖、无秘密的Windows账号初始化入口，补上原材料缓存路径“要求已有配置”的新机缺口。指南 `D:\weixinpojie\weixin-key\docs\AGENT_WORKFLOW.md`。
- 诊断不读配置正文/数据库/进程，也不继承秘密环境；初始化仅写所选账号元数据、同目录锁和私有临时文件，不取钥。后续缓存仍由已审路径逐库认证后DPAPI写入；旧setup与主动安全门不变。
- r1独立只读Review发现源目录检查/实际写根未绑定P1，以及映射网络盘判断、CLI信号取消、账号缺值三项P2。原参数缺值实际red复现；额外“已打开父目录移动”变种本来就被拒，不能冒充P1的真实red。r2改为逐层无reparse的本地目录句柄固定、最终卷和Root身份复核、同Root锁/写入；CLI接入信号取消并等待worker；账号缺值拒绝。旧采集和秘密写入行为不扩展。
- r2定向检查及全仓 `go test ./... -json -count=1 -timeout=180s` exit0：454测试PASS、11包PASS、4测试skip、2无测试包、零fail。vet、Windows CLI/probe/GUI构建及Darwin arm64 CLI交叉构建均exit0；无race detector/Mac运行证据。后续复审及候选见下文。
- 编译后的r2 EXE在空HOME、中文空格路径和污染环境的合成新机目录里：doctor不写文件、init创建私有metadata、重跑拒绝覆盖、metadata不能冒充密钥缓存、源/输出/默认HOME不写，六项检查全部符合预期。不是新电脑实际取钥/聊天导出。
- r2复审遗留P1：metadata-only目录句柄不建立实际write/delete共享排除；两种访问权限实际red。r3加入FILE_LIST_DIRECTORY后真排除，但Root.Link/CreateHardLink/MoveFileEx在这些强pins下实测sharing violation，三份失败保留。
- r4只调整无秘密bootstrap为O_CREATE|O_EXCL独占新建，不再承诺原子发布；创建即Applied，写前设置/验证私有DACL，取消/失败可留空或不完整metadata，绝不覆盖重试。真正密钥缓存的DPAPI/备份/原子替换算法不变。r4复审关闭pins及此契约，发现强pins在未持/已释放文件锁时会干扰其他合作writer的P2。
- r5新增当前用户命名mutex，在所有Windows配置写者打开目录前协调，init强pins释放后才释放协调；普通写者等待，显式缓存更新/init立即报忙。原文件锁保留；无生产环境变量旁路。API/CLI取消、先创建目标竞争、祖先真实write/delete冲突、读回失败、空文件状态和两向writer交错均有合成检查。测试命名空间仅在config测试二进制中隔离；没有旧EXE或不合作进程兼容承诺。
- **本批最终r5：**定向检查、全仓test/vet、Windows CLI/probe/GUI及Darwin arm64 CLI交叉构建均exit0；460测试PASS+11包PASS、4测试skip、2无测试包、零fail。编译EXE空HOME/中文路径/污染env六项预期检查通过，正式配置hash不变、正式输出仍空。r5首轮漏处理CreateMutexEx的“有效句柄+ERROR_ALREADY_EXISTS”使并发测试真实FAIL，修正后新日志通过，失败未覆盖。
- r5独立只读复审核对18/18哈希、关闭writer协调P2，本次delta无新增实质发现；结合此前限定审查，不是全产品PASS。候选清单SHA `82FDFBA3481C10454AF478A75C255C77AF52B907550069F67370EEEE39B0F123`。代理已关闭；未独立跑测试/真实数据。跨进程/跨会话Global协调、原生Ctrl+C、第二台独立电脑、Mac运行和race detector仍无专门实测。
- 证据根 `D:\weixinpojie\weixin-key\.build\validation-20260913-onboarding`。保留r1启动器1.26.1强制local造成的未执行失败，以及r2测试unused import首次编译失败；修正后用已有Go1.26.5真实工具链检查，不把失败日志覆盖成PASS。
- 原有22库缓存、主库解密证据仍有效；本批不重复扫描微信、不改真实配置/源库。Windows正式staging已改为创建时私有DACL并回读校验，同时加入源库/WAL容量下限；解析器已适配新版packed local type、real_sender_id、WCDB_CT标记及压缩/二进制无损保留。**真实全量导出已实际运行，但当前结果为partial：numeric real_sender_id目录关联、压缩正文解码和附件仍未完成；第二台新电脑端到端仍未完成。**

更新：2026-09-13 UTC（本机9月12日晚）。**真实22/22库已验证并保存当前用户DPAPI缓存；采集helper退出后，独立无进程程序22/22纯缓存命中，无需重新做口令派生。主消息库127,119页认证、SQLite完整性及真实列结构均已通过；512张消息表共702,852行、同一种17列结构，临时明文已删除。正式消息导出/完整产品尚未完成，主动采集仍硬隔离。** 主线已转为真实消息语义适配，不再重复取钥。

## 当前下一步（r8真实结构核验之后）

1. 在不改变源库的前提下继续核验 numeric `real_sender_id` 与联系人/会话目录的真实关联；当前已尝试 rowid 映射，真实样本仍未形成可确认的显示名，方向保持 unknown，不冒充收/发。
2. 为 `WCDB_CT_message_content=4` 的压缩BLOB寻找版本绑定的字典/解码证据；当前仅保留 base64 原始字节并标记 partial，不把压缩数据当正文。
3. 对正式输出做抽样对账：消息ID、时间、方向、原始字节长度、Markdown与JSONL一致性；之后再处理附件引用/解码、增量与GUI。
4. 全平台/附件/增量/GUI/发布要求保持；不能把本次partial输出当成完整导出。报告中的`derivations=0`是resolver口令派生次数为零，页面认证本身仍需要计算。

完整要求：D:\weixinpojie\.owner-supervision\执行清单-桌面一键导出.md。全部平台、采集方向、解析、附件及最终验收要求保留。

## 材料路线新批次（20260913-material，尚未全量验收）

- 新增明确 opt-in 的材料诊断和静态检查 CLI；详见 D:\weixinpojie\weixin-key\docs\MATERIAL_PROBE.md。原 setup 路径和主动门不改。
- 本机固定模块只读静态结果有两个59字节形状，RVA 0x33caa5 / 0x33ed6c，仅为转换假设，不是有效密钥或生产ABI。
- 区分旧D0 `[ptr,length32]` 与新 `[ptr,0,length32,capacity47]`；新路线绑定显式主库、模块hash、代码前后复核，增加有界直接口令和XOR32还原验证。
- 定向检查命令：`go test ./internal/wxkey ./internal/sqliteengine ./cmd/material-probe-diagnostic -run 'Test(NativeBinaryPassphrase|NormalizeObservedPassphrase|Material|BoundedString|Diagnostic|BoundedObjectProbe|PassiveProbe|Probe)' -count=1 -v -timeout=120s`，exit0；日志 D:\weixinpojie\weixin-key\.build\validation-20260913-material\material-green-r2.txt。
- 原生独立加密 fixture 的两个不同盐均完成还原、材料类型验证、原生快照、Go全页验证与SQLite完整性/查询；未使用真实微信数据。
- 新模块复核将“不能验证”与“实际变化”分开；未完成复核不保留成功计数。预算/取消、错误映射、PE/hash、错误转换及损坏页测试通过。
- material-r1冻结173文件（清单75EB46427D7092B6BE38DB8F90DF8D7A484B23A2AAE957156C5D98B900FD4333），全仓11测试包/356通过事件、4测试skip，vet/gofmt/Windows probe+CLI+GUI与Darwin CLI交叉构建exit0；没有race或Mac运行证据。
- r1独立Review为FAIL：账号owner只开始核对，结束可能已解除关联而PID/旧页/模块不变。未启动r1真实采样，不写PASS gate。工作树已补账号owner结束复核；关联丢失、PID重用、查询失败、取消都拒绝并清零，正在新的r2候选复审。
- r2清单5D02A29154A563B5D1CA29EBCFDCB1237152811AD4F7D6155174934B797E8566，173/173。独立Review关闭P1；外层脚本“后置失败仍exit0”P2另已修复、合成9+3分支通过及限定复审关闭。源码、脚本、EXE三项hash精确绑定，详见 .build\validation-20260913-material\r2。
- r2全仓11测试包通过、364 pass事件、4测试skip，vet/gofmt/Windows各构建/Darwin CLI交叉构建均exit0；没有race/Mac运行证据。
- **真实r2：2026-09-13T03:15:38Z至03:15:55Z**，22/22首页面HMAC均通过，normalized_dbs=22；310私有RW区域、46186400请求读取字节、163结构、38候选、134KDF、172HMAC、0读取失败。进程/账号owner/模块/源页前后稳定，module代码共236字节；没有读出聊天正文/材料到报告，没有写配置或重启/调试微信；helper21072已退出，外层exit0，配置hash仍2E97C9C8BA421CB603B15C4CF0926ABAEB7E127CC6DFA128F85C0BAA5C648AF9。
- 报告 D:\weixinpojie\weixin-key\.build\validation-20260913-material\r2\live-attempt-1\report.json；execution.json为外层证据。该诊断丢弃材料，不是缓存可用或整库导出证据。
- 新增同步 `WithVerifiedPassiveKeys` 和CLI `--snapshot-check --scratch`，在所有验证稳定后才把派生原始key借给现有resolver；仅对主库做私有临时快照、全页认证及SQLite检查，正常删除明文，不写配置。其定向合成检查已通过，尚未冻结复审或真实运行；详见MATERIAL_PROBE.md。
- r3冻结177文件（854D078B1C836EE8FE8B6465BD349F4E977E732226468200844BED246923B69B）全仓11包/376 pass事件、4测试skip及其余检查exit0；独立Review仍FAIL：消费重新开Root未绑定原身份P1、临时目录先宽松创建后收紧ACL P1、值副本绕过指针脱敏P2。未写r3gate、未真实整库尝试。工作树现保留原Root至消费、原子私有创建、值receiver保护，并新增根替换/junction/值副本/DACL继承回归；值副本有实际red日志，正在r4复核。

## 当前授权及恢复基线候选

### 材料r4已验及缓存r5准备

- r4冻结177文件，清单SHA256 `87C37C3CE192D5581BBC41D2D9D07B0A78C9F0ECE74EBE75F07D858FC1BB7CE6`；此前r3三项发现已在r4限定独立Review中关闭。全仓11包、382通过事件、4测试skip；vet/gofmt/Windows probe+CLI+GUI/Darwin CLI交叉构建均exit0；无race/Mac运行证据。
- r4真实执行2026-09-13T04:11:18Z–04:11:48Z：再验22/22库，主库全页127119页、522张表、512张Msg_表共702852行，SQLite完整性通过。原生读事务快照、Go全页认证；实际私有DACL及临时清理通过，helper38236退出，内外exit0，配置hash不变。报告：`D:\weixinpojie\weixin-key\.build\validation-20260913-material\r4\live-attempt-1\report.json`。行数不是已解析、去重和导出的消息数。
- 当前新增显式缓存：固定配置绝对路径/旧SHA，账号不变，沿原Root逐库重新认证，fail-fast旧配置锁、DPAPI保护备份及原子写入，持久化只含typed per-salt raw keys。写后保留Applied证据并校验实际内容/DACL；失败后须核对真实状态，不盲重试。默认诊断与独立快照仍不写配置。
- 原来仅metadata的保护备份会阻塞第二次显式缓存更新，已实际red复现并修复为仅显式路径接受同账号可读metadata备份；旧迁移的秘密备份约束不放松。失败日志保留 `cache-backup-red.txt`；`cache-check-r3.txt`为定向通过日志。测试隔离用户目录；首次测试harness因PowerShell只读HOME变量停止，未运行测试，另有记录。
- 新增 `--verify-cache`：显式root/config/hash、拒绝进程/模块/写入参数，不读环境秘密、不访问进程、不迁移或写配置；用现有resolver逐库再验typed缓存，要求覆盖全部加密库且零派生。合成两盐、错误账号/hash、取消、源损坏、环境污染已覆盖。
- 上述缓存新代码尚待新冻结全仓检查、独立Review及一次真实持久化/独立进程缓存验证；不能用r4通过代替缓存通过。正式导出前还须修复Windows staging实际DACL边界、容量核对，并验证真实schema/发送者/压缩内容，先有界小样本再全量。
- r5冻结182文件（清单0000E717A1FEB85D8BA051C363705E59C3194739A5132B122334539EF801D395），全仓11包/425通过事件/4测试skip及其余六项检查exit0；独立Review仍FAIL：提交/读回期间取消被吞P2、空typed key回退legacy仍报告typed证明P2。未写r5gate、未真实写缓存。两项已实际red复现；工作树合并ctx错误但保留Applied/读回，严格离线验收拒绝空typed key并核对实际resolver使用值，不改其他消费者legacy兼容。`cache-review-green.txt`定向通过，正在新r6冻结复审。
- 仅元数据统计2026-09-13T05:01:52Z：22个DB共983584768字节，22个WAL共47612432字节；消息目录8库共923586560字节。不能把单主库行数当全部库消息数；正式导出还需空间预算、逐库快照/解析对账。
- **r6缓存实际成功，2026-09-13T05:15:52Z–05:16:26Z**：再次22/22页一、主库127119页/SQLite完整性及512消息表702852行通过；22个唯一per-salt raw key存入schema5当前用户DPAPI，受保护原metadata备份创建、原子写和读回/DACL通过。helper30912退出后，新helper37496在无PID/模块参数、无进程/写配置路径下纯缓存22/22命中、零KDF，内外exit0；scratch空。配置SHA现为 `ACA75065BF2EA9A38F68C0D9C86E3EA3BF88FE1F13BDC3F634EBF51BB27FFE02`。不再重复采集；新增数据库/轮换仍须重新验证。
- r6限定Review关闭r5两项P2；冻结182文件清单 `F1FCF3D850DC8EA270D11A7183AE465BD687CBC8AFADB1AF6EDC437D4D85F383`，七项检查exit0，418测试PASS+11包PASS、4测试skip；无race/Mac运行。最终 `D:\weixinpojie\weixin-key\.build\validation-20260913-material\r6\evidence-binding.json` SHA `8AE267927359CBBFBFFE8997940C1C45CA7B5B60286BA885B6CBC5EA287EA208`。
- 当前r7准备：`WithVerifiedProtectedCache`在所有typed缓存/源页验证后延续同一Root到同步私有快照，不访问进程或写配置；新增离线`--schema-check`只报告消息列名/声明类型的去表身份shape，最多4096表/128列/16种shape，不读取或输出消息正文/字段默认值。合成callback生命周期/取消、参数隔离、真实SQLite元数据分组/预算和原生快照清理通过，日志 `cache-schema-check-r1.txt`；尚待新冻结Review及实际schema读取。
- r7组合Review FAIL：`table_info`漏生成/隐藏列导致shape冲突与128总列预算被绕过P2，两个触发均已实际red；工作树改`table_xinfo`，仍仅取name/type，不输出default/表达式/正文，`schema-generated-green.txt`通过。r7七项检查exit0、420测试+11包PASS、4测试skip；有一个清单外PowerShell ModuleAnalysisCache（184已列源码hash无漂移），原候选不改。两个合成PowerShell测试现指定t.TempDir工作目录，惰性child改为先Kill再Wait并隐藏窗口，不再在退出清理时自然等满60秒；不改生产进程控制。即将新r8冻结复审，r7无gate/live。
- **r8真实离线结构检查通过，2026-09-13T05:44:32Z–05:44:43Z**：无需微信进程，22/22typed缓存、主库127119页/SQLite完整性、512消息表702852行通过；全部512表同一17列shape，SHA `fc10316b2abb16c7d2bce87169436ca0e03444b0c03cae7f6022388376e3ce2f`。包含`local_id/server_id/local_type/sort_seq/real_sender_id/create_time/status/upload_status/download_status/server_seq/origin_source/source/message_content/compress_content/packed_info_data/WCDB_CT_message_content/WCDB_CT_source`，只有packed_info_data声明BLOB、三列声明TEXT，其余INTEGER。没有查询或输出正文/default/生成表达式/表身份。
- r8 helper25248退出，内外exit0；配置与保护备份hash不变，scratch空，无源写/进程访问/配置写。冻结184文件清单 `65868BB9967E23FFAAFCDD27458FC19D33C92236242F7E4F7D3F33086DAD22A6`，最终184/184且零额外文件；限定Review关闭r7 P2，七项检查exit0、421测试PASS+11包PASS/4测试skip/2无测试包/零fail，无race/Mac运行。证据 `D:\weixinpojie\weixin-key\.build\validation-20260913-material\r8\evidence-binding.json` SHA `DD7C4457B516BD4FBB323489A46C69C4BF5DE90C9C111024473D502315A5F1BE`。正式输出目录仍空。

- 仓库：D:\weixinpojie\weixin-key；HEAD 99ea0d3fb2a8f245949bec8b9aff75f29b4c55dc；当前任务唯一 writer，保留无关未提交变化。没有 commit/push/release、dist 或桌面替换。
- recovery-r4验收时仅运行合成测试、自建原生线程/惰性子进程和隔离GUI fixture。此后用户明确“允许你做任何事”，已接收本人本机真实核验及解密推进授权；不再等待模块只读核验授权。仍不上传聊天/秘密，不以广泛授权豁免未验收主动采集安全门。
- 用户已批准只读代理 01a095e0-1903-7503-9ba4-b579d57d217a。它只审冻结代码和 Owner 证据，不运行测试/原生操作、不读真实数据、不改文件。没有后台 automation。
- 最终内部候选：D:\weixinpojie\weixin-key\.build\candidates\20260912-recovery-r4\source，154文件。
- 清单：D:\weixinpojie\weixin-key\.build\candidates\20260912-recovery-r4\source-sha256.json。
- 清单 SHA256：CA24560C4671BD468734FC0F703669CC978D0827A6339EC5910F0888891CB22D。Owner 最终核对工作树/source 均154/154匹配，代理也核对154/154且无额外文件。
- 证据：D:\weixinpojie\weixin-key\.build\validation-20260912-recovery\r4\evidence-binding.json。
- 证据 SHA256：20F5BE7DBA3097C44855107DA1E88D5632578E44454774F88311C24A2D170A90。

| 内部产物（未发布，不是可用性承诺） | SHA256 |
|---|---|
| D:\weixinpojie\weixin-key\.build\candidates\20260912-recovery-r4\weixin-key.exe | 1EDC426046E031C8F354E2B8FDB8A96548213AAF16F5636F1A9235E55B7A91F8 |
| D:\weixinpojie\weixin-key\.build\candidates\20260912-recovery-r4\weixin-key-gui.exe | 19CD946298B5D172F764DE931963642C90B764D9BC5715087C94EF1553D4C8BF |
| D:\weixinpojie\weixin-key\.build\candidates\20260912-recovery-r4\weixin-key-darwin-arm64 | 82296BB5B308097AE0AD2DE5B9360E9827C20ABC294361372F8B09C2EBA46200 |

CLI版本：0.2.0-dev-recovery-r4-20260912。Darwin仅CLI交叉构建，无Mac运行、Keychain、GUI/app/dmg证据。

## 本轮实际实现

1. 恢复 Get/SetContext 失败不再释放带残留 DR/TF 的目标。记录原上下文、恢复确认、自有 suspension；所有修改线程读回验证前不 Resume/Continue/detach。失败写若实际已应用，通过新 readback 核对，不盲重写。
2. detach 持续失败保留同一个锁定 OS 线程和恢复句柄。两次带新状态核对的尝试后可见等待；显式重试/新观察退出或脱离才继续，不无限盲写。确认退出/脱离与清理完成分开处理。
3. 已 detached 的旧 pending 不再证明停止，必须另取自有 suspension；暂停失败保留恢复 owner，不读写该线程 context、不继续失效事件。
4. CaptureRecovery 属于单一作业，零值可用、重试有界合并，通知 panic 不传播破坏 owner。等待通知窗口中的退出变化不会被轮询基线吞掉。
5. GUI 增加恢复阶段、重试按钮及安全退出等待窗口，错误窗口用 Windows 原生提示兜底；Job.Done 只在清理/join 后关闭，退出后禁止新作业。State直接查询恢复状态，晚到Progress不能隐藏恢复。
6. CLI setup 的 Ctrl+C 请求取消；恢复等待时再按 Ctrl+C 请求一次恢复而非新采集。GUI、CLI、setup源头均保留 ErrCaptureCleanup/原始错误，不被取消覆盖。不能拦截任务管理器强杀、终端关闭、系统退出或进程崩溃。
7. 同线程先在工具自建隐藏惰性 bootstrap 上建立 KillOnExit(false)，准备成功且回收后才进入账号 WM_CLOSE/目标 attach/launch；准备凭据绑定线程。没有环境变量或CLI豁免主动安全门。
8. launch 使用 CreateProcess 的固定 PROCESS_INFORMATION，恢复 owner 持有精确进程对象到清理完成，避免裸PID重用或启动后失去恢复句柄。

## 独立 Review：仅既定子范围 PASS

归档：D:\weixinpojie\weixin-key\.build\validation-20260912-recovery\r4\review-result.md（Owner按代理返回归档）。

| 发现 | r4结论及边界 |
|---|---|
| 原P1-1：事件读取失败后错交自有异常 | 保持旧限定PASS：cleanup后来能读context时，在清DR6/TF前重判；原错误保留 |
| 原P1-2：持续恢复失败仍释放目标 | 既定故障场景限定PASS；不能替代所有可能生命周期验证 |
| 原P1-3：持续detach失败丢owner | 既定场景限定PASS：保留同OS线程、句柄、恢复状态 |
| R1/P1：detached后旧pending仍当停止 | r3复审FAIL；Owner真实red→green，r4限定关闭 |
| R2/P2：setup取消覆盖cleanup原错误 | r3修复，r4保持；新增测试证明原配置不存在时不创建，不冒充已有配置保旧测试 |
| R3：等待通知期间退出被漏掉 | 真实red→green，限定关闭 |
| R4：晚到Progress隐藏恢复 | State直接核对controller，静态限定关闭；完整GUI交互未验收 |
| 同线程bootstrap与双失败 | 限定PASS；生产startupAddr/profile和实际微信生命周期不在范围 |

代理未独立执行测试。**这些PASS不授权启用主动捕获**，不是全产品或全部生命周期无条件认可。

## 执行检查（r4冻结source）

环境：Go1.26.5 / Windows amd64 / CGO_ENABLED=0。全套测试隔离 USERPROFILE/HOME，清空 WECHAT_CLI_* / WX_MCP_* 等宿主覆盖，保留既有Go缓存；无真实配置依赖。

| 实际检查 | 结果 |
|---|---|
| go test ./... -json -count=1 -timeout=180s | exit0，10测试包通过，4测试skip；GUI/testutil无测试不计通过 |
| go vet ./... | exit0 |
| gofmt -l（仅冻结source的.go） | 空输出 |
| Windows CLI + GUI build（trimpath，GUI windowsgui） | 两者exit0 |
| GOOS=darwin GOARCH=arm64 CLI交叉构建 | exit0；未在Mac运行 |
| 新exe --version / help | exit0，版本匹配 |
| 新exe setup --pretty（合成加密库，restart/hook env打开） | 预期exit1、明确安全拒绝、未创建配置、合成源DB hash不变 |
| git diff --check | exit0 |

4项skip：TestQueryReadsRealColumn、TestWriteConfigToRootPinsOpenedParent、TestRelocateAgainstRealWeixinDLL、TestExecutableSearchDirsIncludesResolvedSymlinkDir。没有-race运行证据，不能将并发合成测试说成race detector通过。

原生证据均只针对自建fixture：
- 自建CREATE_SUSPENDED Sleep线程原DR/TF恢复和自有暂停平衡。
- 自建cmd /d /q /c exit 0收到11个debug事件，CREATE_PROCESS→EXIT_PROCESS，退出事件Continue后exit0。fixture的启动断点策略不证明生产profile。
- bootstrap准备后第二个自建debug child在实际owner OS线程退出后仍存活、detached；仅恢复创建暂停一次后正常exit0。
- policy setter失败＋持续detach失败不返回readiness；显式fixture重试后收回原错误并回收自建child。

## 失败记录和GUI证据不混淆

- recovery-r1冻结遗漏两个既有embed资产（LICENSE.sqlite3mc、sqlite3mc_x64.dll），全仓/vet/build真实exit1；r2新候选补齐，不覆盖r1失败。固定DLL sha5030DECC6D914539E3B9B7E28AA4F6DE1E7161DAC6FD4B21EB02E6754D9B175E。
- 根证据目录保留recovery-red、session-first旧调用预期失败、exit-transition-red、setup-cancel-red、detached-pending-red及各轮green。r2/r3全仓也通过，但不替代最终r4证据。
- r3 GUI正确显示中文及隔离目录回填；第一次SW_HIDE没有可操作窗口，仅终止已核对身份的自建空闲测试进程。第二次Normal启动可见，ComputerUse操作因用户活动保护被拒，不计自动点击成功。
- 用户提供的截图显示自行点击合成账号启动/确认后安全门拒绝。该账号wxid_packaged不是本人微信，源为 D:\weixinpojie\weixin-key\.build\validation-20260912-control\fixture-data\wxid_packaged_0001，输出仅 D:\weixinpojie\weixin-key\.build\validation-20260912-recovery\r3\gui-smoke\synthetic-output。
- 合成DB前后hash均89393469250D55D00A8DF27095955E50294082C65D0D888132FB47E847B980C9。该截图不是导出成功、完整GUI测试，也不能当r4窗口验收。r3测试窗口留给正在查看的用户正常关闭，未强制关闭其交互窗口。

## 保留的旧基线与限制

- control-r4：135文件清单15295AAB635EA264992F28FF0DE45FCF344B1D92CDE7A021F52A05A96AC0C3FD；debug-r2：139文件清单986DAEB13140ED0B877BDA767CAA1AEA7EEADB941F3D022EC08F061FECF273A7。它们只证明各自候选，不替代r4。
- 保留安装发现、多安装拒绝、完整进程枚举错误传播、账号文件持有者/PID创建时间身份固定、schema5当前用户DPAPI/私有DACL/受保护迁移备份/失败保旧。非Windows秘密持久化未实现。
- 保留离线优先、独立等待预算、原生加密backup+全页HMAC、SQLite/WAL一致快照、源不变、发布journal/恢复、账号输出隔离、hash绑定历史/冲突副本及物理路径检查；不把这些合成证据说成真实微信导出完成。
- 历史control默认配置测试曾只隔离HOME而误读真实默认配置，未输出/写回；后续都同时隔离USERPROFILE/HOME。本轮未重现。禁止对整个仓库递归gofmt -w，旧冻结候选不得改写。

## 新授权后的真实核验（与recovery-r4合成证据分开）

- 真实密钥未取得/验证，真实数据库未解密，不能把下列代码一致性检查称为解密成功。
- 2026-09-12T18:20:04Z：以只读权限固定PID43288，创建时间2026-09-12T12:04:44.0077783Z，核对完整exe和Weixin.dll路径、基址0x7FF838950000、映像大小198553600。固定模块SHA与旧静态研究一致。
- 只读取3个已研究的可执行MEM_IMAGE范围共583字节（候选函数574、长度getter5、数据getter4），均与固定SHA磁盘映像一致。读取前后核对进程/模块/磁盘身份，没有读堆/栈/材料、没有调试或目标写入。该证据不证明材料种类、账号关联或生产捕获ABI。
- 2026-09-12T18:21:26Z：配置概要schema4、缓存0、KeyEntries0、无passphrase；目标账号匹配，读取前后config SHA不变。仅枚举账号db_storage元数据：22个DB，未读其内容。
- 本地诊断源码/测试/产物/去敏证据：D:\weixinpojie\weixin-key\.build\validation-20260912-live-r1。module-probe-live.json记录代码比对，real-metadata.json记录配置概要与DB计数。诊断首次编译失败（x/sys常量未导出）日志保留，修正后2个测试及7个边界子用例通过，vet/build exit0。没有修改生产源码、配置或真实输出。
- 已复用获授权只读代理审查冻结r4被动路径，确认不宜直接跑完整setup：取消内循环、候选集合上限、配置副作用/阻塞锁、未配置signature错误残留四项仍OPEN。没有运行该真实setup；新增独立probe绕开它，不冒称已修复它。

## 最新独立被动probe与真实结果

**候选与范围**
- 生产新增仅D:\weixinpojie\weixin-key\internal\wxkey\passive_probe_windows.go及对应_test.go；原recovery-r4的154文件未变。没有更换CLI/GUI/dist/桌面文件，没有commit/push/release。
- r5冻结：D:\weixinpojie\weixin-key\.build\candidates\20260912-passive-probe-r5\source，157文件（原154+上述两文件+仅候选中的诊断runner）。
- 清单SHA256：347E6AEE4572EA2A68F816EE194AFD769E86C99C034D7C9F923A0B313AB4D42F。
- 同级passive-probe.exe SHA256：DA58ECDC8C9992832F962A37942DAB6F4636A9BFE3E0D7B85EE1D98D557AD3EE。它不是最终桌面导出交付。
- 单一路径只验证D0结构候选中的raw enc_key；不猜口令，不走其他路由/调试/重启，不读写默认配置，不返回或保存候选。
- 显式PID/创建时间/exe/RM账号文件owner绑定；只读MEM_PRIVATE+PAGE_READWRITE。单worker、512MiB读取请求、65536查询、65536结构、4096候选、64DB上限。90秒是协作取消，外层只对工具自己的探测子进程有150秒等待/终止边界，不控制微信。
- 根目录通过不跟随reparse的原生句柄取真实ID并持续保留；与os.Root实际句柄及当前路径ID比较。目录和所有验证页读取都相对于所保留root。批次128、累计65536目录项、深度8；源复核每次读取前后及成功判定前检查取消。

**独立审查与检查**
- r4 probe静态限定PASS，r5仅结构计数上限8192→65536的数字增量限定PASS；代理每次核对157/157，但未独立执行测试，也没有读取真实数据。
- 历史限定发现和整改：D:\weixinpojie\weixin-key\.build\validation-20260912-live-r1\probe-review-history.md。目录junction测试实际失败日志保留；其中名称带green的早期日志也为exit1，不能按文件名算通过。另补同路径普通目录替换的真实自建测试。
- r1/r2冻结曾跑全仓test，各10包pass/4skip；r1 runner有gofmt失败，r2修正。r3/r4冻结跑wxkey全包test，各exit0、2skip（真实DLL和符号链接环境测试），全仓vet、诊断build、gofmt通过。
- 最终r5只改一个预算数字：实际跑15定向测试+4预算子例，19个pass/0skip；全仓vet、诊断build exit0。它不是r5全仓test重跑或race detector证据。测试覆盖跨块、HMAC、取消、候选去重、读失败、数目/目录批次/深度限额、root/子junction和同路径替换拒绝。

**真实执行，不是合成账号**
| UTC时段／候选 | 实际结果 |
|---|---|
| 19:04:26–19:04:28／r4 | 22库、169候选、3718次页一HMAC、0命中；读取请求157114656字节，8192结构上限耗尽，exit1；不是微信异常或“内存无密钥”证明 |
| 19:09:03–19:09:05／r5 | 604区域、18411结构、505候选、11110次页一HMAC，0/22通过；读取请求284654488字节，无读取失败；正常完成该单一路径，exit0不是解密成功 |

- 两次主进程PID43288/创建时间2026-09-12T12:04:44.0077783Z保持不变，22库加密验证页前后相同；配置hash仍2E97C9C8BA421CB603B15C4CF0926ABAEB7E127CC6DFA128F85C0BAA5C648AF9（原概要缓存0/材料0）。
- 没有写源库、配置或聊天导出，没有重启/关闭/调试微信，没有输出原始候选或内存转储。确实读过有界进程内存，不能声称完全没接触可能含私密内容的缓冲区；没有解析、显示或保存聊天正文。
- 两个诊断子进程40588/47284均正常退出其运行流程，外层超时均false，无后台采集。r5“no-match-in-observed-regions”只覆盖本次raw D0区域，不排除其他材料类型、区域、布局或生命周期中存在有效密钥。
- 最新真实报告：D:\weixinpojie\weixin-key\.build\validation-20260912-live-r1\passive-live-r5\report.json；同目录execution.json记录时间、二进制hash和配置/目标身份核对。r4结果在同级passive-live-r4，未覆盖。
- 最终证据绑定：D:\weixinpojie\weixin-key\.build\validation-20260912-live-r1\evidence-binding.json，SHA256=FD38F9380C8E03E2C2DB33294F582C29B3DB1BB08032264A1719D98C57921B20。Owner最终核对工作树和冻结157/157、gofmt及git diff --check通过；代理已完成并关闭，可必要时resume。

## 下一关键路径／未完成项

1. **材料获取仍是当前阻塞点**：D0 raw真实诊断已完成但未命中，不原样重跑或机械增大预算。下一步转向有上下文的其他材料/采集边界，完成生产可信startupAddr、材料ABI及账号关联；主动门继续拒绝。已有本人本机授权不再重复询问。
2. 本人授权真机登录/冷启动/热缓存/取消/错误闭环。当前无真实账号采集成功、可交付桌面一键导出的证据。
3. 真实schema、群发送者、分片、压缩富消息、实际附件、迟到与修订、资源上限及真实性能。not-extracted不算附件完成。
4. staging/缓存整体DACL、路径竞态/断电持久性和数据边界独立Review、GUI完整交互/高DPI/干净安装验收。
5. Mac真机/架构环境未提供；原生采集/权限/Keychain/GUI/加密快照/app/dmg/安装运行未完成。CLI交叉编译或手工导入不能替代。

## 用户路径（已读配置概要、加密验证页、固定代码和有界私有内存，未解密或写真实输出）
- 账号：E:\软件\xwechat_files\wxid_6lm1pwjnrbnr12_4d65。
- 预期输出：C:\Users\admin\Documents\微信导出。
- 安装exe：D:\软件\weixin\Weixin.exe；DLL本次复核SHA256=8F7406A8A465E851EE10EAECCCB572D0E5B4E00D480C770938C714C396097B74。
- E:\软件\xwechat\_files不是当前核实的账号路径。GUI本次fixture路径也不得替代真实账号路径。
