# WeChat 4.1+ 数据库密钥提取工具开发任务书

## 项目目标

开发一个 Windows 命令行程序，能够从运行中的微信 4.1+ 进程（以 4.1.11.55 为主要目标版本）中提取 SQLCipher passphrase，并自动解密本地数据库文件导出聊天记录。

程序要求：
- 单文件命令行工具
- 不需要微信版本签名即可工作（至少支持暴力扫描模式）
- 如果用户提供了 signature，则优先使用 signature 模式快速提取
- 最终输出解密后的 SQLite 数据库或聊天记录文本

---

## 技术背景

### 微信 4.1+ 加密变化

- WeChat 4.0 及之前：内存中直接保存 `x'<64 hex key><32 hex salt>'` 形式 raw key
- WeChat 4.1+：内存中只保存一个 32 字节 passphrase，真正的加密 key 需要派生

### 派生公式

```
enc_key = PBKDF2-HMAC-SHA512(passphrase, db_salt, iterations=256000, dklen=32)
```

其中：
- `passphrase` 是 32 字节
- `db_salt` 是数据库文件前 16 字节
- `enc_key` 是 32 字节，用于 SQLCipher 4 解密

### SQLCipher 4 参数

| 参数 | 值 |
|------|-----|
| 算法 | AES-256-CBC |
| KDF | PBKDF2-HMAC-SHA512 |
| 迭代次数 | 256000 |
| 页大小 | 4096 字节 |
| 每页保留 | 80 字节（IV 16 + HMAC 64）|
| Salt | 16 字节 |

---

## 开发阶段

### 第一阶段：基础框架搭建

#### 1.1 创建项目结构

```
wechat-key-extractor/
├── cmd/
│   └── main.go              # CLI 入口
├── internal/
│   ├── config/              # 配置和路径发现
│   ├── crypto/              # PBKDF2 + SQLCipher 验证
│   ├── db/                  # SQLite 解密和导出
│   ├── proc/                # Windows 进程操作
│   ├── scan/                # 内存扫描
│   └── verify/              # 候选 key 验证
├── go.mod
└── README.md
```

#### 1.2 初始化 Go 模块

```bash
go mod init github.com/yourname/wechat-key-extractor
```

#### 1.3 添加依赖

```bash
go get golang.org/x/crypto/pbkdf2
```

#### 交付物

- 可编译的空项目框架
- `go build ./cmd` 成功

---

### 第二阶段：Windows 进程内存读取模块

#### 2.1 功能需求

实现以下 Windows API 调用：

- `CreateToolhelp32Snapshot` 枚举进程
- `Module32FirstW / Module32NextW` 枚举模块
- `OpenProcess` 打开目标进程
- `VirtualQueryEx` 查询内存区域
- `ReadProcessMemory` 读取进程内存
- `CloseHandle` 关闭句柄

#### 2.2 接口设计

```go
package proc

// Process 表示目标微信进程
type Process struct {
    PID        uint32
    Executable string
}

// FindWeChatProcesses 查找所有 wechat.exe / weixin.exe 进程
func FindWeChatProcesses() ([]Process, error)

// ModuleRange 返回指定模块在目标进程中的内存范围
func (p *Process) ModuleRange(moduleName string) (start, end uintptr, err error)

// ReadMemory 从目标进程读取内存
func (p *Process) ReadMemory(addr uintptr, size int) ([]byte, error)

// ReadableRegions 遍历目标进程所有可读内存区域
func (p *Process) ReadableRegions(callback func(start, end uintptr) bool) error
```

#### 2.3 错误处理

- 进程不存在
- 权限不足（需要管理员权限）
- 模块未找到
- 内存读取失败

#### 交付物

- `internal/proc/proc_windows.go`
- 单元测试：能枚举本进程、读取本进程内存

---

### 第三阶段：密钥派生与数据库验证模块

#### 3.1 PBKDF2 派生

```go
package crypto

// DeriveEncKey 从 passphrase 和 salt 派生 enc_key
func DeriveEncKey(passphraseHex, saltHex string) (string, error)

// DeriveEncKeyFromBytes 从原始字节派生
func DeriveEncKeyFromBytes(passphrase, salt []byte) ([]byte, error)
```

#### 3.2 数据库 Salt 读取

```go
package crypto

// ReadDBSalt 读取数据库文件前 16 字节 salt
func ReadDBSalt(dbPath string) ([]byte, error)
```

#### 3.3 SQLCipher 第一页 HMAC 验证

不依赖外部库，用 Go 原生实现 SQLCipher 4 第一页验证：

