//go:build linux
// +build linux

package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"os"
	"reflect"
	"time"
	"unsafe"

	mgl32 "github.com/go-gl/mathgl/mgl32"
	"github.com/vulkan-go/vulkan"
)

// createOverlayPipeline builds the HUD pipeline for the FPS text overlay.
func (a *VulkanApp) createOverlayPipeline() error {
	vertCode, err := os.ReadFile("shaders/overlay_vert.spv")
	if err != nil {
		return fmt.Errorf("read overlay vertex shader: %w", err)
	}
	fragCode, err := os.ReadFile("shaders/overlay_frag.spv")
	if err != nil {
		return fmt.Errorf("read overlay fragment shader: %w", err)
	}

	vertModule, err := a.createShaderModule(vertCode)
	if err != nil {
		return err
	}
	defer vulkan.DestroyShaderModule(a.device, vertModule, nil)
	fragModule, err := a.createShaderModule(fragCode)
	if err != nil {
		return err
	}
	defer vulkan.DestroyShaderModule(a.device, fragModule, nil)

	mainName := "main\x00"
	shaderStages := []vulkan.PipelineShaderStageCreateInfo{
		{
			SType:  vulkan.StructureTypePipelineShaderStageCreateInfo,
			Stage:  vulkan.ShaderStageVertexBit,
			Module: vertModule,
			PName:  mainName,
		},
		{
			SType:  vulkan.StructureTypePipelineShaderStageCreateInfo,
			Stage:  vulkan.ShaderStageFragmentBit,
			Module: fragModule,
			PName:  mainName,
		},
	}

	bindingDescription := vulkan.VertexInputBindingDescription{
		Binding:   0,
		Stride:    uint32(unsafe.Sizeof(overlayVertex{})),
		InputRate: vulkan.VertexInputRateVertex,
	}
	attributeDescriptions := []vulkan.VertexInputAttributeDescription{
		{Location: 0, Binding: 0, Format: vulkan.FormatR32g32Sfloat, Offset: uint32(unsafe.Offsetof(overlayVertex{}.pos))},
		{Location: 1, Binding: 0, Format: vulkan.FormatR32g32b32Sfloat, Offset: uint32(unsafe.Offsetof(overlayVertex{}.color))},
	}

	vertexInput := vulkan.PipelineVertexInputStateCreateInfo{
		SType:                           vulkan.StructureTypePipelineVertexInputStateCreateInfo,
		VertexBindingDescriptionCount:   1,
		PVertexBindingDescriptions:      []vulkan.VertexInputBindingDescription{bindingDescription},
		VertexAttributeDescriptionCount: uint32(len(attributeDescriptions)),
		PVertexAttributeDescriptions:    attributeDescriptions,
	}

	inputAssembly := vulkan.PipelineInputAssemblyStateCreateInfo{
		SType:                  vulkan.StructureTypePipelineInputAssemblyStateCreateInfo,
		Topology:               vulkan.PrimitiveTopologyTriangleList,
		PrimitiveRestartEnable: vulkan.False,
	}

	viewport := vulkan.Viewport{
		X:        0,
		Y:        0,
		Width:    float32(a.swapchainExtent.Width),
		Height:   float32(a.swapchainExtent.Height),
		MinDepth: 0,
		MaxDepth: 1,
	}
	scissor := vulkan.Rect2D{
		Offset: vulkan.Offset2D{X: 0, Y: 0},
		Extent: a.swapchainExtent,
	}
	viewportState := vulkan.PipelineViewportStateCreateInfo{
		SType:         vulkan.StructureTypePipelineViewportStateCreateInfo,
		ViewportCount: 1,
		PViewports:    []vulkan.Viewport{viewport},
		ScissorCount:  1,
		PScissors:     []vulkan.Rect2D{scissor},
	}

	rasterizer := vulkan.PipelineRasterizationStateCreateInfo{
		SType:                   vulkan.StructureTypePipelineRasterizationStateCreateInfo,
		RasterizerDiscardEnable: vulkan.False,
		PolygonMode:             vulkan.PolygonModeFill,
		LineWidth:               1.0,
		CullMode:                vulkan.CullModeFlags(vulkan.CullModeNone),
		FrontFace:               vulkan.FrontFaceCounterClockwise,
	}

	multisampling := vulkan.PipelineMultisampleStateCreateInfo{
		SType:                vulkan.StructureTypePipelineMultisampleStateCreateInfo,
		RasterizationSamples: vulkan.SampleCount1Bit,
	}

	depthStencil := vulkan.PipelineDepthStencilStateCreateInfo{
		SType:                 vulkan.StructureTypePipelineDepthStencilStateCreateInfo,
		DepthTestEnable:       vulkan.False,
		DepthWriteEnable:      vulkan.False,
		DepthCompareOp:        vulkan.CompareOpAlways,
		DepthBoundsTestEnable: vulkan.False,
		StencilTestEnable:     vulkan.False,
	}

	colorBlendAttachment := vulkan.PipelineColorBlendAttachmentState{
		ColorWriteMask: vulkan.ColorComponentFlags(vulkan.ColorComponentRBit | vulkan.ColorComponentGBit | vulkan.ColorComponentBBit | vulkan.ColorComponentABit),
		BlendEnable:    vulkan.False,
	}
	colorBlending := vulkan.PipelineColorBlendStateCreateInfo{
		SType:           vulkan.StructureTypePipelineColorBlendStateCreateInfo,
		AttachmentCount: 1,
		PAttachments:    []vulkan.PipelineColorBlendAttachmentState{colorBlendAttachment},
	}

	layoutInfo := vulkan.PipelineLayoutCreateInfo{
		SType: vulkan.StructureTypePipelineLayoutCreateInfo,
	}
	var zeroLayout vulkan.PipelineLayout
	layoutOut := (*vulkan.PipelineLayout)(C.malloc(C.size_t(unsafe.Sizeof(zeroLayout))))
	if layoutOut == nil {
		return fmt.Errorf("allocate overlay pipeline layout handle")
	}
	defer C.free(unsafe.Pointer(layoutOut))
	if res := vulkan.CreatePipelineLayout(a.device, &layoutInfo, nil, layoutOut); res != vulkan.Success {
		return fmt.Errorf("create overlay pipeline layout: %w", vulkan.Error(res))
	}
	a.overlayPipelineLayout = *layoutOut

	pipelineInfo := vulkan.GraphicsPipelineCreateInfo{
		SType:               vulkan.StructureTypeGraphicsPipelineCreateInfo,
		StageCount:          uint32(len(shaderStages)),
		PStages:             shaderStages,
		PVertexInputState:   &vertexInput,
		PInputAssemblyState: &inputAssembly,
		PViewportState:      &viewportState,
		PRasterizationState: &rasterizer,
		PMultisampleState:   &multisampling,
		PDepthStencilState:  &depthStencil,
		PColorBlendState:    &colorBlending,
		Layout:              a.overlayPipelineLayout,
		RenderPass:          a.renderPass,
		Subpass:             0,
	}

	var zeroPipeline vulkan.Pipeline
	cBuf := C.calloc(C.size_t(1), C.size_t(unsafe.Sizeof(zeroPipeline)))
	if cBuf == nil {
		vulkan.DestroyPipelineLayout(a.device, a.overlayPipelineLayout, nil)
		return fmt.Errorf("allocate overlay pipeline buffer")
	}
	defer C.free(cBuf)
	sh := &reflect.SliceHeader{
		Data: uintptr(cBuf),
		Len:  1,
		Cap:  1,
	}
	pipelines := *(*[]vulkan.Pipeline)(unsafe.Pointer(sh))

	if res := vulkan.CreateGraphicsPipelines(a.device, vulkan.PipelineCache(vulkan.NullHandle), 1, []vulkan.GraphicsPipelineCreateInfo{pipelineInfo}, nil, pipelines); res != vulkan.Success {
		vulkan.DestroyPipelineLayout(a.device, a.overlayPipelineLayout, nil)
		return fmt.Errorf("create overlay pipeline: %w", vulkan.Error(res))
	}
	a.overlayPipeline = pipelines[0]
	return nil
}

