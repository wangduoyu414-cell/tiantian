# 微信本地数据读取技术演进与项目续建建议

> 后续本机实施更新（2026-09-13T05:44:43Z）：下文保留研究时的原始基线，不再代表当前“尚未取钥”的状态。还原路线已真实验证22/22库、主库整库认证；22个派生raw key已进入当前用户DPAPI缓存，另起无进程程序验证复用，并完成主库17列结构核验。当前应转向发送者/压缩内容及标准导出，不重复增加同一取钥扫描。最新证据与未完成项以 `D:\weixinpojie\weixin-key\docs\IMPLEMENTATION_STATUS.md` 为准。

### 后续压缩标记研究线索（未当成本机解码证明）

公开Tencent/WCDB固定提交`39dd797099d41cf1953d5668acd8cb608016c599`的`src/common/core/compression/CompressionConst.hpp`定义`WCDB_CT_`列前缀，并把压缩类别与原始TEXT/BLOB类别合并进整数标记：类别为标记右移一位，最低位区分原始类型；公开类别包含未压缩、ZSTD字典、ZSTD普通模式。按该公式可推得标记0/1、2/3、4/5对应上述三组；这只是公开实现规则，不是本机记录分布或闭源客户端必定完全相同的证明。

来源：`https://github.com/Tencent/wcdb/blob/39dd797099d41cf1953d5668acd8cb608016c599/src/common/core/compression/CompressionConst.hpp`。

下一步须本机有界统计实际值类型/标记及必要字典可用性，再选解码路径；声明TEXT不等于SQLite运行时值必为未压缩文本。未经这些验证，不应直接将非UTF-8字节转string并当成已解析正文，也不应以列名推定所有内容都是同一种ZSTD编码。

## 1. 结论

**当前最值得增加的，不是另一个“大范围扫内存工具”，而是版本绑定的材料还原与验证工具。** 已有公开实现把“内存中的32字节候选”先还原为口令材料，再派生数据库密钥；本项目现有的 `enc_key/passphrase` 分型尚未包含这层还原。这个方向与最近一次只验证 raw `enc_key` 的诊断具有不同的失败条件，应优先核验，但目前没有本机微信4.1.13.65的成功证据。[^12][^13][^14][^L1][^L2]

建议采用以下顺序：

1. **先排除材料表示与验证器的问题**：静态核对当前模块是否存在有证据的材料转换；用独立数据库引擎校验合成向量，验证转换前后类型没有混淆。
2. **只进行有新假设的有限被动验证**：复用现有受限读取入口，增加直接口令与有版本依据的材料还原，不再机械放大相同 raw 扫描。
3. **被动证据不足时再推进生命周期观测**：先证明目标身份、调用边界、参数类型和恢复能力，再验证登录／开库期间的短生命周期材料。静态定位到函数不等于采集入口已经安全、正确。
4. **一旦获得真实有效材料，立即完成一条真实导出链路**：材料验证 → 一致快照 → 实际消息 → Markdown → 二次缓存复用；不要继续把主要投入放在与这条链路无关的框架扩展上。

**“高概率”目前只能用于判断研发投入的价值，不能用于承诺取钥成功率。** 没有覆盖版本、架构、登录状态、账户类型和失败样本的统计数据，不能声称任何工具在4.1.13.65上有80%、90%或“通用必成”的成功率。公开 issue 中同时存在失败报告和进入更下游读取阶段的报告，不能把其中任一条推广成所有环境的结论。[^15][^16]

本报告是研究与续建建议，不是新的实施完成报告。核验时本机日期为2026年9月12日，UTC已为9月13日；GitHub采用UTC记录时间。没有执行上游采集脚本、安装其二进制、操作微信或重新读取微信进程内存。

## 2. 证据口径

下文区分四类证据：

- **实现事实**：在固定提交的源码中看到了相应行为，但不代表本机可运行或已成功。
- **作者／使用者报告**：原始文档、提交说明或 issue 提供的实测描述，未经本项目独立复现。
- **本机实测**：绑定本项目候选与运行报告的结果；其结论只能覆盖实际测试范围。
- **分析与建议**：基于上述事实作出的判断，不冒充微信官方技术说明。

本文所说的“对抗演进”，只讨论本人本机数据读取所面对的加密、表示、生命周期、兼容性和生态变化。**符号缺失、编译优化、内存清理、系统签名保护，以及仓库下架，不能一概认定为专门针对某个工具的技术反制。** 更不能把这些现象解释成AES或SQLCipher已经被密码学破解。SQLCipher本身公开支持口令派生与直接原始密钥两种输入。[^1][^2]

## 3. 可核验的演进时间线

