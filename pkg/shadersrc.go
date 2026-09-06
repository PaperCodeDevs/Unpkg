package pkg

import "strings"

type EngineShaderCode struct {
	Name  string
	Type  uint8
	File  string
	Entry string
}

var engineShaderCodes = []EngineShaderCode{
	{"BlitCubemapFacePS", 2, "Utils/Blit.hlsl", "CubemapFaceMainPS"},
	{"BlitCubemapFaceVS", 1, "Utils/Blit.hlsl", "CubemapFaceMainVS"},
	{"BlitPS", 2, "Utils/Blit.hlsl", "MainPS"},
	{"BlitVS", 1, "Utils/Blit.hlsl", "MainVS"},
	{"BloomDownsamplePS", 2, "Postprocess/PostprocessBloom.hlsl", "DownsampleMainPS"},
	{"BloomDownsampleVS", 1, "Postprocess/PostprocessBloom.hlsl", "DownsampleMainVS"},
	{"BloomMergePS", 2, "Postprocess/PostprocessBloom.hlsl", "MergeMainPS"},
	{"BloomMergeVS", 1, "Postprocess/PostprocessBloom.hlsl", "MergeMainVS"},
	{"BloomUpscalePS", 2, "Postprocess/PostprocessBloom.hlsl", "UpscaleMainPS"},
	{"BloomUpscaleVS", 1, "Postprocess/PostprocessBloom.hlsl", "UpscaleMainVS"},
	{"CopyDepthPS", 2, "Utils/Blit.hlsl", "CopyDepthMainPS"},
	{"CustomLineRenderMainPS", 2, "Gizmos/Gizmo.hlsl", "MainPS"},
	{"CustomLineRenderMainVS", 1, "Gizmos/Gizmo.hlsl", "MainVS"},
	{"ForwardShadingMainPS", 2, "RenderPasses/ForwardShadingMainPS.hlsl", "MainPS"},
	{"ForwardShadingMainVS", 1, "RenderPasses/ForwardShadingMainVS.hlsl", "MainVS"},
	{"FullScreenVS", 1, "Utils/Screen.hlsli", "FullScreenMainVS"},
	{"FXAAPS", 2, "Postprocess/PostprocessFXAA.hlsl", "MainPS"},
	{"GIDownsampleDepthPS", 2, "GI/GIRenderPass.hlsl", "GIDownsampleDepthPS"},
	{"GIDownsampleGatherPS", 2, "GI/GIRenderPass.hlsl", "GIDownsampleGatherPS"},
	{"GIGatherPS", 2, "GI/GIRenderPass.hlsl", "GIGatherPS"},
	{"GIUpscalePS", 2, "GI/GIRenderPass.hlsl", "GIUpscalePS"},
	{"GizemoRenderIcon2DMainPS", 2, "Gizmos/GizmoIcon2D.hlsl", "MainPS"},
	{"GizemoRenderIcon2DMainVS", 1, "Gizmos/GizmoIcon2D.hlsl", "MainVS"},
	{"GizmoRenderColorMainPS", 2, "Gizmos/Gizmo.hlsl", "MainPS"},
	{"GizmoRenderColorMainVS", 1, "Gizmos/Gizmo.hlsl", "MainVS"},
	{"GizmoRenderColorOcclusionMainPS", 2, "Gizmos/Gizmo.hlsl", "MainPS"},
	{"GizmoRenderColorOcclusionMainVS", 1, "Gizmos/Gizmo.hlsl", "MainVS"},
	{"GizmoRenderLitMainPS", 2, "Gizmos/Gizmo.hlsl", "MainPS"},
	{"GizmoRenderLitMainVS", 1, "Gizmos/Gizmo.hlsl", "MainVS"},
	{"GizmoRenderTextMainPS", 2, "Gizmos/Gizmo.hlsl", "MainPS"},
	{"GizmoRenderTextMainVS", 1, "Gizmos/Gizmo.hlsl", "MainVS"},
	{"GizmoRenderTextureMainPS", 2, "Gizmos/GizmoTexture.hlsl", "MainPS"},
	{"GizmoRenderTextureMainVS", 1, "Gizmos/GizmoTexture.hlsl", "MainVS"},
	{"GizmoRenderWireMainPS", 2, "Gizmos/Gizmo.hlsl", "MainPS"},
	{"GizmoRenderWireMainVS", 1, "Gizmos/Gizmo.hlsl", "MainVS"},
	{"GodrayRadialBlurPS", 2, "Postprocess/PostprocessGodray.hlsl", "RadialBlurMainPS"},
	{"GTAOHorizonSearchIntegralPS", 2, "RenderPasses/GTAO.hlsl", "HorizonSearchIntegralPS"},
	{"GTAOHorizonSearchIntegralVS", 1, "RenderPasses/GTAO.hlsl", "HorizonSearchIntegralVS"},
	{"GTAOSpatialFilterlVS", 1, "RenderPasses/GTAO.hlsl", "GTAOSpatialFilterlVS"},
	{"GTAOSpatialFilterPS", 2, "RenderPasses/GTAO.hlsl", "GTAOSpatialFilterlPS"},
	{"GUIMainPS", 2, "Gizmos/SimpleGUI.hlsl", "MainPS"},
	{"GUIMainVS", 1, "Gizmos/SimpleGUI.hlsl", "MainVS"},
	{"GUITexMainPS", 2, "Gizmos/SimpleGUI.hlsl", "TexMainPS"},
	{"GUITexMainVS", 1, "Gizmos/SimpleGUI.hlsl", "TexMainVS"},
	{"HZBBuildPS", 2, "RenderPasses/HZB.hlsl", "BuildMainPS"},
	{"HZBTestPS", 2, "RenderPasses/HZB.hlsl", "TestMainPS"},
	{"PassThroughPS", 2, "Utils/PassThroughPS.hlsl", "MainPS"},
	{"PositionOnlyPS", 2, "Utils/PositionOnlyPS.hlsl", "MainPS"},
	{"PositionOnlyVS", 1, "Utils/PositionOnlyVS.hlsl", "MainVS"},
	{"PostprocessDofBlurPS", 2, "Postprocess/PostprocessDof.hlsl", "DofBlurMainPS"},
	{"PostprocessDofBlurVS", 1, "Postprocess/PostprocessDof.hlsl", "DofBlurMainVS"},
	{"PostprocessDofDownPS", 2, "Postprocess/PostprocessDof.hlsl", "DofDownMainPS"},
	{"PostprocessDofDownVS", 1, "Postprocess/PostprocessDof.hlsl", "DofDownMainVS"},
	{"PostprocessDofNearPS", 2, "Postprocess/PostprocessDof.hlsl", "DofNearMainPS"},
	{"PostprocessDofNearVS", 1, "Postprocess/PostprocessDof.hlsl", "DofNearMainVS"},
	{"PostprocessDownsampleSetupPS", 2, "Postprocess/PostprocessSetup.hlsl", "DownsampleMainPS"},
	{"PostprocessLUTsPS", 2, "Postprocess/PostprocessLUTs.hlsl", "MainPS"},
	{"PostprocessMaterialPS", 2, "Postprocess/PostprocessMaterial.hlsl", "MainPS"},
	{"PostprocessRadialFlashPS", 2, "Postprocess/PostprocessRadialFlash.hlsl", "MainPS"},
	{"PostprocessResolvePS", 2, "Postprocess/PostprocessResolve.hlsl", "ResolveMainPS"},
	{"PostprocessResolveVS", 1, "Postprocess/PostprocessResolve.hlsl", "ResolveMainVS"},
	{"PostprocessSetupPS", 2, "Postprocess/PostprocessSetup.hlsl", "SetupMainPS"},
	{"PreZPassPS", 2, "RenderPasses/PreZPassPS.hlsl", "MainPS"},
	{"PreZPassVS", 1, "RenderPasses/PreZPassVS.hlsl", "MainVS"},
	{"ScreenBoxSampleVS", 1, "Utils/Screen.hlsli", "ScreenBoxSampleMainVS"},
	{"ScreenPS", 2, "Utils/Screen.hlsli", "ScreenMainPS"},
	{"ScreenVS", 1, "Utils/Screen.hlsli", "ScreenMainVS"},
	{"ShadowDepthPositionOnlyVS", 1, "RenderPasses/ShadowDepthPositionOnlyVS.hlsl", "MainVS"},
	{"ShadowDepthPS", 2, "RenderPasses/ShadowDepthPS.hlsl", "MainPS"},
	{"ShadowDepthVS", 1, "RenderPasses/ShadowDepthVS.hlsl", "MainVS"},
	{"SihouetteOutlineMeshPositionOnlyPS", 2, "RenderPasses/SihouetteOutlineMeshPositionOnlyPS.hlsl", "MainPS"},
	{"SihouetteOutlineMeshPS", 2, "RenderPasses/SihouetteOutlineMeshPS.hlsl", "MainPS"},
	{"SihouetteOutlineMeshVS", 1, "RenderPasses/SihouetteOutlineMeshVS.hlsl", "MainVS"},
	{"SihouetteOutlinePS", 2, "RenderPasses/SihouetteOutlinePS.hlsl", "MainPS"},
	{"SihouetteOutlineVS", 1, "RenderPasses/SihouetteOutlineVS.hlsl", "MainVS"},
	{"SMAABlendPS", 2, "Postprocess/PostprocessSMAA.hlsl", "BlendMainPS"},
	{"SMAABlendVS", 1, "Postprocess/PostprocessSMAA.hlsl", "BlendMainVS"},
	{"SMAAEdgePS", 2, "Postprocess/PostprocessSMAA.hlsl", "EdgeMainPS"},
	{"SMAAEdgeVS", 1, "Postprocess/PostprocessSMAA.hlsl", "EdgeMainVS"},
	{"SMAANeighborPS", 2, "Postprocess/PostprocessSMAA.hlsl", "NeighborMainPS"},
	{"SMAANeighborVS", 1, "Postprocess/PostprocessSMAA.hlsl", "NeighborMainVS"},
	{"TextureChannelPackerPS", 2, "RenderPasses/TextureChannelPackerPS.hlsl", "MainPS"},
	{"UIMainPS", 2, "UI/UIMainPS.hlsl", "MainPS"},
	{"UIMainVS", 1, "UI/UIMainVS.hlsl", "MainVS"},
}

var engineShaderByAsset = func() map[string]EngineShaderCode {
	m := make(map[string]EngineShaderCode, len(engineShaderCodes))
	for _, c := range engineShaderCodes {
		m[strings.ToLower(c.Name)] = c
	}
	return m
}()

func EngineShaderCodeOf(asset string) (EngineShaderCode, bool) {
	c, ok := engineShaderByAsset[strings.ToLower(asset[strings.LastIndex(asset, "/")+1:])]
	return c, ok
}

func EngineShaderCodes() []EngineShaderCode {
	return append([]EngineShaderCode(nil), engineShaderCodes...)
}

func (c EngineShaderCode) NameHash() uint32 {
	return RainbowNameHash(c.Name)
}
