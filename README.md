# Unpkg

Go 函式庫：離線解析 LuaJIT 2.1 dump（標準 `1B 4C 4A 02`、迷你世界 `1B 4C 4A 90`），並反編譯成 Lua。除 LuaJIT 外提供迷你世界 `.pkg` 解包（`github.com/PaperCodeDevs/Unpkg/pkg`）。

```text
github.com/PaperCodeDevs/Unpkg
github.com/PaperCodeDevs/Unpkg/pkg
github.com/PaperCodeDevs/Unpkg/crn
github.com/PaperCodeDevs/Unpkg/zdeflate
```

```text
.
luajit.go          對外 API
parse/             dump 頭、proto、kgc/knum、掃描
op/                標準 opcode 名 + 迷你世界編號對照
lua/               反彙編 / 反編譯（if-else、and/or、while/repeat、泛型 for、TDUP）/ 覆蓋檢查
pkg/               .pkg 容器 Parse/Lookup、LZ4、engine/material 索引、貼圖/FGUI/mesh、指令碼 dump
zdeflate/          zlib 1.2.7 raw deflate level 9（純 Go，`Compress` / `pkg.RawDeflate`）
crn/               Unity Crunch（format=29）
cmd/ljdump         程序入口；掃 blob 時覆蓋率不滿則 exit 1
```

```text
ljdump <file|dir> [out-dir]
```

## zdeflate

純 Go 移植 zlib 1.2.7 `deflate_slow`（無 zlib 頭尾、無 cgo、不調 DLL）。`Compress` 預設 level 9、windowBits 15、memLevel 8、strategy 0（good=32、max_lazy=258、nice=258、max_chain=4096）。`pkg.RawDeflate` 轉調它。

本機 `%APPDATA%\miniworddata110\data` 與主倉 `.temp/rev-reslist2/samples`：遊戲族成員（local ver=20、flags=0x0003、method=8、DOS time=0 date=32、extra=0；EncHead 非 11 位元組零頭。LCG EncHead 僅當壓縮體對不上 zdeflate 時剔除）deflate **9952/9952** 與遊戲壓縮體逐位元組相同（json 4976、bin 3773、script 707、trigger 496）。C zlib 1.2.7 oracle（空 / 1B / 64KiB 零 / 1MiB / 重複文字）逐位元組一致。1MiB 壓縮 4.2ms。

## pkg 實測（2026-08-31）

本機 `miniworddata110` + 啟動器，16 個內容不同的 `.pkg`，`ParseFile`+`OpenReader` 全過。全路徑 Lookup：

| 包 | names | 有資料 | empty | fail |
|---|---:|---:|---:|---:|
| common_res | 99616 | 43714 | 55902 | 0 |
| patch_common_res | 11407 | 7690 | 3717 | 0 |
| dx_res | 69396 | 69396 | 0 | 0 |
| 其餘 indexed/engine/material | 見 `.temp/pkg_census/lookup.json` | 全有資料 | 0 | 0 |

empty = 索引 `size=0`（雲端 remotes 佔位），不是解碼失敗。`CensusCommonRes`：png 解碼 **19773/19773**，方塊 `resources/minigame/blocks/*.png` **4473/4473**。`ultrastone` 容器 39472 → PNG 62214。

`OpenOverlay` 必須合併 base/patch 的 `bases`，否則 `ResolveTex` 在 overlay 上恆失敗。方塊表金標是 `Lookup("../script/csvdef/utf8/blockdef.csv")`：**3450** 行 / 86 列（雙表頭），不是 ConcatPlain 掃出來的 85 列 2988 行。抽檢 id 1/100/104 Texture1 在 common_res 能解。

