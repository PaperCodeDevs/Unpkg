# Unpkg

Go 库：离线解析 LuaJIT 2.1 dump（标准 `1B 4C 4A 02`、迷你世界 `1B 4C 4A 90`），并反编译成 Lua。除 LuaJIT 外提供迷你世界 `.pkg` 解包（`github.com/PaperCodeDevs/Unpkg/pkg`）。

```text
github.com/PaperCodeDevs/Unpkg
github.com/PaperCodeDevs/Unpkg/pkg
github.com/PaperCodeDevs/Unpkg/crn
```

```text
.
luajit.go          对外 API
parse/             dump 头、proto、kgc/knum、扫描
op/                标准 opcode 名 + 迷你世界编号对照
lua/               反汇编 / 反编译（if-else、and/or、while/repeat、泛型 for、TDUP）/ 覆盖检查
pkg/               .pkg 容器 Parse/Lookup、LZ4、engine/material 索引、贴图/FGUI/mesh、脚本 dump
crn/               Unity Crunch（format=29）
cmd/ljdump         进程入口；扫 blob 时覆盖率不满则 exit 1
```

```text
ljdump <file|dir> [out-dir]
```

## pkg 实测（2026-08-31）

本机 `miniworddata110` + 启动器，16 个内容不同的 `.pkg`，`ParseFile`+`OpenReader` 全过。全路径 Lookup：

| 包 | names | 有数据 | empty | fail |
|---|---:|---:|---:|---:|
| common_res | 99616 | 43714 | 55902 | 0 |
| patch_common_res | 11407 | 7690 | 3717 | 0 |
| dx_res | 69396 | 69396 | 0 | 0 |
| 其余 indexed/engine/material | 见 `.temp/pkg_census/lookup.json` | 全有数据 | 0 | 0 |

empty = 索引 `size=0`（云端 remotes 占位），不是解码失败。`CensusCommonRes`：png 解码 **19773/19773**，方块 `resources/minigame/blocks/*.png` **4473/4473**。`ultrastone` 容器 39472 → PNG 62214。

`OpenOverlay` 必须合并 base/patch 的 `bases`，否则 `ResolveTex` 在 overlay 上恒失败。方块表金标是 `Lookup("../script/csvdef/utf8/blockdef.csv")`：**3450** 行 / 86 列（双表头），不是 ConcatPlain 扫出来的 85 列 2988 行。抽检 id 1/100/104 Texture1 在 common_res 能解。

`TintKey` / `CubeFacesRuntime`：与 SandboxGame `0x19E2D00` `"%s_%x"` 相同。平原 `grass_side_46a922` DXT SHA `f6e575b319d755e376c9c3cec51e908e54d9487805dac4aaf206893c23b21964`。`glass` 64×64 DXT SHA `b7979ed52885d0f15711d0df9a1eaf2b1a38aba6d8c37f79864486e509d7f38a`。fmt29 `DecodeTextureDXT` **4956/4956**（2026-09-01 `offline`）。

未攻破（禁止盲撞）：common_res 内嵌 ZipCrypto 若干 `btree.lua` inflate 损坏或 `a0817i` 解出非 Lua；`libMiniBlock` `CompileSection` 逐面消隐未还原（`MiniCraftRendererBuildSectionMeshEvent` 在 SandboxGame 里是 SprayPaintMgr/BulletMgr 监听，不是 mesher）；3 个 `0x1C=2` 的 `.blockmesh`。
