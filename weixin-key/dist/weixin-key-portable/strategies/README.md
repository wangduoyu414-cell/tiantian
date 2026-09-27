# strategies/ — 版本策略库

每个 JSON 是一份**已验证过的微信版本知识**。工具链运行时按 `Weixin.dll` 的
SHA256 精确匹配（不用版本号，因为同版本号可能有不同构建）。

## Schema

```json
{
  "weixin_version": "4.1.15.13",        // 展示用版本号
  "module_sha256": "<Weixin.dll SHA256, 大写 hex>",  // 匹配键
  "verified_on": "2026-09-27",           // 真实验证日期
  "material": {
    "layout": "ptr-zero-len32-cap47",    // 内存材料对象布局
    "static_probe": "two-59byte-shapes", // material-probe --inspect 预期
    "db_kdf": "pbkdf2-sha512-256000"     // 库密钥 KDF
  },
  "message_db": {
    "page_size": 4096,
    "reserve": 80,
    "table_naming": "Msg_ + md5(username)",
    "sender_resolution": "per-db Name2Id rowid",
    "compressed_content": "zstd-nodict (WCDB_CT=4)",
    "schema": "17-column v4.1"
  },
  "image_dat": {
    "container": "v2 (07 08 56 32)",
    "head_encryption": "aes-128-ecb, len field @6:10",
    "tail_encryption": "single-byte xor, len field @10:14, derive from jpg tail",
    "key_source": "process-memory lazy-load (user must open images once)"
  },
  "notes": "自由文本:该版本的坑"
}
```

## 命中与降级

1. `module_sha256` 命中 → 走该策略声明的验证过的路径。
2. 未命中 → `material-probe --inspect` 静态分析;指令形状兼容则可谨慎尝试
   有界被动采集(它本身只读、自带全部安全门),不兼容则停止并生成新策略
   草稿(填好模块哈希和静态分析结果,其余留空待验证)。
3. 策略只是经验记录,**不是安全门**;所有运行时校验(HMAC、快照、逐库
   认证)照常进行,策略未命中不改变任何安全行为。

## 禁止事项

- 策略文件**永不包含密钥、passphrase 或任何账号材料**。
- 不把"静态形状匹配"写成"已验证"——`verified_on` 只在真实取钥+导出
  跑通后填写。