`TintKey` / `CubeFacesRuntime`：與 SandboxGame `0x19E2D00` `"%s_%x"` 相同。平原 `grass_side_46a922` DXT SHA `f6e575b319d755e376c9c3cec51e908e54d9487805dac4aaf206893c23b21964`。`glass` 64×64 DXT SHA `b7979ed52885d0f15711d0df9a1eaf2b1a38aba6d8c37f79864486e509d7f38a`。fmt29 `DecodeTextureDXT` **4956/4956**（2026-09-01 `offline`）。

內嵌 ZIP 必須在明文塊空間取（2026-09-05）：Data 池是 `stor` 塊拼接、相鄰塊可能 LZ4 壓縮，在原始 Data 上線性掃 `PK\x03\x04` 再按偏移切載荷，頭能掃到、載荷一跨進壓縮塊就是壓縮位元組，表現為 `inflate corrupt / unexpected EOF`。`Reader.ScanZipPlain` / `ExtractZipPlain` 按塊取明文、4K 尾巴拼跨塊頭、`read` 跨塊拼載荷；`DumpScripts` / `DumpBlockTexturesReport` / `ExtractBlockTexturePkg` / `StatZipFallback` 已切過去。實測 common_res 原始掃 644 條 → 明文掃 **5566** 條（Lua 920/920、方塊 `png_` 628/628、其它 4018/4018，CRC32 全部一致）；patch_common_res 813 → **5128**（1023/1023、4105/4105）。曾記為損壞的 9 條 `btree.lua`、`water_still` / `moss1_1` 全是這一現象。`bp*` 藍圖包 22 條成員是資料描述符形態（flag bit3、頭內 comp/crc=0），按塊喂 flate 到流末尾再在消費位置 -8..+24 內找 CRC+長度命中的描述符，明文與 patch 裡同名普通形態條目逐位元組一致。`method=0` 原樣儲存條目直接返回載荷。

`ParseCsvDef` 非 UTF-8 時按 GB18030 轉碼（2026-09-05）：`csvdef/utf8/` 下 `antiaddiction`（63 行）/ `bperrorcode`（5）/ `liteconfig`（36）/ `itemuseskindef`（41，patch）4 張表實為 GBK，此前被當空表；其餘 184+55 張 UTF-8 表不受影響。

engine_res 的 20 張 `.png` 是 Rainbow 貼圖物件（2026-09-05，`texrainbow.go`）：頭 `u32 0 / u32 1 / u32 0xD90C0772 / u16 1 / u32 寬 / u32 高 / u32 載荷長 / u32 格式=29 / u32 mip 級數 / 37B 常量 / u32 源寬 / u32 源高 / 12B 常量 / u32 載荷長（重複）`，偏移 95 起是 Crunch（`48 78`，DXT5 單面），尾部補零到 4 對齊。`IsRainbowTex` / `ParseRainbowTex` / `DecodeRainbowTex` 複用 `crn.Decode` 並上下翻轉，20/20 解出且尺寸與頭一致（32² 圖示、256² 皮膚、4096² 天空盒）；源寬高是匯入前的非 2 冪尺寸（icon_chat 37×37→32×32）。

`.blockmesh` 按流描述塊解析（2026-09-05，`meshbin.go` / `meshstream.go`）：36B 頭（`0x00` f32 版本、`0x04` 6×f32 整體 AABB 釐米、`0x1C` u32 流數、`0x20` 恆 0）+ 每流 48B 描述（`+0` nidx、`+12` 起始頂點、`+16` nvert、`+20` 6×f32 本流 AABB、`+44` 累計 IB 位元組）+ 各流 u16 IB 直接拼接（4 對齊）+ 64B 頂點宣告（14 槽，pos@0 type3 / nrm@12 / tan@24 type4 / uv@40 type2，stride 48）+ u32 VB 位元組 + 共享 VB。索引是共享 VB 的絕對下標，各流頂點區間可重疊。單流即 `streams=1` 特例（舊的 `0x24/0x34/0x50` 就是 desc[0]）。全 pkg 1866 份 `1:1863 2:3` 全過，1863 份單流結果與舊演算法逐份雜湊一致。