| 时段／样本 | 原始证据 | 可以得出的结论 | 不应得出的结论 |
|---|---|---|---|
| 2020年，Windows微信2.8.0.121样本 | `zhimian/decrypt-PC-WeChat-db` 明确限定版本，说明4096字节页、SHA-1、64000轮派生和48字节保留区。[^4] | 早期读取工具已围绕特定客户端布局和已知加密配置工作。 | 这组参数适用于所有平台或所有旧微信；它也不等同于SQLCipher 3的全部默认值。 |
| 2024年起的3.9系列适配记录 | WeChatFerry记录3.9.10.19、3.9.12.17、3.9.12.51等适配，源码含大量模块／对象偏移。[^5] | 固定版本、固定ABI的客户端内接口调用是一条重要路线。 | 3.9的偏移、发消息接口或Hook能力可以直接迁移到4.x导出。 |
| 4.x可检查源码样本，2026年4月至6月 | 两份实现扫描SQLCipher形式的十六进制材料，并以数据库盐与页HMAC验证。[^6][^7] | 某些实现依赖内存里仍存在可识别的 raw-key 表示；这些策略可能高度重合。 | 多换几个语言或GUI，就等于增加了独立取钥机会。 |
| 2025年10月20日 | chatlog、PyWxDump作者分别发布移除通知。[^8][^9] | 原仓库不再提供原有源码与维护，旧教程的依赖入口已失效。 | 是新密码算法导致项目技术上不可行；通知等同于法院判决。 |
| 2026年1月 | GitHub公开的1月8日腾讯投诉材料列出 `0xlane/wechat-dump-rs`；该仓库API当前返回451，阻断记录时间为1月29日。[^10] | 源码供应与维护还受到发布平台／投诉流程影响。 | 1月8日就是所有相关仓库被阻断的日期，或所有分支有相同许可和安全性。 |
| 2026年6月5日至6日 | 一份Windows研究日志从“扫描失败、认为材料不可读”，修正为读取API、目标状态和材料生命周期等问题；作者随后报告在特定ARM模拟x64环境跑通。[^11] | 失败计数必须连同观察器是否有效、目标是否真正运行一起解释。 | 该日志证明所有新版密钥都不在内存，或它的成功能外推到本机。 |
| 2026年6月25日 | WeChatDataAnalysis提交接入V4内存扫描和DLL辅助材料；当前实现包含还原、派生、HMAC及消费者侧规范化。[^12][^13][^14] | “观察到的候选”不一定就是可直接使用的 `enc_key` 或口令。 | 某个32字节常量就是跨版本通用密钥，或当前本机一定采用相同变换。 |
| 2026年7月至8月的仓库状态 | Thearas项目当前树仅余README；WeFlow当前树仅有README与图片，其8月9日README声明移除取钥、解密功能。[^17][^18] | 需要检查实际文件树，不能只看仓库描述、星数和下载入口。 | 所有Mac路线或所有WeFlow相关衍生实现都已消失。 |
| 2026年9月，Windows4.1.13.65 | `she-love-me#31` 报告多个组件超时或候选验证失败；另一个项目的 `#143` 报告收藏schema／UTF-8读取异常。[^15][^16] | 当前小版本仍有材料兼容与下游数据契约问题；故障层次必须区分。 | 前者证明全部方法失效；后者证明已成功完成首次取钥或全部数据导出。 |

这里故意没有给出一个“某日之后所有微信从raw模式统一切到password模式”的断言。部分README这样概括4.1.10.31前后的变化，但没有提供足够的跨平台、跨构建样本支持这个全称命题。应把它作为版本假设，而不是永久分界。[^19]

## 4. 演进的实质：七条相互独立的变化

### 4.1 加密算法与密钥取得是两件事

SQLCipher的公开设计包含AES页面加密、独立的页认证，以及基于每库盐的口令派生。相同口令可以对不同盐产生不同的加密密钥；页HMAC密钥也不是数据加密密钥。直接密钥输入和口令输入的含义不同，不能因为两者都编码为64个十六进制字符就混用。[^1][^2]

对本项目来说，核心困难不是“重新实现AES”，而是识别：

- 观察到的是可用口令、已派生 `enc_key`、包装／混淆材料，还是不相关的随机字节；
- 该材料属于哪个账号、设备、数据库盐及加密配置；
- 它是在当前时刻仍被保存，还是只在登录或开库期间出现。

**分析：** 在类型判断不正确时，更快的扫描与更多CPU只能更快地验证错误假设。

### 4.2 从固定偏移到语义定位，但语义定位仍需验证

WeChatFerry的版本记录与偏移表体现了“为一个客户端ABI做完整适配”的路线。它能说明维护成本来自什么位置，却不是本项目4.x材料采集的直接兼容包。[^5]

Tencent公开WCDB的 `CipherConfig` 源码则显示：对象既有原始输入材料，也有保存派生后 raw-key 表示的路径，之后可能释放原始输入。这个公开实现可以帮助理解对象生命周期，但**不能证明本机闭源Weixin.dll逐字采用同一结构或偏移**。[^3]

**建议：** 从“字符串命中即认为找到入口”，提升为“文件身份 → 可执行区段 → 交叉引用 → 函数边界 → 参数数据流 → 材料类型 → 数据库验证”的证据链。Ghidra适合辅助数据流理解；Capstone适合指令级核对。两者是分析工具，不是微信密钥提取器。[^25][^26]

### 4.3 新缺口是材料还原，不只是多一种扫描

WeChatDataAnalysis固定源码的链路是：

```text
版本相关辅助材料
          ＋
内存结构候选
          ↓
候选还原为口令材料
          ↓
按目标数据库盐派生
          ↓
页一HMAC验证
          ↓
消费者取得还原后的规范材料
```

其中 `key_v4.py` 实现候选转换与派生验证，`dll_key_scan.py` 从模块代码产生辅助材料候选，`key_service.py` 实际调用并处理返回值，不是三个互不相干的示例。引入该路线的提交日期是2026年6月25日。[^12][^13][^14]

与此对照，本项目的 `normalizeMaterialHex` 只是整理编码；验证器把原候选直接按raw或passphrase解释。盐的 XOR `0x3a` 属于MAC派生，和上述候选还原不是一回事。只读代码核对确认本地没有同类版本绑定还原链路。[^L2]

**分析：** 这是本次发现中最直接、最值得优先验证的差异。它能解释“已有大量结构候选，但直接raw验证全失败”的一种可能性；但也可能本机根本不使用这层表示，所以第一步应该是静态核验适用性，而不是直接增加一轮昂贵派生。

**接入边界：** 不复制上游的默认账号选择、全局状态、无界候选集合或秘密输出方式。该仓库元数据没有明确的统一许可证标识，源码可读不等于已经具备任意复制和分发许可。先核实具体文件许可；本项目应按清楚的材料合同做最小实现，而非搬入整套扫描器。[^12][^13][^14]

### 4.4 稳态扫描与生命周期观测有不同的失败条件

