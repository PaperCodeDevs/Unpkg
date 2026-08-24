# Unpkg

Go 库：离线解析 LuaJIT 2.1 dump（标准 `1B 4C 4A 02`、迷你世界 `1B 4C 4A 90`），并反编译成 Lua。除 LuaJIT 外提供迷你世界 `.pkg` 解包（`github.com/PaperCodeDevs/Unpkg/pkg`）。

```text
github.com/PaperCodeDevs/Unpkg
github.com/PaperCodeDevs/Unpkg/pkg
github.com/PaperCodeDevs/Unpkg/crn
github.com/PaperCodeDevs/Unpkg/rainbow
```

```text
.
luajit.go          对外 API
parse/             dump 头、proto、kgc/knum、扫描
op/                标准 opcode 名 + 迷你世界编号对照
lua/               反汇编 / 反编译（if-else、and/or、while/repeat、泛型 for、TDUP）/ 覆盖检查
pkg/               .pkg 容器 Parse/Lookup、LZ4、engine/material 索引、贴图/FGUI/mesh、脚本 dump
crn/               Unity Crunch（format=29）
rainbow/           templatemat（HLSL / 槽 / 关键字）；云 `.mat`（ParseMat）；`.prefab`（ParsePrefab）
cmd/ljdump         进程入口；扫 blob 时覆盖率不满则 exit 1
```

```text
ljdump <file|dir> [out-dir]
```