貼圖容器畫素起點是 **107** 不是 108（2026-09-05）：`[103:107)` 是 dataSize 副本，`Hx` 恆在 107，檔案尾多 1 位元組填充；舊程式碼 raw 格式整體錯位一位元組（core_res 360 張 format 4 的 alpha∈{0,255} 佔比 @107=0.851 / @108=0.440，format 25 從 108 是噪點）。`modules/Unpkg/astc` 純 Go ASTC LDR 解碼接入 format 48..53（量化表與 astcenc 逐項對拍全等），core_res 4575/4575；HDR / sRGB / 3D 塊不支援。

HDR 貼圖容器（2026-09-05，`texhdr.go`）：format 15/16/17/18/19/20/22（RHalf … RGBAFloat、RGB9e5）按 +44 面數拼橫條（立方圖 6 面，每面 = DataSize 含 mip 鏈，取 mip0），線性轉 sRGB 8bit；`DecodeTextureImage` 依次分派 Rainbow 物件 → HDR → CRN → raw。

UI 族（2026-09-05）：FairyGUI `.fui` 五包 **758/758**（v5 314 / v6 354 / v7 90；items 56777、元件 23665、切片 30878），`ParseFGUIPack` 匯出、`FGUIPackInfo.Version/Sprites`；引用圖集 925 張全解，127 個引用在本機五包不存在（活動 UI 熱更）。pkg 內貼圖被重取樣到最近 2 的冪（1560×720 → 2048×512）而 atlas item 保留原尺寸，`Crop` 按原始→解碼尺寸縮放矩形並重取樣回邏輯尺寸。新子包 `spine/`：Spine 二進位制骨骼 3.6–4.1（`Probe` / `ParseSkeleton`，3.8 字串表、4.0 curve 位置與顏色時間軸差異均實現）與文字圖集（`ParseAtlas` / `PageRect` / `ScaledRect`），`.skel` **870/870**（3.8.99 / 3.8.75 / 4.0.64）、`.atlas` **868/868**，附件名↔region 42667 命中 0 缺失；時間軸只保留型別/目標/幀時間，不重建曲線值與 deform 頂點。

容器與文字類（2026-09-05，`plainkind.go` / `plainbin.go`）：zip 12、mod 3、trigger 16、db 96、`.r` 296（含 292 份 homegarden 單 chunk：`0c1c0706` + 11 位元組 XOR 頭 + LZMA）、fb 2、json 1385/1386（唯一失敗是資源本身的全形逗號）、xml 215、csv 192、bil 17（明文 Lua）、pb 2、asset 2、gif 65 全過；`serverlist.data` 19 份（2026-09-06，`serverlist.go`）：libSandBoxEngine 自帶 DES-EDE2 ECB，查表與標準 DES 相同，差別是位元組低位在前展開成位元、S 盒輸出 nibble 低位先寫、16 輪後不交換左右半；金鑰 ≤16 位元組零填充切成 k1/k2，明文零填充到 8 的倍數；`DecryptServerList` / `EncryptServerList` 19/19 解出 `<Config>` XML 並逐位元組重加密還原，18 份與同名 `.xml` 只差 CRLF。金鑰由呼叫方經 `PlainOpt.ServerList` 傳入（主倉 `config.PkgServerListKey`），演算法細節見主倉 `docs/crypto/pkg.md` `.data` 行。