一份Windows攻关记录的价值，不在其“唯一办法”“一击必成”等措辞，而在它保留了前后修正：早期的扫描失败并没有正确排除观察器故障；后来又区分了派生输入、数据加密密钥和认证材料。其公开脚本仍有强制终止所有同名进程、粗略挑选PID、按填充字节推测函数入口、只以部分明文头判定等行为，不适合直接作为本项目生产依赖。[^11]

另一份Windows主动流程源码选择在WCDB配置阶段观察材料；本项目也已有定位、启动期控制与恢复设施。这意味着下一步应验证**材料到底在什么边界以什么类型出现**，而不是从头重写整套生命周期框架。[^20][^L2]

SQLCipher文档还说明，密码操作的内部内存会进行锁定与清理；`cipher_memory_security` 扩展清理范围且默认关闭。不能仅凭未命中就断言该选项已开启，也不能将锁页理解为外部读取必然失败。[^1][^2]

### 4.5 数据库解开后，仍有快照与数据语义问题

SQLite的WAL是已提交但可能尚未回写主文件的数据组成部分。直接复制一个运行中的主DB，不保证包含最新记录；随意拼接不同时刻的主DB与WAL，也不是一致快照。SQLite官方提供Online Backup API，但选用的加密引擎、权限模式和并发边界仍需本项目验证。[^21][^22]

本项目已经实现快照、WAL校验／应用及导出框架，下一阶段应验证其真实兼容性，而不是再新增一个相同职责的快照系统。特别要区分：

- 页面无法认证；
- SQLite结构无法完整打开；
- schema不支持；
- 压缩／二进制消息无法解码；
- 消息解析成功但附件缺失；
- 导出完成但增量状态未正确提交。

4.1.13.65的收藏读取issue说明，故障可能已经发生在字段解释层。它不是“缺少密钥”的同义词。[^16]

Tencent WCDB公开说明支持基于Zstd的字段压缩。因此，Zstandard解码与有界字段解析值得列入消息读取工具链；但不能将所有无法解码字段都自动送入Zstd，必须先识别真实schema、类型和压缩标记，并限制解压后的大小。[^36]

### 4.6 媒体格式有独立演进，DB密钥不能包办

可检查的媒体实现同时处理旧XOR格式以及带V1／V2标识的混合加密格式，还区分解密后为常规图片或特殊封装的数据。源代码中的“V1/V2/V3”命名也未必等同于微信主版本，不能混用。[^23][^24]

**建议：** 媒体模块以文件签名、长度边界和实际解码为依据，不以扩展名或数据库已解密为依据。区分“消息引用了附件”“本地存在附件”“已解密”“已成功解码”“只是缩略图”。缺少原文件或媒体材料时保留可解释占位与清单，不能把低清图伪装成原图。

### 4.7 平台安全与源码供应已成为兼容性的一部分

macOS的结果对系统版本、架构、微信build、签名、entitlements和调试器路径敏感。`wx-cli` 的README要求关闭SIP，而2026年9月12日的issue报告在修改过签名的特定4.1.13环境、SIP开启时成功；另一个项目则报告临时签名副本被系统阻止且恢复体验失败。这些记录不是互相简单推翻，而是测试条件不同。[^27][^28][^29]

WeChatDataAnalysis的Mac验证文档也明确区分真实样本、模拟测试和未复测build，并记录原生路径的权限／恢复限制。Windows的成功不能替代Mac验证，反之亦然。[^30][^31]

项目撤除同样是实质问题。chatlog、PyWxDump、WeFlow及 `wx_key` 的当前原仓库不能再当作完整、持续可编译的取钥依赖；不同包装产品也可能使用相同的底层wheel或DLL。需要检查**原生组件来源、源码覆盖、哈希、许可及实际兼容记录**，不能用上层产品名称计算独立策略数量。[^8][^9][^17][^32]

相关下架通知和投诉是作者／投诉方的陈述与平台记录，不是本文作出的法律认定。个人数据归档目的也不自动证明任意实现和发布方式均无风险。

## 5. 当前项目实际拥有的能力

本项目不能再被概括为“只差写一个GUI”，也不能说“解密机制已经完成”。准确状态如下。代码以r5冻结候选为基准，实施状态文档用于标识执行证据范围。[^L1][^L2][^L3]

| 层次 | 已有内容 | 尚缺的关键证据或能力 |
|---|---|---|
| 材料类型 | `enc_key/passphrase` 区分、按盐派生、resolver与缓存验证 | 版本绑定的包装材料还原；真实有效材料 |
| Windows秘密保护 | 当前用户DPAPI、私有DACL及迁移相关逻辑 | 真实采集到验证缓存再复用的闭环 |
| 被动策略 | SQLCipher字面量、结构候选、盐邻域、堆候选、变化区域优先等 | 不同路线在本机的独立覆盖证据；旧完整setup的取消、数量界限和错误传播问题 |
| 独立诊断 | 无配置副作用、有限读取与候选预算的D0 raw probe | 它仅验证raw，不覆盖直接口令及材料还原 |
| 主动设施 | 定位、账号／进程绑定、调试准备、恢复owner及限定合成测试 | 本机材料ABI、生产profile、真实捕获及生命周期闭环；安全门仍硬拒绝 |
| 数据库 | 单一SQLCipher4 profile、页HMAC、Go解密、WAL及sqlite3mc引擎集成 | 其他profile不能靠尺寸推定；独立实现交叉验证和当前真实库全链路验收 |
| 导出与GUI | 消息／发布／增量框架、Windows GUI | 附件仍有 `not-extracted` 占位，富文本／压缩内容解析未完成；实际消息、群发送者和GUI仍需真实验收 |
| macOS | 部分跨平台基础和外部入口 | 原生材料获取、Keychain、真实.app/.dmg、签名权限与运行证据 |

### 最近一次实测的正确解释

