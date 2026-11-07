# Unpkg

Go 库：迷你世界资源解包与底层加解密算法本体。

## 模块路径

```text
github.com/PaperCodeDevs/Unpkg
github.com/PaperCodeDevs/Unpkg/pkg
github.com/PaperCodeDevs/Unpkg/crn
github.com/PaperCodeDevs/Unpkg/zdeflate
```

## 包结构

```text
.
luajit.go          LuaJIT 2.1 dump 解析对外 API（标准 1B 4C 4A 02 / 迷你世界 1B 4C 4A 90）
parse/             dump 头、proto、kgc/knum、扫描
op/                标准 opcode 名 + 迷你世界编号对照
lua/               反汇编 / 反编译（if-else、and/or、while/repeat、泛型 for、TDUP）
pkg/               .pkg 容器 Parse/Lookup、LZ4、XXTEA、engine/material 索引、贴图/FGUI/mesh/Spine
zdeflate/          zlib 1.2.7 raw deflate level 9（纯 Go）
crn/               Unity Crunch（format=29）
astc/              ASTC LDR 解码（format 48..53）
spine/             Spine 二进制骨骼 3.6–4.1 + 文字图集
cmd/ljdump         调试入口；扫 blob 时覆盖率不满则 exit 1
```

## 作为库用

```go
import "github.com/PaperCodeDevs/Unpkg/pkg"

r, err := pkg.ParseFile(path)
if err != nil { ... }
defer r.Close()

data, err := r.Lookup("../script/csvdef/utf8/blockdef.csv")
```

加解密：

```go
enc := pkg.EncryptXXTEA(plain, key, true)
dec, err := pkg.DecryptXXTEA(enc, key, true)
```

## zdeflate

纯 Go 移植 zlib 1.2.7 `deflate_slow`（无 zlib 头尾、无 cgo、不调 DLL）。`Compress` 默认 level 9、windowBits 15、memLevel 8、strategy 0。`pkg.RawDeflate` 转调它。

## ljdump

```bash
ljdump <file|dir> [out-dir]
```

## 与主仓的边界

Unpkg 提供 `.pkg` 容器解析、LZ4 与 XXTEA 算法本体；Rainbow 头之后的引用表 / 对象记录 / MaterialInstance 由 SpawnPro `world/rainbow` 逐字段解析。密钥由调用方经 `PlainOpt.ServerList` 等参数传入（主仓 `config.PkgServerListKey`）。

逆向细节与对拍数据见主仓 `docs/crypto/`。

## 规范

详见主仓 `CLAUDE.md`。