著色器快取三包（2026-09-05，`lz4wrap.go` / `dxwrap.go` / `binding.go` / `bindmat.go`）：material_res 著色器 blob 是 **`u32 rawSize | LZ4 block`**（15388/15388 可解，28B 記錄 extra=isCompress），解壓後 `sha1[20] | u32(6+DXBC 總長) | 01 SRV數 CB數 取樣器數 00 00 | DXBC | 繫結表`；舊記錄的 Type / Slot / Stage 是 LZ4 令牌與字面量長度被誤讀，不存在管線型別下標。dx_res / engine_res / first_res 的 `d3d11/<hash>` 是同一 10B 包裝（`u32 = DXBC 總長+6 | 01 | SRV | CB | Sampler | 00 00`），四包 93276 條與繫結表計數 100% 一致；繫結表語法（屬性對、紋理 kind 2/3/4/5、常量緩衝變數）77953 條精確吃盡，material 舊版變體 15322 條另一語法。無副檔名條目 unknown=0：根名條目是 20B 或 `u32 20+sha1` 的 24B 指紋。`.compute` 6 份含多份 DXBC，`ExtractAllDXBC` 切全部。engine_res 舊「45 個物件」是壓縮流偽結構，真實條目是 `u32 rawSize | LZ4`，解壓後 Rainbow 頭 `u32 0 | u32 n | n×{u32 型別 id, u16 版本} | guid`，`Reader.EngineObjects()` 出 106 個資源；`d3d11/shaderidlist.bin` Lookup 修正為真正的 idlist（engine 40B/條，dx/first 48B/條）。兩份 first_res 的 3 位元組差是 14 條著色器 cbuffer Size 不同經 LZ4 後的長度差。雜湊原像（2026-09-06，`xxh32.go` / `cbunit.go` / `pipelinemap.go` / `shadersrc.go`）：material 尾表 6×21 單元的 `u32 hash` = `RainbowNameHash`（XXH32 seed 0x8F37154B，libengine 0x1E2BD0）of 常量緩衝名，`DXBinding.CBUnits` 10877/10877 解出 ViewCB(320B)/PrimitiveCB(144B) 且與繫結表同名 cbuffer 的 size/slot 一致；`ParseShaderPipelineMap` 解 `effectshaderpipelinemap.bin` 三種佈局（dx/first 數值版 646 管線 / 42 型別名 / 7 頂點佈局 / 350 關鍵字集 / 950 shaderId，engine 命名版 124 管線（階段型別取 `EngineShaderCodeOf` 的 VS=1 / PS=2，非內建名按字尾推，推不出為 0），material 命名版 24B 關鍵字集 10 管線），`ParseShaderIDList` 解 40B/48B idlist，dx 950 與 engine 172 個 shaderId（124 管線 × 2 階段 = 248 處引用，29 個型別名）全部經 idlist 落到存在的 `d3d11/` 檔案；`EngineShaderCodeOf` 是 libengine 靜態註冊的 84 個內建著色器（名字 / 階段 / HLSL 檔案 / 入口），dx_res 24B 根名指紋按 HLSL 檔案分組恰一值、跨檔案互異，即原始檔 SHA-1。

未攻破（禁止盲撞）：attrMask 語義、kind 5 名字、idlist src 的兩項編輯機側輸入（`g_ShaderMgr+0x100` 與著色器描述符 `+0x28`，拼接式已釘）、engine_res 20B 繫結雜湊所指"另一份 fileId"的語義、Rainbow 頭之外的數值欄位；`libMiniBlock` `CompileSection` 逐面消隱未還原（`MiniCraftRendererBuildSectionMeshEvent` 在 SandboxGame 裡是 SprayPaintMgr/BulletMgr 監聽，不是 mesher）。

engine_res 條目 Rainbow 頭之後的引用表 / 物件記錄 / 舊版 MaterialInstance 與 FMaterial v1 體由 SpawnPro `world/rainbow`（`ParseEngineAsset` / `ParseEngineTemplate`）逐欄位解析（2026-09-06，51/51 templatemat + 5/5 mat 嚴格到 EOF），Unpkg 仍只負責 `u32 rawSize | LZ4` 與 `EngineObjects()` 頭；結論見主倉 `docs/crypto/pkg.md`「未決」第 1 條與 `docs/crypto/rainbow.md`「mat」engine_res 小節。