2026年9月12日的r5报告记录：

- 22个源库；
- 604个观察区域；
- 284,654,488字节读取请求，读取失败为0；
- 18,411个结构、505个候选；
- 11,110次页一HMAC校验；
- **通过的数据库为0个**，状态为 `no-match-in-observed-regions`。[^L1]

它证明：**指定范围、指定结构及直接raw解释没有找到有效材料。**

它不证明：所有内存都没有材料、所有passphrase都无效、所有需要还原的候选都无效，或未来生命周期观测必定失败。进程身份与源验证页稳定增强了该次诊断的可信度，但没有扩大其测试范围。

这些候选没有保存，因此不能声称可以直接对“原来的505个候选”离线补做另一模式。若推进新材料假设，需要一轮新的、限定范围的授权读取；本报告没有执行该读取。

现有引擎的单库只读事务备份，也不能自动证明22个库来自完全相同的全局时点。导出清单应说明每库快照边界及跨库关联的一致性限制；不要把逐库一致误报为跨库原子快照。

### 旧setup仍需修复，但不要让它吞掉关键路径

已知未关闭问题包括：部分内循环取消不充分、部分候选集合缺乏硬上限、诊断可能进入配置读写／锁副作用，以及未配置signature的错误可能污染后续成功结果。新probe绕开这些问题，不代表旧流程已经修复。[^L3]

建议先在独立诊断入口验证新假设；在将任何有效策略合入正式setup之前，关闭其实际会经过的这些问题，并运行影响范围内的回归。不能为了快而直接使用已知有问题的旧自动回退链，也不必为了探索一个材料假设先重审所有无关GUI或平台代码。

## 6. 工具与能力增量的优先级

### 6.1 推荐矩阵

“通用性”指跨版本可以复用的工程机制；“预期价值”指减少不确定性或完成链路的价值，不是保证取得密钥。

| 优先级 | 工具／能力 | 与现有项目的真实增量 | 预期价值与适用限制 |
|---|---|---|---|
| **P0** | **版本绑定材料还原探针** | 增加“观察材料 → 规范口令／enc_key”这层，而不只是增加raw/passphrase标签 | 本次最有针对性的新假设；上游存在实现，本机版本未验证 |
| **P0** | **独立加密配置与材料验证台** | 用现有sqlite3mc与Go结果交叉检查；必要时使用固定版本SQLCipher作开发验证器 | 有材料时高度确定；能排除自制验证器／参数共同错误，但本身不取钥 |
| **P0** | **静态profile诊断器**；Ghidra辅助、必要时Capstone核对 | 在现有定位上补充唯一性、函数边界、数据流、材料类型证据 | 无需运行微信即可缩小假设；不能把反编译结果当ABI实测 |
| **P1** | **有界表示覆盖探针** | 同一受限读取下区分字面量、结构、直接口令、已验证还原profile；逐项报告覆盖 | 比重复跑不同包装扫描器更有信息量；派生预算必须单独计量 |
| **P1** | **生命周期事件诊断器**；Procmon仅开发辅助 | 关联主进程代际、模块就绪、观察器就绪、目标库打开和候选类型 | 帮助区分未经过入口、时机已过、账号不符和观察器故障；Procmon不是取钥器，也不是开源依赖 |
| **P1，条件性** | **受控动态观察适配器**；Frida／平台调试器用于研发验证 | 对已证明的材料边界进行有限观测，复用现有恢复owner与安全门 | 比稳态扫描具有不同机会；侵入性与兼容成本更高，不作为默认“万能兜底” |
| **P1** | **真实快照／schema／压缩消息验收工具**，含按格式启用的Zstandard解码 | 在现有快照和解析上增加真实格式契约、缺失字段与二进制类型诊断 | 取得材料后立即提升导出成功率；不能反向解决取钥 |
| **P2** | **媒体格式分流与验真工具** | 将原文件存在性、材料可用性、解密和实际解码分层统计 | 图像、语音等需要各自证据；只得到文件头不算附件恢复 |
| **持续** | **兼容性／组件来源清单** | 绑定源码提交、原生资产哈希、许可、适用build、实际成功／失败范围 | 降低重复试错和供应链风险；不需要先建设大型插件平台 |

依据分别是材料转换实现、SQLCipher官方接口、静态分析工具说明、Microsoft事件诊断工具说明和Frida原始文档。推荐的是受控用途，不是默认执行其全套功能。[^2][^12][^13][^14][^25][^26][^33][^34]

### 6.2 材料还原探针应怎样设计

建议只增加一个小而明确的边界：

- **输入**：受限读取产生的候选及上下文、明确目标数据库、固定模块身份、可用的材料转换profile。
- **处理**：仅执行已验证适用的转换，然后交给现有typed verifier；转换失败与页认证失败分开记录。
- **输出**：材料种类、转换profile ID、目标库覆盖、验证结果和资源计数；不输出秘密本身。
- **缓存**：只把已经验证的规范材料交给resolver，不能把包装字节误存为 `enc_key`；版本辅助材料不自动成为账号秘密的替代品。
- **限制**：辅助材料候选数、内存候选数、数据库数、KDF次数、时间与取消延迟分别设界限；不能形成未预算的“候选×辅助材料×数据库”笛卡尔积。

选择少量代表库做快速验证时，不应先验假定“全部库共用一个口令”。一个直接 `enc_key` 命中单库，只获得该库覆盖；拟作为共享口令的材料，至少先用不同盐、不同业务角色的库交叉验证，再继续扩展到本次必需库清单。

### 6.3 独立验证台为什么比再添一个扫描器更重要

现在的0命中可以来自观察范围、材料表示、材料类型、数据库profile或验证器错误。若不同路线最终调用同一份有问题的验证器，它们会一起失败。

建议最少覆盖以下合成向量：

