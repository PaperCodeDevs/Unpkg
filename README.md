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

内嵌 ZIP 必须在明文块空间取（2026-09-05）：Data 池是 `stor` 块拼接、相邻块可能 LZ4 压缩，在原始 Data 上线性扫 `PK\x03\x04` 再按偏移切载荷，头能扫到、载荷一跨进压缩块就是压缩字节，表现为 `inflate corrupt / unexpected EOF`。`Reader.ScanZipPlain` / `ExtractZipPlain` 按块取明文、4K 尾巴拼跨块头、`read` 跨块拼载荷；`DumpScripts` / `DumpBlockTexturesReport` / `ExtractBlockTexturePkg` / `StatZipFallback` 已切过去。实测 common_res 原始扫 644 条 → 明文扫 **5566** 条（Lua 920/920、方块 `png_` 628/628、其它 4018/4018，CRC32 全部一致）；patch_common_res 813 → **5128**（1023/1023、4105/4105）。曾记为损坏的 9 条 `btree.lua`、`water_still` / `moss1_1` 全是这一现象。`bp*` 蓝图包 22 条成员是数据描述符形态（flag bit3、头内 comp/crc=0），按块喂 flate 到流末尾再在消费位置 -8..+24 内找 CRC+长度命中的描述符，明文与 patch 里同名普通形态条目逐字节一致。`method=0` 原样存储条目直接返回载荷。

`ParseCsvDef` 非 UTF-8 时按 GB18030 转码（2026-09-05）：`csvdef/utf8/` 下 `antiaddiction`（63 行）/ `bperrorcode`（5）/ `liteconfig`（36）/ `itemuseskindef`（41，patch）4 张表实为 GBK，此前被当空表；其余 184+55 张 UTF-8 表不受影响。

engine_res 的 20 张 `.png` 是 Rainbow 贴图对象（2026-09-05，`texrainbow.go`）：头 `u32 0 / u32 1 / u32 0xD90C0772 / u16 1 / u32 宽 / u32 高 / u32 载荷长 / u32 格式=29 / u32 mip 级数 / 37B 常量 / u32 源宽 / u32 源高 / 12B 常量 / u32 载荷长（重复）`，偏移 95 起是 Crunch（`48 78`，DXT5 单面），尾部补零到 4 对齐。`IsRainbowTex` / `ParseRainbowTex` / `DecodeRainbowTex` 复用 `crn.Decode` 并上下翻转，20/20 解出且尺寸与头一致（32² 图标、256² 皮肤、4096² 天空盒）；源宽高是导入前的非 2 幂尺寸（icon_chat 37×37→32×32）。

`.blockmesh` 按流描述块解析（2026-09-05，`meshbin.go` / `meshstream.go`）：36B 头（`0x00` f32 版本、`0x04` 6×f32 整体 AABB 厘米、`0x1C` u32 流数、`0x20` 恒 0）+ 每流 48B 描述（`+0` nidx、`+12` 起始顶点、`+16` nvert、`+20` 6×f32 本流 AABB、`+44` 累计 IB 字节）+ 各流 u16 IB 直接拼接（4 对齐）+ 64B 顶点声明（14 槽，pos@0 type3 / nrm@12 / tan@24 type4 / uv@40 type2，stride 48）+ u32 VB 字节 + 共享 VB。索引是共享 VB 的绝对下标，各流顶点区间可重叠。单流即 `streams=1` 特例（旧的 `0x24/0x34/0x50` 就是 desc[0]）。全 pkg 1866 份 `1:1863 2:3` 全过，1863 份单流结果与旧算法逐份哈希一致。

贴图容器像素起点是 **107** 不是 108（2026-09-05）：`[103:107)` 是 dataSize 副本，`Hx` 恒在 107，文件尾多 1 字节填充；旧代码 raw 格式整体错位一字节（core_res 360 张 format 4 的 alpha∈{0,255} 占比 @107=0.851 / @108=0.440，format 25 从 108 是噪点）。`modules/Unpkg/astc` 纯 Go ASTC LDR 解码接入 format 48..53（量化表与 astcenc 逐项对拍全等），core_res 4575/4575；HDR / sRGB / 3D 块不支持。

HDR 贴图容器（2026-09-05，`texhdr.go`）：format 15/16/17/18/19/20/22（RHalf … RGBAFloat、RGB9e5）按 +44 面数拼横条（立方图 6 面，每面 = DataSize 含 mip 链，取 mip0），线性转 sRGB 8bit；`DecodeTextureImage` 依次分派 Rainbow 对象 → HDR → CRN → raw。

UI 族（2026-09-05）：FairyGUI `.fui` 五包 **758/758**（v5 314 / v6 354 / v7 90；items 56777、组件 23665、切片 30878），`ParseFGUIPack` 导出、`FGUIPackInfo.Version/Sprites`；引用图集 925 张全解，127 个引用在本机五包不存在（活动 UI 热更）。pkg 内贴图被重采样到最近 2 的幂（1560×720 → 2048×512）而 atlas item 保留原尺寸，`Crop` 按原始→解码尺寸缩放矩形并重采样回逻辑尺寸。新子包 `spine/`：Spine 二进制骨骼 3.6–4.1（`Probe` / `ParseSkeleton`，3.8 字符串表、4.0 curve 位置与颜色时间轴差异均实现）与文本图集（`ParseAtlas` / `PageRect` / `ScaledRect`），`.skel` **870/870**（3.8.99 / 3.8.75 / 4.0.64）、`.atlas` **868/868**，附件名↔region 42667 命中 0 缺失；时间轴只保留类型/目标/帧时间，不重建曲线值与 deform 顶点。