// createOverlayBuffers allocates the overlay vertex and indirect draw buffers.
func (a *VulkanApp) createOverlayBuffers() error {
	vertexBufferSize := vulkan.DeviceSize(maxOverlayVertices) * vulkan.DeviceSize(unsafe.Sizeof(overlayVertex{}))
	vb, vbMem, err := a.createBuffer(vertexBufferSize, vulkan.BufferUsageFlags(vulkan.BufferUsageVertexBufferBit), vulkan.MemoryPropertyHostVisibleBit|vulkan.MemoryPropertyHostCoherentBit)
	if err != nil {
		return fmt.Errorf("create overlay vertex buffer: %w", err)
	}
	a.overlayVertexBuffer = vb
	a.overlayVertexBufferMemory = vbMem

	indirectSize := vulkan.DeviceSize(unsafe.Sizeof(vulkan.DrawIndirectCommand{}))
	ib, ibMem, err := a.createBuffer(indirectSize, vulkan.BufferUsageFlags(vulkan.BufferUsageIndirectBufferBit), vulkan.MemoryPropertyHostVisibleBit|vulkan.MemoryPropertyHostCoherentBit)
	if err != nil {
		return fmt.Errorf("create overlay indirect buffer: %w", err)
	}
	a.overlayIndirectBuffer = ib
	a.overlayIndirectMemory = ibMem
	return nil
}