1. 同一口令、两个不同盐，派生出不同 `enc_key`；
2. 直接 `enc_key` 与相同字节按passphrase解释产生不同结果；
3. 包装材料只有经过正确profile还原才通过；
4. 错误profile、错误盐、损坏HMAC、截断页都应失败；
5. HMAC认证材料不能当作可解密材料；
6. Go路径与独立引擎对完整测试库结果一致。

这些是建议新增／补齐的检查，不是本轮已执行的测试。优先利用现有依赖；SQLCipher若作为额外开发验证器，应锁定版本和来源，不必随桌面产品再分发一套运行时。[^2][^L2]

### 6.4 暂不建议加入或作为默认路线的工具

- **不明来源的 `wx_key.dll`、封装器和网盘exe**：没有当前源码、许可、哈希与本机build兼容证据，不因“别人成功过”直接加载。某上层项目提供源码，不代表其原生捕获依赖也开源。[^32]
- **把Rust／Python／Node同类扫描器全部装上**：如果最终扫描同一种字面量或依赖同一个DLL，策略数量并未增加。[^6][^7]
- **无条件全内存口令暴搜、GPU KDF穷举**：当前没有证据支持这条投入；先解决候选表示和生命周期问题。
- **默认增加AES轮密钥恢复／YARA全内存猎取**：可以保留为研究假设，但没有本机目标实现与命中证据，优先级低于材料还原和明确边界观测。YARA适合表达模式，不会自动证明模式正确。
- **默认Hook系统密码API**：应用的加密提供者可能不是预想的系统接口。先核验实际调用路径，不能因为监听不到调用就说密钥不存在。[^1][^11]
- **降级微信、关闭SIP／杀软、替换官方安装文件**：不应作为产品默认路径，也不是本次研究所授权执行的动作。
- **发消息机器人、防撤回、客户端写库能力**：和只读归档目标不一致，不因项目名称相关就纳入。
- **云端视觉模型读取截图**：视觉导出是独立的替代路线，不应默默上传聊天截图。已检查的视觉项目需要额外评估模型调用与隐私边界；即使使用本地视觉，也只能覆盖可见内容，不能冒充完整历史／原附件导出。[^35]

## 7. 哪些上游值得借鉴，哪些不能直接依赖

| 项目／组件 | 本次确认的状态 | 建议用途 |
|---|---|---|
| Tencent/WCDB | 公开核心实现可读；`CipherConfig`包含输入材料／raw材料两种路径 | 语义、生命周期和格式理解；不是闭源微信ABI保证 |
| SQLCipher、SQLite | 有明确设计、原始密钥接口、完整性检查及备份文档 | 验证器与快照基准 |
| WeChatFerry | 3.9系列适配与固定偏移实现可读，仓库标识MIT | 历史演进对照；不作为4.1.13.65导出引擎 |
| `H3CoF6/wechat-decrypt-rs`、`328336690/wechat-decrypt` | 源码可读，仓库标识MIT；存在字面量／盐／HMAC与媒体处理 | 比较失败机制、媒体实现；没有本机版本独立成功证据 |
| WeChatDataAnalysis | 当前大量应用与部分捕获源码可读；Windows也有外部wheel，原生核心能力不等于全部可从源码重建；统一许可需核查 | **优先研究材料还原链路**、真实schema和Mac恢复边界，不直接复制整套运行时 |
| `tzwkb/wechat-decrypt` | 源码与修正过的Windows研究记录可读，仓库标识MIT | 学习观察器自检、材料类型与时序教训；现成脚本不满足本项目生产边界 |
| `labazhou2024/chatlog-keeper` | 主动／被动与恢复合同源码可读，仓库标识MIT | 比较生命周期合同；其“唯一”“全版本”类宣传不作为兼容证明 |
| `pandorafuture/wx-cli` | Mac源码可读，仓库标识MIT；README与最新issue的系统前提不同 | Mac条件矩阵与材料分型参考，不迁移其关闭SIP默认要求 |
| chatlog、PyWxDump、WeFlow、Thearas、`wx_key`原仓库 | 当前不提供原有完整采集源码 | 历史／维护状态来源；不是现成生产依赖 |

以上状态来自当前树、固定源码和对应说明，不是下载量排名。具体来源见文末；此表不是许可证法律审查或安全认证。[^3][^5][^6][^7][^8][^9][^11][^12][^17][^18][^19][^27][^32]

## 8. 当前项目的下一批工作

### 第一步：静态验证新材料假设

对象限定为已核实的本机Windows4.1.13.65模块身份。只查看磁盘代码及本项目已有静态定位结果，不重启或调试微信。

**产出：** 明确回答是否存在与上游相符的材料表示／转换；可能有多个候选时保留歧义，不选择“第一个看起来像的”。静态没有足够证据时记录为不适用或未知，不把上游pattern强行写入生产profile。

### 第二步：实现最小材料还原与独立验证

仅在第一步有依据时加入转换适配；同时补齐独立验证向量。沿用现有resolver和DPAPI，不另造秘密存储体系。

**验收：** 正确材料通过；错误profile不通过；包装材料不会误入缓存；取消和KDF预算有效。该阶段可以不触碰真实微信。

### 第三步：一次有新假设的受限真实验证

在合成检查与相关静态审查通过后，才使用新的受限诊断候选验证材料解释。诊断不走有配置副作用的旧setup自动回退，不保存原始内存或候选清单。

**判定规则：**

- 单库通过：只宣布单库材料命中。
- 多库通过：记录具体覆盖，不把局部命中称为账号全覆盖。
- 原候选不通过、还原后通过：形成材料转换适配的真实证据。
- 没有适用转换或全部不通过：结束该假设；不再仅扩大预算重跑。

### 第四步：必要时推进一个明确的生命周期边界

优先复用已有定位和控制代码，补齐本机ABI、材料类型、就绪时序和恢复证据。可以用自建小程序先验证观察器与异常恢复，但自建程序通过不代表微信入口通过。