容器与文本类（2026-09-05，`plainkind.go` / `plainbin.go`）：zip 12、mod 3、trigger 16、db 96、`.r` 296（含 292 份 homegarden 单 chunk：`0c1c0706` + 11 字节 XOR 头 + LZMA）、fb 2、json 1385/1386（唯一失败是资源本身的全角逗号）、xml 215、csv 192、bil 17（明文 Lua）、pb 2、asset 2、gif 65 全过；`serverlist.data` 19 份（2026-09-06，`serverlist.go`）：libSandBoxEngine 自带 DES-EDE2 ECB，查表与标准 DES 相同，差别是字节低位在前展开成比特、S 盒输出 nibble 低位先写、16 轮后不交换左右半；密钥 ≤16 字节零填充切成 k1/k2，明文零填充到 8 的倍数；`DecryptServerList` / `EncryptServerList` 19/19 解出 `<Config>` XML 并逐字节重加密还原，18 份与同名 `.xml` 只差 CRLF。密钥由调用方经 `PlainOpt.ServerList` 传入（主仓 `config.PkgServerListKey`），算法细节见主仓 `docs/crypto/pkg.md` `.data` 行。

着色器缓存三包（2026-09-05，`lz4wrap.go` / `dxwrap.go` / `binding.go` / `bindmat.go`）：material_res 着色器 blob 是 **`u32 rawSize | LZ4 block`**（15388/15388 可解，28B 记录 extra=isCompress），解压后 `sha1[20] | u32(6+DXBC 总长) | 01 SRV数 CB数 采样器数 00 00 | DXBC | 绑定表`；旧记录的 Type / Slot / Stage 是 LZ4 令牌与字面量长度被误读，不存在管线类型下标。dx_res / engine_res / first_res 的 `d3d11/<hash>` 是同一 10B 包装（`u32 = DXBC 总长+6 | 01 | SRV | CB | Sampler | 00 00`），四包 93276 条与绑定表计数 100% 一致；绑定表语法（属性对、纹理 kind 2/3/4/5、常量缓冲变量）77953 条精确吃尽，material 旧版变体 15322 条另一语法。无扩展名条目 unknown=0：根名条目是 20B 或 `u32 20+sha1` 的 24B 指纹。`.compute` 6 份含多份 DXBC，`ExtractAllDXBC` 切全部。engine_res 旧「45 个对象」是压缩流伪结构，真实条目是 `u32 rawSize | LZ4`，解压后 Rainbow 头 `u32 0 | u32 n | n×{u32 类型 id, u16 版本} | guid`，`Reader.EngineObjects()` 出 106 个资源；`d3d11/shaderidlist.bin` Lookup 修正为真正的 idlist（engine 40B/条，dx/first 48B/条）。两份 first_res 的 3 字节差是 14 条着色器 cbuffer Size 不同经 LZ4 后的长度差。哈希原像（2026-09-06，`xxh32.go` / `cbunit.go` / `pipelinemap.go` / `shadersrc.go`）：material 尾表 6×21 单元的 `u32 hash` = `RainbowNameHash`（XXH32 seed 0x8F37154B，libengine 0x1E2BD0）of 常量缓冲名，`DXBinding.CBUnits` 10877/10877 解出 ViewCB(320B)/PrimitiveCB(144B) 且与绑定表同名 cbuffer 的 size/slot 一致；`ParseShaderPipelineMap` 解 `effectshaderpipelinemap.bin` 三种布局（dx/first 数值版 646 管线 / 42 类型名 / 7 顶点布局 / 350 关键字集 / 950 shaderId，engine 命名版 124 管线（阶段类型取 `EngineShaderCodeOf` 的 VS=1 / PS=2，非内建名按后缀推，推不出为 0），material 命名版 24B 关键字集 10 管线），`ParseShaderIDList` 解 40B/48B idlist，dx 950 与 engine 172 个 shaderId（124 管线 × 2 阶段 = 248 处引用，29 个类型名）全部经 idlist 落到存在的 `d3d11/` 文件；`EngineShaderCodeOf` 是 libengine 静态注册的 84 个内建着色器（名字 / 阶段 / HLSL 文件 / 入口），dx_res 24B 根名指纹按 HLSL 文件分组恰一值、跨文件互异，即源文件 SHA-1。

未攻破（禁止盲撞）：attrMask 语义、kind 5 名字、idlist src 的两项编辑机侧输入（`g_ShaderMgr+0x100` 与着色器描述符 `+0x28`，拼接式已钉）、engine_res 20B 绑定哈希所指"另一份 fileId"的语义、Rainbow 头之外的数值字段；`libMiniBlock` `CompileSection` 逐面消隐未还原（`MiniCraftRendererBuildSectionMeshEvent` 在 SandboxGame 里是 SprayPaintMgr/BulletMgr 监听，不是 mesher）。

engine_res 条目 Rainbow 头之后的引用表 / 对象记录 / 旧版 MaterialInstance 与 FMaterial v1 体由 SpawnPro `world/rainbow`（`ParseEngineAsset` / `ParseEngineTemplate`）逐字段解析（2026-09-06，51/51 templatemat + 5/5 mat 严格到 EOF），Unpkg 仍只负责 `u32 rawSize | LZ4` 与 `EngineObjects()` 头；结论见主仓 `docs/crypto/pkg.md`「未决」第 1 条与 `docs/crypto/rainbow.md`「mat」engine_res 小节。