```go
package crypto

// VerifyEncKey 验证 enc_key 是否能解密 dbPath 的第一页
func VerifyEncKey(dbPath string, encKey []byte) (bool, error)
```

验证步骤：
1. 读取第 1 页（4096 字节）
2. 取前 16 字节作为 salt
3. salt XOR 0x3a 得到 hmac_salt
4. `PBKDF2-HMAC-SHA512(enc_key, hmac_salt, 2, 64)` 得到 hmac_key
5. 计算 `HMAC-SHA512(ciphertext_region + pgno_LE32, hmac_key)`
6. 与页面末尾 64 字节 stored MAC 比较

#### 3.4 候选 passphrase 验证

```go
package verify

// VerifyPassphrase 验证候选 passphrase 是否能打开任一数据库
func VerifyPassphrase(passphrase []byte, dbs []string) (matchedDBs []string, encKeyMap map[string]string, err error)
```

#### 交付物

- `internal/crypto/kdf.go`
- `internal/crypto/verify.go`
- 用测试数据库验证正确性

---

### 第四阶段：内存扫描模块

#### 4.1 Signature 扫描模式

```go
package scan

// Signature 描述一个内存签名
type Signature struct {
    Name       string
    Pattern    []byte       // 字节模式
    Mask       []bool       // true 表示该字节必须匹配
    Offset     int          // 从匹配点到 key/passphrase 的偏移
    Length     int          // 读取多少字节
    KeyType    string       // "passphrase" 或 "rawkey"
    ModuleHint string       // 模块名，如 wechatwin.dll
}

// SignatureScanner 按 signature 扫描模块内存
type SignatureScanner struct {
    proc *proc.Process
}

// Scan 返回所有候选 key/passphrase
func (s *SignatureScanner) Scan(sigs []Signature, verifyFn func([]byte, string) bool) ([]Candidate, error)
```

#### 4.2 暴力扫描模式

```go
package scan

// BruteForceScanner 暴力扫描高熵 32 字节候选
type BruteForceScanner struct {
    proc *proc.Process
}

// ScanPassphrases 扫描所有可读内存，把每个 32 字节高熵块当 passphrase 验证
func (s *BruteForceScanner) ScanPassphrases(verifyFn func([]byte) bool, opts BruteOptions) error

type BruteOptions struct {
    ModuleOnly bool          // 只扫描 WeChatWin.dll 模块
    MinEntropy int           // 最小不同字节数，默认 20
    MaxCandidates int64      // 最大候选数，0 表示无限制
    Timeout    time.Duration // 扫描超时
}
```

#### 4.3 候选熵值过滤

```go
func hasHighEntropy(b []byte, minDistinct int) bool
```

#### 4.4 内存区域过滤

- 只扫描 `MEM_COMMIT` 区域
- 跳过 `PAGE_NOACCESS` 和 `PAGE_GUARD`
- 可选只扫描 `WeChatWin.dll` 模块范围

#### 交付物

- `internal/scan/signature.go`
- `internal/scan/bruteforce.go`
- 能扫描 notepad 或本进程做基本测试

---

### 第五阶段：数据库解密与导出模块

#### 5.1 解密数据库

```go
package db

// DecryptDB 用 enc_key 解密单个数据库到明文 SQLite
func DecryptDB(srcPath, dstPath string, encKey []byte) error
```

实现方式二选一：
- 调用 SQLCipher 库（需要 libsqlcipher.dll）
- 用 Go 实现 SQLCipher 4 解密逻辑

#### 5.2 批量解密

```go
package db

// DecryptAll 解密目录下所有数据库
func DecryptAll(dbRoot, outDir string, keys map[string][]byte) error
```

#### 5.3 聊天记录导出

```go
package db

// ExportMessages 从解密后的数据库导出聊天记录
func ExportMessages(dbPath string) ([]Message, error)

type Message struct {
    Time        time.Time
    Talker      string
    Content     string
    IsSender    bool
    Type        int
}
```

#### 交付物

- `internal/db/decrypt.go`
- `internal/db/export.go`

---

### 第六阶段：命令行界面

#### 6.1 子命令设计

```bash
wechat-key-extractor.exe scan          # 扫描并提取 passphrase/key
wechat-key-extractor.exe decrypt       # 解密数据库
wechat-key-extractor.exe export        # 导出聊天记录
```

#### 6.2 scan 命令参数

```bash
wechat-key-extractor.exe scan [flags]
  --mode brute          # 暴力扫描
  --mode signature      # 按 signature 扫描
  --sig string          # 环境变量式 signature
  --module-only         # 只扫描 WeChatWin.dll
  --timeout duration    # 扫描超时，默认 5 分钟
  --db-root string      # 数据库目录
  --output string       # 输出 key 文件
```