**生产前提：** 版本／模块身份匹配、账号绑定、观察器已就绪、有限回调、秘密不外泄、取消和异常恢复可以验收。不能为了跑通而删除当前硬安全门。Frida若用于实验，也是辅助验证器，不是以另一个工具绕过门禁。[^34][^L2]

### 第五步：命中后立即完成真实窄闭环

```text
材料通过页认证
  → 必需库覆盖
  → 一致快照
  → 完整数据库检查
  → 真实消息解析
  → 一段可核对的Markdown导出
  → 扩展全部要求的消息／分片
  → 二次执行只走缓存和增量
```

保留五条验收底线：

1. 退出码0不等于取得密钥，SQLite文件头不等于整库正确。
2. 只读副本通过解密不等于快照包含最新WAL数据。
3. 一条文字消息不等于群聊、压缩消息和附件已完整支持。
4. 缓存有效的二次执行不能再扫描或重启微信。
5. 每个未覆盖库、无法解析类型、缺失附件和恢复失败都明确报告，不把“部分完成”显示成全部成功。

### 第六步：回到完整交付，不缩减Mac与媒体范围

Windows窄闭环完成后，再扩展真实附件、全部消息类型、大库资源边界及完整GUI交互。macOS继续作为独立平台验收：没有真机、签名权限环境及实际.app/.dmg证据，就保持未完成，而不是把导入明文或Darwin交叉编译当作替代交付。

## 9. 停止低信息量试错的规则

建议每个实验开始前写清三个问题：

1. **本次相对上次改变了哪一个假设？**
2. **成功和失败分别能排除什么？**
3. **到什么预算或观察结果就停止？**

具体应用：

- 同区域、同结构、同raw验证器再跑一次，信息量低；没有新证据不重跑。
- 增加直接passphrase解释，改变的是材料类型；增加版本还原，改变的是材料表示；增加开库期观察，改变的是生命周期。三者不能混成“扩大扫描”。
- 更换相同哈希的DLL，不是升级策略；更换包装GUI，不是独立验证。
- 编译通过、合成测试通过、静态审查通过、真实材料命中、完整导出成功必须分开记账。
- 研究结束或作业退出后不再后台采集；只有明确存在并可观察的作业，界面才能显示“运行中”。

## 10. 证据局限与最终判断

本机目前仍然没有有效密钥、真实解密或真实聊天导出成功证据。上游源码中的材料还原是一个明确新增的待验方向，不是替本机预先宣布成功。

这次研究改变了下一步的优先级：**先确认材料表示是否遗漏，再验证生命周期；先完成真实数据窄闭环，再扩展产品覆盖。** 通用能力应放在类型、验证、版本身份、资源界限和结果可解释性上；具体取钥策略保持明确适用范围。

没有独立执行上游工具、没有上游当前版本的大规模成功率数据、没有本机Mac验证，也没有审计所有依赖的完整许可证与二进制供应链。公开成功／失败报告仅按各自环境引用。

## Sources

原始来源地址保留如下。固定源码均使用提交SHA；未固定的issue、公告与官方文档属于核验时快照。公开源码与API摘要保存在 `D:\weixinpojie\.owner-supervision\research-20260912`，源码文件清单为 `D:\weixinpojie\.owner-supervision\research-20260912\sources-manifest.json`。本地工作树存在既有未提交改动，不能只用HEAD代表此次代码状态。

