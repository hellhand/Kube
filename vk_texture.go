//go:build linux
// +build linux

package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"log"
	"strconv"
	"unsafe"

	"github.com/vulkan-go/vulkan"
)

func (a *VulkanApp) createTextureImage() error {
	texWidth, texHeight, pixels, err := loadVkcubeTexture()
	if err != nil {
		log.Printf("load embedded vkcube texture failed, using fallback checker: %v", err)
		texWidth, texHeight, pixels = fallbackCheckerTexture()
	}

	imageSize := vulkan.DeviceSize(len(pixels))
	stageBuf, stageMem, err := a.createBuffer(imageSize, vulkan.BufferUsageFlags(vulkan.BufferUsageTransferSrcBit), vulkan.MemoryPropertyHostVisibleBit|vulkan.MemoryPropertyHostCoherentBit)
	if err != nil {
		return fmt.Errorf("create staging buffer: %w", err)
	}
	defer vulkan.DestroyBuffer(a.device, stageBuf, nil)
	defer vulkan.FreeMemory(a.device, stageMem, nil)

	var data unsafe.Pointer
	if res := vulkan.MapMemory(a.device, stageMem, 0, imageSize, 0, &data); res != vulkan.Success {
		return fmt.Errorf("map staging buffer: %w", vulkan.Error(res))
	}
	dst := (*[1 << 30]byte)(data)[:imageSize:imageSize]
	copy(dst, pixels)
	vulkan.UnmapMemory(a.device, stageMem)

	image, memory, err := a.createImage(texWidth, texHeight, vulkan.FormatR8g8b8a8Srgb, vulkan.ImageTilingOptimal, vulkan.ImageUsageFlags(vulkan.ImageUsageTransferDstBit|vulkan.ImageUsageSampledBit), vulkan.MemoryPropertyDeviceLocalBit)
	if err != nil {
		return fmt.Errorf("create texture image: %w", err)
	}

	if err := a.transitionImageLayout(image, vulkan.FormatR8g8b8a8Srgb, vulkan.ImageLayoutUndefined, vulkan.ImageLayoutTransferDstOptimal); err != nil {
		vulkan.DestroyImage(a.device, image, nil)
		vulkan.FreeMemory(a.device, memory, nil)
		return err
	}
	if err := a.copyBufferToImage(stageBuf, image, texWidth, texHeight); err != nil {
		vulkan.DestroyImage(a.device, image, nil)
		vulkan.FreeMemory(a.device, memory, nil)
		return err
	}
	if err := a.transitionImageLayout(image, vulkan.FormatR8g8b8a8Srgb, vulkan.ImageLayoutTransferDstOptimal, vulkan.ImageLayoutShaderReadOnlyOptimal); err != nil {
		vulkan.DestroyImage(a.device, image, nil)
		vulkan.FreeMemory(a.device, memory, nil)
		return err
	}

	a.textureImage = image
	a.textureImageMemory = memory
	return nil
}

func (a *VulkanApp) createTextureImageView() error {
	view, err := a.createImageView(a.textureImage, vulkan.FormatR8g8b8a8Srgb, vulkan.ImageAspectFlags(vulkan.ImageAspectColorBit))
	if err != nil {
		return err
	}
	a.textureImageView = view
	return nil
}

func (a *VulkanApp) createTextureSampler() error {
	samplerInfo := vulkan.SamplerCreateInfo{
		SType:                   vulkan.StructureTypeSamplerCreateInfo,
		MagFilter:               vulkan.FilterLinear,
		MinFilter:               vulkan.FilterLinear,
		AddressModeU:            vulkan.SamplerAddressModeRepeat,
		AddressModeV:            vulkan.SamplerAddressModeRepeat,
		AddressModeW:            vulkan.SamplerAddressModeRepeat,
		AnisotropyEnable:        vulkan.False,
		MaxAnisotropy:           1.0,
		BorderColor:             vulkan.BorderColorIntOpaqueBlack,
		UnnormalizedCoordinates: vulkan.False,
		CompareEnable:           vulkan.False,
		CompareOp:               vulkan.CompareOpAlways,
		MipmapMode:              vulkan.SamplerMipmapModeLinear,
		MipLodBias:              0,
		MinLod:                  0,
		MaxLod:                  0,
	}
	var zero vulkan.Sampler
	samplerOut := (*vulkan.Sampler)(C.malloc(C.size_t(unsafe.Sizeof(zero))))
	if samplerOut == nil {
		return fmt.Errorf("allocate sampler handle")
	}
	defer C.free(unsafe.Pointer(samplerOut))

	if res := vulkan.CreateSampler(a.device, &samplerInfo, nil, samplerOut); res != vulkan.Success {
		return fmt.Errorf("create sampler: %w", vulkan.Error(res))
	}
	a.textureSampler = *samplerOut
	return nil
}

func loadVkcubeTexture() (uint32, uint32, []byte, error) {
	if len(lunargPPM) == 0 {
		return 0, 0, nil, fmt.Errorf("embedded texture payload missing")
	}
	w, h, rgb, err := parsePPM(lunargPPM)
	if err != nil {
		return 0, 0, nil, err
	}
	pixelCount := int(w * h)
	if len(rgb) != pixelCount*3 {
		return 0, 0, nil, fmt.Errorf("unexpected ppm pixel length: got %d want %d", len(rgb), pixelCount*3)
	}
	rgba := make([]byte, pixelCount*4)
	for i := 0; i < pixelCount; i++ {
		copy(rgba[i*4:], rgb[i*3:i*3+3])
		rgba[i*4+3] = 255
	}
	return w, h, rgba, nil
}

func parsePPM(data []byte) (uint32, uint32, []byte, error) {
	if len(data) < 3 || data[0] != 'P' || data[1] != '6' {
		return 0, 0, nil, fmt.Errorf("not a P6 ppm")
	}
	idx := 2
	var tokens []string
	for len(tokens) < 3 && idx < len(data) {
		for idx < len(data) && isSpace(data[idx]) {
			idx++
		}
		start := idx
		for idx < len(data) && !isSpace(data[idx]) {
			idx++
		}
		if start < idx {
			tokens = append(tokens, string(data[start:idx]))
		}
	}
	if len(tokens) < 3 {
		return 0, 0, nil, fmt.Errorf("ppm header incomplete")
	}
	width, err := strconv.Atoi(tokens[0])
	if err != nil {
		return 0, 0, nil, fmt.Errorf("ppm width: %w", err)
	}
	height, err := strconv.Atoi(tokens[1])
	if err != nil {
		return 0, 0, nil, fmt.Errorf("ppm height: %w", err)
	}
	maxVal, err := strconv.Atoi(tokens[2])
	if err != nil {
		return 0, 0, nil, fmt.Errorf("ppm max value: %w", err)
	}
	if maxVal != 255 {
		return 0, 0, nil, fmt.Errorf("unsupported max value %d", maxVal)
	}
	for idx < len(data) && isSpace(data[idx]) {
		idx++
	}
	pixels := data[idx:]
	expected := width * height * 3
	if len(pixels) < expected {
		return 0, 0, nil, fmt.Errorf("ppm data truncated: got %d expected %d", len(pixels), expected)
	}
	return uint32(width), uint32(height), pixels[:expected], nil
}

func fallbackCheckerTexture() (uint32, uint32, []byte) {
	texWidth, texHeight := uint32(2), uint32(2)
	pixels := []byte{
		255, 255, 255, 255, 50, 50, 50, 255,
		50, 50, 50, 255, 255, 255, 255, 255,
	}
	return texWidth, texHeight, pixels
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\n' || b == '\r' || b == '\t'
}