#### 6.3 decrypt 命令参数

```bash
wechat-key-extractor.exe decrypt [flags]
  --key-file string     # 包含 passphrase/key 的 JSON 文件
  --db-root string      # 数据库目录
  --out string          # 输出目录
```

#### 6.4 export 命令参数

```bash
wechat-key-extractor.exe export [flags]
  --plain-db string     # 已解密的数据库路径
  --out string          # 导出文件路径
  --format json|csv|html
```

#### 交付物

- `cmd/main.go`
- 完整的 CLI 使用说明

---

### 第七阶段：配置与结果持久化

#### 7.1 输出文件格式

```json
{
  "version": "4.1.11.55",
  "passphrase": "a1b2c3...",
  "passphrase_source": "bruteforce",
  "kdf": "pbkdf2-sha512-256000",
  "keys": {
    "salt_hex_1": "enc_key_hex_1",
    "salt_hex_2": "enc_key_hex_2"
  },
  "dbs": [
    "MicroMsg.db",
    "MSG0.db",
    "MediaMSG0.db"
  ]
}
```

#### 7.2 自动发现数据库目录

```go
func AutoDetectDBRoot() (string, error)
```

常见路径：
- `%USERPROFILE%\Documents\WeChat Files\[wxid_xxx]\Msg`
- `%USERPROFILE%\Documents\WeChat Files\[wxid_xxx]\db_storage`

#### 交付物

- `internal/config/config.go`

---

### 第八阶段：测试与验证

#### 8.1 单元测试

- PBKDF2 派生正确性
- SQLCipher HMAC 验证
- 内存扫描基本功能
- 熵值过滤

#### 8.2 集成测试

- 在已登录微信的电脑上运行 `scan`
- 验证是否能找到 passphrase
- 验证解密后的数据库能否打开

#### 8.3 性能测试

- 暴力扫描 100MB 内存耗时
- 候选验证速率

#### 交付物

- 测试用例
- 测试报告

---

## 关键技术点

### 如何确定微信进程

检查可执行文件名：
- `WeChat.exe`
- `Weixin.exe`

排除当前进程自身。

### 如何定位 WeChatWin.dll

用 `Module32FirstW/Module32NextW` 遍历目标进程的模块，匹配 `WeChatWin.dll` 或 `Weixin.dll`。

### 如何验证候选 passphrase

对每个候选：
1. 用 PBKDF2 对每个数据库的 salt 派生 enc_key
2. 用 SQLCipher 第一页 HMAC 验证
3. 只要有一个数据库通过，就是正确 passphrase

### 暴力扫描优化

- 先扫描 `WeChatWin.dll` 模块（通常 passphrase 在主模块里）
- 如果失败，再扩展到整个进程可读内存
- 使用熵值过滤减少候选数
- 用 goroutine 并行验证多个候选

---

## 风险与限制

### 技术风险

- 微信可能有反调试机制，附加调试器会崩溃
- 不同子版本内存布局可能不同
- passphrase 可能在堆上，扫描不一定能找到

### 法律风险

- 仅可用于自己的微信数据
- 不得用于获取他人数据
- 违反微信用户协议可能导致封号

### 维护成本

- 微信每次大版本更新可能需要重新定位 signature
- 暴力扫描模式相对稳定，但速度较慢

---

## 里程碑

| 阶段 | 目标 | 预计时间 |
|------|------|---------|
| 1 | 项目框架搭建 | 0.5 天 |
| 2 | 进程内存读取 | 1 天 |
| 3 | 密钥派生与验证 | 1 天 |
| 4 | 内存扫描 | 2 天 |
| 5 | 数据库解密导出 | 1.5 天 |
| 6 | 命令行界面 | 1 天 |
| 7 | 配置持久化 | 0.5 天 |
| 8 | 测试验证 | 2 天 |
| **合计** | | **9.5 天** |

---

## 最小可行版本（MVP）

如果要尽快跑通，优先实现：

1. 进程枚举 + 模块定位
2. 整个进程内存暴力扫描 32 字节高熵候选
3. 对每个候选做 PBKDF2 + HMAC 验证
4. 找到后解密 `MicroMsg.db`
5. CLI 只保留 `scan` 和 `decrypt` 两个命令

MVP 预计 3-4 天可完成。

---

## 下一步

如果要开始，先做第一阶段：创建项目框架，然后第二阶段实现 Windows 进程内存读取。

需要我直接帮你开始写第一阶段的代码吗？