[^1]: Zetetic，SQLCipher Design，核验于2026-09-12。`https://www.zetetic.net/sqlcipher/design/`
[^2]: Zetetic，SQLCipher API，重点为raw key、explicit salt、`cipher_memory_security`、`cipher_integrity_check`。`https://www.zetetic.net/sqlcipher/sqlcipher-api/`
[^3]: Tencent，WCDB `CipherConfig.cpp`，提交 `39dd797099d41cf1953d5668acd8cb608016c599`，行31–75。`https://github.com/Tencent/wcdb/blob/39dd797099d41cf1953d5668acd8cb608016c599/src/common/core/config/CipherConfig.cpp`
[^4]: zhimian，decrypt-PC-WeChat-db README，提交 `e4a2c4d025bbafd7a4c16a8afc7ba3cc8d3b4205`，2020-07-06。`https://github.com/zhimian/decrypt-PC-WeChat-db/blob/e4a2c4d025bbafd7a4c16a8afc7ba3cc8d3b4205/README.md`
[^5]: lich0821，WeChatFerry README／offsets，提交 `0f5c60a034fcac234cabd000b49c9200defa7f7d`。`https://github.com/lich0821/WeChatFerry/blob/0f5c60a034fcac234cabd000b49c9200defa7f7d/README.MD`；`https://github.com/lich0821/WeChatFerry/blob/0f5c60a034fcac234cabd000b49c9200defa7f7d/WeChatFerry/spy/offsets.h`
[^6]: H3CoF6，wechat-decrypt-rs `db_decrypt.rs`，提交 `185a95eaef24b9afedfb6fdee0c6282dc2ed2700`。`https://github.com/H3CoF6/wechat-decrypt-rs/blob/185a95eaef24b9afedfb6fdee0c6282dc2ed2700/src/db_decrypt.rs`
[^7]: 328336690，wechat-decrypt `find_all_keys.py`，提交 `44427c45786feba4e5fc21625f7934528a83f624`，重点行71–84、126–190。`https://github.com/328336690/wechat-decrypt/blob/44427c45786feba4e5fc21625f7934528a83f624/find_all_keys.py`
[^8]: sjzar，chatlog《项目移除通知》，2025-10-20，提交 `7dad93d7b55a1f801bad3be16620083d2f998599`。`https://github.com/sjzar/chatlog/blob/7dad93d7b55a1f801bad3be16620083d2f998599/README.md`
[^9]: xaoyaoo，PyWxDump《项目移除通知》，2025-10-20。`https://github.com/xaoyaoo/PyWxDump/blob/master/README.md`
[^10]: GitHub公开投诉资料，Tencent，2026-01-08；以及仓库451响应中2026-01-29的DMCA阻断记录。投诉是投诉方主张，非裁判结论。`https://github.com/github/dmca/blob/master/2026/01/2026-01-08-tencent.md`；`https://api.github.com/repos/0xlane/wechat-dump-rs`
[^11]: tzwkb，Windows研究记录及 `extract_raw_key.py`，提交 `2199f64c29c4fd43be7e1fecac8197057eb775d7`。`https://github.com/tzwkb/wechat-decrypt/blob/2199f64c29c4fd43be7e1fecac8197057eb775d7/docs/2026-06-06-windows-raw-key-journey.md`；前序反例 `https://github.com/tzwkb/wechat-decrypt/blob/2199f64c29c4fd43be7e1fecac8197057eb775d7/docs/2026-06-05-windows-keyextract-findings.md`；脚本 `https://github.com/tzwkb/wechat-decrypt/blob/2199f64c29c4fd43be7e1fecac8197057eb775d7/scripts/windows/extract_raw_key.py`
[^12]: LifeArchiveProject，WeChatDataAnalysis `key_v4.py`，提交 `ff1650ff308915f6746ea99a89f0c120ff2c23c5`，行46–54、175–205。`https://github.com/LifeArchiveProject/WeChatDataAnalysis/blob/ff1650ff308915f6746ea99a89f0c120ff2c23c5/src/wechat_decrypt_tool/key_v4.py`
[^13]: 同项目 `dll_key_scan.py`，同一提交，行17–28、40–68、73–138。`https://github.com/LifeArchiveProject/WeChatDataAnalysis/blob/ff1650ff308915f6746ea99a89f0c120ff2c23c5/src/wechat_decrypt_tool/dll_key_scan.py`；引入提交（2026-06-25）`https://github.com/LifeArchiveProject/WeChatDataAnalysis/commit/9026756ff5f5dc6c5ef0f081f4e2f99d74db8247`
[^14]: 同项目 `key_service.py`，同一提交，重点行329–358、423–542、723–765。`https://github.com/LifeArchiveProject/WeChatDataAnalysis/blob/ff1650ff308915f6746ea99a89f0c120ff2c23c5/src/wechat_decrypt_tool/key_service.py`
[^15]: 863401402/she-love-me，issue #31，2026-09-12。使用者报告，未独立复现，也未证明各组件实现真正独立。`https://github.com/863401402/she-love-me/issues/31`
[^16]: LifeArchiveProject/WeChatDataAnalysis，issue #143，2026-09-09，核验时closed；关闭状态本身不证明本项目已兼容。`https://github.com/LifeArchiveProject/WeChatDataAnalysis/issues/143`
[^17]: hicccc77，WeFlow README及当前树，提交 `318d5ac4a968eaef9a5cbd7c663151b6ba2efcf0`，2026-08-09。`https://github.com/hicccc77/WeFlow/tree/318d5ac4a968eaef9a5cbd7c663151b6ba2efcf0`
[^18]: Thearas，wechat-db-decrypt-macos当前树，提交 `2278f6a83c61b1811244afbedd121bd0ac47f05b`，2026-07-13。`https://github.com/Thearas/wechat-db-decrypt-macos/tree/2278f6a83c61b1811244afbedd121bd0ac47f05b`
[^19]: labazhou2024，chatlog-keeper README.zh，提交 `8c22d7982ec071745d374996f48c1a6f8c4bbf0c`，行44–74为作者支持范围声明。`https://github.com/labazhou2024/chatlog-keeper/blob/8c22d7982ec071745d374996f48c1a6f8c4bbf0c/README.zh.md`
[^20]: 同项目Windows主动脚本与生命周期合同，同一提交。`https://github.com/labazhou2024/chatlog-keeper/blob/8c22d7982ec071745d374996f48c1a6f8c4bbf0c/chatlog_keeper/scripts/windows_wechat_get_key.ps1`；`https://github.com/labazhou2024/chatlog-keeper/blob/8c22d7982ec071745d374996f48c1a6f8c4bbf0c/docs/key-recovery-v1-contract.zh.md`
[^21]: SQLite，Online Backup API。`https://sqlite.org/backup.html`
[^22]: SQLite，Write-Ahead Logging。`https://sqlite.org/wal.html`
[^23]: H3CoF6，wechat-decrypt-rs `media_decrypt.rs`，提交 `185a95eaef24b9afedfb6fdee0c6282dc2ed2700`。`https://github.com/H3CoF6/wechat-decrypt-rs/blob/185a95eaef24b9afedfb6fdee0c6282dc2ed2700/src/media_decrypt.rs`
[^24]: 328336690，wechat-decrypt `decode_image.py`，提交 `44427c45786feba4e5fc21625f7934528a83f624`。`https://github.com/328336690/wechat-decrypt/blob/44427c45786feba4e5fc21625f7934528a83f624/decode_image.py`
[^25]: National Security Agency，Ghidra README，官方仓库，核验时默认分支。`https://github.com/NationalSecurityAgency/ghidra`
[^26]: Capstone项目，官方README，核验时next分支；只用于能力说明，不作为已安装版本。`https://github.com/capstone-engine/capstone/blob/next/README.md`
[^27]: pandorafuture，wx-cli README，提交 `2abe708f55bfe135539a385df856fdc58f97fc74`，行38–45、260–268。`https://github.com/pandorafuture/wx-cli/blob/2abe708f55bfe135539a385df856fdc58f97fc74/README.md`
[^28]: 同项目issue #20，2026-09-12，特定修改签名环境的使用者报告。`https://github.com/pandorafuture/wx-cli/issues/20`
[^29]: Rion-Wu-tech/wechat-intelligence-hub，issue #2，2026-09-07，特定系统策略失败的使用者报告。`https://github.com/Rion-Wu-tech/wechat-intelligence-hub/issues/2`
[^30]: LifeArchiveProject，Mac捕获说明与脱敏验证记录，提交 `ff1650ff308915f6746ea99a89f0c120ff2c23c5`。`https://github.com/LifeArchiveProject/WeChatDataAnalysis/blob/ff1650ff308915f6746ea99a89f0c120ff2c23c5/docs/MACOS_KEY_CAPTURE.md`；`https://github.com/LifeArchiveProject/WeChatDataAnalysis/blob/ff1650ff308915f6746ea99a89f0c120ff2c23c5/docs/macos-wcdb-key-capture-validation.md`
[^31]: 同项目issue #126，2026-08-29，macOS27不同观察器行为的报告，未独立复现其系统机制归因。`https://github.com/LifeArchiveProject/WeChatDataAnalysis/issues/126`
[^32]: ycccccccy/wx_key当前目录仅README；WeChatDataAnalysis固定提交的 `tools/key_wheels/README.md` 指向外部预编译wheel来源，`key_service.py`导入 `wx_key`。`https://api.github.com/repos/ycccccccy/wx_key/contents`；`https://github.com/LifeArchiveProject/WeChatDataAnalysis/blob/ff1650ff308915f6746ea99a89f0c120ff2c23c5/tools/key_wheels/README.md`
[^33]: Microsoft Sysinternals，Process Monitor官方说明；闭源开发辅助工具，未安装或运行。`https://learn.microsoft.com/en-us/sysinternals/downloads/procmon`
[^34]: Frida，JavaScript API官方说明；仅核实动态观测与拦截能力，不证明微信兼容性。`https://frida.re/docs/javascript-api/`
[^35]: Numaira-Technology，weclaw-cua README_CN，提交 `7e25226ed1cd836754c887504b2f51df4e59b825`。`https://github.com/Numaira-Technology/weclaw-cua/blob/7e25226ed1cd836754c887504b2f51df4e59b825/README_CN.md`
[^36]: Tencent，WCDB README，提交 `39dd797099d41cf1953d5668acd8cb608016c599`，行86，字段压缩能力说明。`https://github.com/Tencent/wcdb/blob/39dd797099d41cf1953d5668acd8cb608016c599/README.md`
[^L1]: 本机r5真实报告：`D:\weixinpojie\weixin-key\.build\validation-20260912-live-r1\passive-live-r5\report.json`。绑定记录：`D:\weixinpojie\weixin-key\.build\validation-20260912-live-r1\evidence-binding.json`。本轮复读报告确认0/22。
[^L2]: r5冻结源码：`D:\weixinpojie\weixin-key\.build\candidates\20260912-passive-probe-r5\source`；清单：`D:\weixinpojie\weixin-key\.build\candidates\20260912-passive-probe-r5\source-sha256.json`，SHA256 `347E6AEE4572EA2A68F816EE194AFD769E86C99C034D7C9F923A0B313AB4D42F`。材料核对：`D:\weixinpojie\weixin-key\.build\candidates\20260912-passive-probe-r5\source\internal\wxkey\keymaterial.go:48–98`；`D:\weixinpojie\weixin-key\.build\candidates\20260912-passive-probe-r5\source\internal\wcdb\kdf.go:34–51`；`D:\weixinpojie\weixin-key\.build\candidates\20260912-passive-probe-r5\source\internal\wcdb\page1.go:94–101`。只读审查提供咨询性代码事实核对，没有独立运行测试或读取真实微信数据。
[^L3]: 实施状态与既有失败记录：`D:\weixinpojie\weixin-key\docs\IMPLEMENTATION_STATUS.md`；`D:\weixinpojie\weixin-key\.build\validation-20260912-live-r1\legacy-passive-review.md`；执行检查点：`D:\weixinpojie\.owner-supervision\checkpoint.md`。本轮未更新实施完成状态。