// updateFPSOverlay rebuilds overlay vertices for the current FPS string and updates buffers.
func (a *VulkanApp) updateFPSOverlay() error {
	now := time.Now()
	a.fpsFrameCount++
	elapsed := now.Sub(a.fpsLastTime)
	if elapsed >= time.Second {
		a.fpsValue = float64(a.fpsFrameCount) / elapsed.Seconds()
		a.fpsFrameCount = 0
		a.fpsLastTime = now
	}

	verts := a.buildOverlayVertices(fmt.Sprintf("FPS: %.1f", a.fpsValue))
	if len(verts) > maxOverlayVertices {
		verts = verts[:maxOverlayVertices]
	}
	a.overlayVertexCount = uint32(len(verts))

	if len(verts) > 0 {
		size := vulkan.DeviceSize(len(verts)) * vulkan.DeviceSize(unsafe.Sizeof(overlayVertex{}))
		var data unsafe.Pointer
		if res := vulkan.MapMemory(a.device, a.overlayVertexBufferMemory, 0, size, 0, &data); res != vulkan.Success {
			return fmt.Errorf("map overlay vertex buffer: %w", vulkan.Error(res))
		}
		dst := (*[1 << 30]byte)(data)[:size:size]
		src := (*[1 << 30]byte)(unsafe.Pointer(&verts[0]))[:size:size]
		copy(dst, src)
		vulkan.UnmapMemory(a.device, a.overlayVertexBufferMemory)
	} else {
		a.overlayVertexCount = 0
	}

	draw := vulkan.DrawIndirectCommand{
		VertexCount:   a.overlayVertexCount,
		InstanceCount: 1,
		FirstVertex:   0,
		FirstInstance: 0,
	}
	drawSize := vulkan.DeviceSize(unsafe.Sizeof(draw))
	var idata unsafe.Pointer
	if res := vulkan.MapMemory(a.device, a.overlayIndirectMemory, 0, drawSize, 0, &idata); res != vulkan.Success {
		return fmt.Errorf("map overlay indirect buffer: %w", vulkan.Error(res))
	}
	dstDraw := (*[1 << 30]byte)(idata)[:drawSize:drawSize]
	srcDraw := (*[1 << 30]byte)(unsafe.Pointer(&draw))[:drawSize:drawSize]
	copy(dstDraw, srcDraw)
	vulkan.UnmapMemory(a.device, a.overlayIndirectMemory)
	return nil
}

// buildOverlayVertices converts text into overlay quads using a tiny bitmap font.
func (a *VulkanApp) buildOverlayVertices(text string) []overlayVertex {
	if a.swapchainExtent.Width == 0 || a.swapchainExtent.Height == 0 {
		return nil
	}
	cellW := float32(8)
	cellH := float32(12)
	margin := float32(8)
	space := float32(4)
	color := mgl32.Vec3{1, 1, 1}
	var verts []overlayVertex
	x := margin
	y := margin
	for _, ch := range text {
		pattern := glyphPattern(ch)
		if pattern == nil {
			pattern = glyphPattern(' ')
		}
		if len(pattern) == 0 {
			continue
		}
		for row := 0; row < len(pattern); row++ {
			for col := 0; col < len(pattern[row]); col++ {
				if pattern[row][col] != '1' {
					continue
				}
				px := x + float32(col)*cellW
				py := y + float32(row)*cellH
				verts = append(verts, quadToVertices(px, py, cellW, cellH, color, a.swapchainExtent)...)
			}
		}
		x += float32(len(pattern[0]))*cellW + space
		if len(verts) >= maxOverlayVertices {
			return verts[:maxOverlayVertices]
		}
	}
	return verts
}

// quadToVertices makes two triangles for a quad at pixel coords mapped to NDC.
func quadToVertices(x, y, w, h float32, color mgl32.Vec3, extent vulkan.Extent2D) []overlayVertex {
	toNDC := func(px, py float32) mgl32.Vec2 {
		nx := (px/float32(extent.Width))*2 - 1
		ny := (py/float32(extent.Height))*2 - 1
		return mgl32.Vec2{nx, ny}
	}
	p0 := toNDC(x, y)
	p1 := toNDC(x+w, y)
	p2 := toNDC(x+w, y+h)
	p3 := toNDC(x, y+h)
	return []overlayVertex{
		{pos: p0, color: color},
		{pos: p1, color: color},
		{pos: p2, color: color},
		{pos: p2, color: color},
		{pos: p3, color: color},
		{pos: p0, color: color},
	}
}

// glyphPattern returns the bitmap rows for a supported character in the HUD font.
func glyphPattern(ch rune) []string {
	font := map[rune][]string{
		'0': {"111", "101", "101", "101", "111"},
		'1': {"010", "110", "010", "010", "111"},
		'2': {"111", "001", "111", "100", "111"},
		'3': {"111", "001", "111", "001", "111"},
		'4': {"101", "101", "111", "001", "001"},
		'5': {"111", "100", "111", "001", "111"},
		'6': {"111", "100", "111", "101", "111"},
		'7': {"111", "001", "001", "001", "001"},
		'8': {"111", "101", "111", "101", "111"},
		'9': {"111", "101", "111", "001", "111"},
		'F': {"111", "100", "110", "100", "100"},
		'P': {"111", "101", "111", "100", "100"},
		'S': {"111", "100", "111", "001", "111"},
		':': {"000", "010", "000", "010", "000"},
		'.': {"000", "000", "000", "000", "010"},
		' ': {"000", "000", "000", "000", "000"},
	}
	if p, ok := font[ch]; ok {
		return p
	}
	return font[' ']
}