### 本地能力核对的具体位置

- 策略主流程：`D:\weixinpojie\weixin-key\.build\candidates\20260912-passive-probe-r5\source\internal\wxkey\setup_flow_windows.go:182–215,232–364`
- 唯一DB profile：`D:\weixinpojie\weixin-key\.build\candidates\20260912-passive-probe-r5\source\internal\wcdb\profile.go:24–77`
- WAL边界：`D:\weixinpojie\weixin-key\.build\candidates\20260912-passive-probe-r5\source\internal\wcdb\wal.go:31–88,114–227`
- 引擎与备份：`D:\weixinpojie\weixin-key\.build\candidates\20260912-passive-probe-r5\source\internal\sqliteengine\engine_windows_amd64.go:27–34,172–284`
- 导出／附件占位：`D:\weixinpojie\weixin-key\.build\candidates\20260912-passive-probe-r5\source\internal\export\export.go:266–412`
- 消息解析：`D:\weixinpojie\weixin-key\.build\candidates\20260912-passive-probe-r5\source\internal\export\reader.go:151–174`
- GUI控制：`D:\weixinpojie\weixin-key\.build\candidates\20260912-passive-probe-r5\source\internal\guicore\guicore.go:394–447,567–619`
- 非Windows秘密存储缺口：`D:\weixinpojie\weixin-key\.build\candidates\20260912-passive-probe-r5\source\internal\config\secrets_other.go:10–14`
- 主动安全门：`D:\weixinpojie\weixin-key\.build\candidates\20260912-passive-probe-r5\source\internal\wxkey\capture_safety_windows.go:14–24`
- 受限raw探针：`D:\weixinpojie\weixin-key\.build\candidates\20260912-passive-probe-r5\source\internal\wxkey\passive_probe_windows.go:24–62,485–603`
