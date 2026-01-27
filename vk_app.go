//go:build linux
// +build linux

package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"reflect"
	"runtime"
	"time"
	"unsafe"

	mgl32 "github.com/go-gl/mathgl/mgl32"
	"github.com/vulkan-go/glfw/v3.3/glfw"
	"github.com/vulkan-go/vulkan"
)

const (
	maxFramesInFlight = 2
)

var (
	validationLayers = []string{"VK_LAYER_KHRONOS_validation"}
	deviceExtensions = []string{"VK_KHR_swapchain"}
)

type vertex struct {
	pos   mgl32.Vec3
	color mgl32.Vec3
	uv    mgl32.Vec2
}

type cString struct {
	str  string
	cptr *C.char
}

func makeCString(s string) cString {
	c := C.CString(s)
	var goStr string
	h := (*reflect.StringHeader)(unsafe.Pointer(&goStr))
	h.Data = uintptr(unsafe.Pointer(c))
	h.Len = len(s)
	return cString{str: goStr, cptr: c}
}

func makeCStringSlice(src []string) ([]string, []*C.char) {
	out := make([]string, len(src))
	ptrs := make([]*C.char, len(src))
	for i, s := range src {
		cs := makeCString(s)
		out[i] = cs.str
		ptrs[i] = cs.cptr
	}
	return out, ptrs
}

func freeCStrings(ptrs []*C.char) {
	for _, p := range ptrs {
		if p != nil {
			C.free(unsafe.Pointer(p))
		}
	}
}

// 24 vertices (4 per face) to allow correct UVs per face.
var cubeVertices = []vertex{
	// Back face (Z-)
	{pos: mgl32.Vec3{-1, -1, -1}, color: mgl32.Vec3{1, 0, 0}, uv: mgl32.Vec2{0, 0}},
	{pos: mgl32.Vec3{1, -1, -1}, color: mgl32.Vec3{1, 0, 0}, uv: mgl32.Vec2{1, 0}},
	{pos: mgl32.Vec3{1, 1, -1}, color: mgl32.Vec3{1, 0, 0}, uv: mgl32.Vec2{1, 1}},
	{pos: mgl32.Vec3{-1, 1, -1}, color: mgl32.Vec3{1, 0, 0}, uv: mgl32.Vec2{0, 1}},
	// Front face (Z+)
	{pos: mgl32.Vec3{-1, -1, 1}, color: mgl32.Vec3{0, 1, 0}, uv: mgl32.Vec2{0, 0}},
	{pos: mgl32.Vec3{1, -1, 1}, color: mgl32.Vec3{0, 1, 0}, uv: mgl32.Vec2{1, 0}},
	{pos: mgl32.Vec3{1, 1, 1}, color: mgl32.Vec3{0, 1, 0}, uv: mgl32.Vec2{1, 1}},
	{pos: mgl32.Vec3{-1, 1, 1}, color: mgl32.Vec3{0, 1, 0}, uv: mgl32.Vec2{0, 1}},
	// Bottom face (Y-)
	{pos: mgl32.Vec3{-1, -1, -1}, color: mgl32.Vec3{0, 0, 1}, uv: mgl32.Vec2{0, 0}},
	{pos: mgl32.Vec3{1, -1, -1}, color: mgl32.Vec3{0, 0, 1}, uv: mgl32.Vec2{1, 0}},
	{pos: mgl32.Vec3{1, -1, 1}, color: mgl32.Vec3{0, 0, 1}, uv: mgl32.Vec2{1, 1}},
	{pos: mgl32.Vec3{-1, -1, 1}, color: mgl32.Vec3{0, 0, 1}, uv: mgl32.Vec2{0, 1}},
	// Top face (Y+)
	{pos: mgl32.Vec3{-1, 1, -1}, color: mgl32.Vec3{1, 1, 0}, uv: mgl32.Vec2{0, 0}},
	{pos: mgl32.Vec3{1, 1, -1}, color: mgl32.Vec3{1, 1, 0}, uv: mgl32.Vec2{1, 0}},
	{pos: mgl32.Vec3{1, 1, 1}, color: mgl32.Vec3{1, 1, 0}, uv: mgl32.Vec2{1, 1}},
	{pos: mgl32.Vec3{-1, 1, 1}, color: mgl32.Vec3{1, 1, 0}, uv: mgl32.Vec2{0, 1}},
	// Left face (X-)
	{pos: mgl32.Vec3{-1, -1, 1}, color: mgl32.Vec3{1, 0, 1}, uv: mgl32.Vec2{0, 0}},
	{pos: mgl32.Vec3{-1, -1, -1}, color: mgl32.Vec3{1, 0, 1}, uv: mgl32.Vec2{1, 0}},
	{pos: mgl32.Vec3{-1, 1, -1}, color: mgl32.Vec3{1, 0, 1}, uv: mgl32.Vec2{1, 1}},
	{pos: mgl32.Vec3{-1, 1, 1}, color: mgl32.Vec3{1, 0, 1}, uv: mgl32.Vec2{0, 1}},
	// Right face (X+)
	{pos: mgl32.Vec3{1, -1, -1}, color: mgl32.Vec3{0, 1, 1}, uv: mgl32.Vec2{0, 0}},
	{pos: mgl32.Vec3{1, -1, 1}, color: mgl32.Vec3{0, 1, 1}, uv: mgl32.Vec2{1, 0}},
	{pos: mgl32.Vec3{1, 1, 1}, color: mgl32.Vec3{0, 1, 1}, uv: mgl32.Vec2{1, 1}},
	{pos: mgl32.Vec3{1, 1, -1}, color: mgl32.Vec3{0, 1, 1}, uv: mgl32.Vec2{0, 1}},
}

var cubeIndices = []uint32{
	0, 1, 2, 2, 3, 0, // back
	4, 5, 6, 6, 7, 4, // front
	8, 9, 10, 10, 11, 8, // bottom
	12, 13, 14, 14, 15, 12, // top
	16, 17, 18, 18, 19, 16, // left
	20, 21, 22, 22, 23, 20, // right
}

type uniformBufferObject struct {
	Model mgl32.Mat4
	View  mgl32.Mat4
	Proj  mgl32.Mat4
}

type appConfig struct {
	enableValidation bool
}

type queueFamilyIndices struct {
	graphicsFamily uint32
	presentFamily  uint32
	hasGraphics    bool
	hasPresent     bool
}

type swapchainSupport struct {
	capabilities vulkan.SurfaceCapabilities
	formats      []vulkan.SurfaceFormat
	presentModes []vulkan.PresentMode
}

type VulkanApp struct {
	cfg                  appConfig
	window               *glfw.Window
	instance             vulkan.Instance
	debugCallback        vulkan.DebugReportCallback
	surface              vulkan.Surface
	physicalDevice       vulkan.PhysicalDevice
	device               vulkan.Device
	graphicsQueue        vulkan.Queue
	presentQueue         vulkan.Queue
	queues               queueFamilyIndices
	swapchain            vulkan.Swapchain
	swapchainImages      []vulkan.Image
	swapchainFormat      vulkan.Format
	swapchainExtent      vulkan.Extent2D
	swapchainViews       []vulkan.ImageView
	depthFormat          vulkan.Format
	depthImage           vulkan.Image
	depthImageMemory     vulkan.DeviceMemory
	depthImageView       vulkan.ImageView
	renderPass           vulkan.RenderPass
	pipelineLayout       vulkan.PipelineLayout
	pipeline             vulkan.Pipeline
	descriptorSetLayout  vulkan.DescriptorSetLayout
	descriptorPool       vulkan.DescriptorPool
	descriptorSets       []vulkan.DescriptorSet
	uniformBuffers       []vulkan.Buffer
	uniformBuffersMemory []vulkan.DeviceMemory
	textureImage         vulkan.Image
	textureImageMemory   vulkan.DeviceMemory
	textureImageView     vulkan.ImageView
	textureSampler       vulkan.Sampler
	vertexBuffer         vulkan.Buffer
	vertexBufferMemory   vulkan.DeviceMemory
	indexBuffer          vulkan.Buffer
	indexBufferMemory    vulkan.DeviceMemory
	framebuffers         []vulkan.Framebuffer
	commandPool          vulkan.CommandPool
	commandBuffers       []vulkan.CommandBuffer
	imageAvailable       []vulkan.Semaphore
	renderFinished       []vulkan.Semaphore
	inFlightFences       []vulkan.Fence
	imagesInFlight       []vulkan.Fence
	currentFrame         int
	debugFrames          int
	framebufferResized   bool
	startTime            time.Time
}

func newVulkanApp(window *glfw.Window) (*VulkanApp, error) {
	cfg := appConfig{
		enableValidation: enableValidationLayers(),
	}

	app := &VulkanApp{
		cfg:    cfg,
		window: window,
	}

	if err := app.initVulkan(); err != nil {
		return nil, err
	}
	return app, nil
}

func enableValidationLayers() bool {
	val := os.Getenv("VK_VALIDATION")
	if val == "" {
		return true
	}
	switch val {
	case "0", "false", "False", "FALSE":
		return false
	default:
		return true
	}
}

func (a *VulkanApp) initVulkan() error {
	vulkan.SetGetInstanceProcAddr(glfw.GetVulkanGetInstanceProcAddress())
	if err := vulkan.Init(); err != nil {
		return fmt.Errorf("vulkan init: %w", err)
	}
	if err := a.createInstance(); err != nil {
		return err
	}
	log.Printf("Instance created")
	if err := vulkan.InitInstance(a.instance); err != nil {
		return fmt.Errorf("vkInitInstance: %w", err)
	}
	if err := a.setupDebugCallback(); err != nil {
		return err
	}
	log.Printf("Debug callback ready")
	if err := a.createSurface(); err != nil {
		return err
	}
	log.Printf("Surface created")
	if err := a.pickPhysicalDevice(); err != nil {
		return err
	}
	log.Printf("Physical device picked")
	if err := a.createLogicalDevice(); err != nil {
		return err
	}
	log.Printf("Logical device created")
	if err := a.createSwapchain(); err != nil {
		return err
	}
	log.Printf("Swapchain created")
	if err := a.createImageViews(); err != nil {
		return err
	}
	log.Printf("Image views created")
	if err := a.createDepthResources(); err != nil {
		return err
	}
	log.Printf("Depth resources created")
	if err := a.createRenderPass(); err != nil {
		return err
	}
	log.Printf("Render pass created")
	if err := a.createDescriptorSetLayout(); err != nil {
		return err
	}
	log.Printf("Descriptor set layout created")
	if err := a.createGraphicsPipeline(); err != nil {
		return err
	}
	log.Printf("Graphics pipeline created")
	if err := a.createFramebuffers(); err != nil {
		return err
	}
	log.Printf("Framebuffers created")
	if err := a.createCommandPool(); err != nil {
		return err
	}
	log.Printf("Command pool created")
	if err := a.createVertexBuffer(); err != nil {
		return err
	}
	log.Printf("Vertex buffer created")
	if err := a.createIndexBuffer(); err != nil {
		return err
	}
	log.Printf("Index buffer created")
	if err := a.createTextureImage(); err != nil {
		return err
	}
	log.Printf("Texture image created")
	if err := a.createTextureImageView(); err != nil {
		return err
	}
	log.Printf("Texture view created")
	if err := a.createTextureSampler(); err != nil {
		return err
	}
	log.Printf("Texture sampler created")
	if err := a.createUniformBuffers(); err != nil {
		return err
	}
	log.Printf("Uniform buffers created")
	if err := a.createDescriptorPool(); err != nil {
		return err
	}
	log.Printf("Descriptor pool created")
	if err := a.createDescriptorSets(); err != nil {
		return err
	}
	log.Printf("Descriptor sets created")
	if err := a.allocateCommandBuffers(); err != nil {
		return err
	}
	log.Printf("Command buffers allocated")
	if err := a.createSyncObjects(); err != nil {
		return err
	}
	a.startTime = time.Now()
	log.Printf("Vulkan initialization complete (swapchain images: %d)", len(a.swapchainImages))
	return nil
}

func (a *VulkanApp) createInstance() error {
	if !glfw.VulkanSupported() {
		return errors.New("GLFW Vulkan loader not found")
	}

	if a.cfg.enableValidation && !a.validationLayersSupported() {
		log.Printf("validation layers not available; continuing without them")
		a.cfg.enableValidation = false
	}

	appName := makeCString("Kube Vulkan")
	engineName := makeCString("No Engine")
	defer C.free(unsafe.Pointer(appName.cptr))
	defer C.free(unsafe.Pointer(engineName.cptr))

	appInfo := vulkan.ApplicationInfo{
		SType:              vulkan.StructureTypeApplicationInfo,
		PApplicationName:   appName.str,
		ApplicationVersion: vulkan.MakeVersion(0, 1, 0),
		PEngineName:        engineName.str,
		EngineVersion:      vulkan.MakeVersion(0, 1, 0),
		ApiVersion:         vulkan.MakeVersion(1, 1, 0),
	}

	rawExts := a.window.GetRequiredInstanceExtensions()
	extensions, cExtPtrs := makeCStringSlice(rawExts)
	if a.cfg.enableValidation {
		cs := makeCString("VK_EXT_debug_report")
		extensions = append(extensions, cs.str)
		cExtPtrs = append(cExtPtrs, cs.cptr)
	}
	defer freeCStrings(cExtPtrs)

	var cLayerPtrs []*C.char
	if a.cfg.enableValidation {
		_, cLayerPtrs = makeCStringSlice(validationLayers)
		defer freeCStrings(cLayerPtrs)
	}

	createInfo := vulkan.InstanceCreateInfo{
		SType:                   vulkan.StructureTypeInstanceCreateInfo,
		PApplicationInfo:        &appInfo,
		EnabledExtensionCount:   uint32(len(extensions)),
		PpEnabledExtensionNames: extensions,
	}
	if a.cfg.enableValidation {
		layerNames, cPtrs := makeCStringSlice(validationLayers)
		cLayerPtrs = append(cLayerPtrs, cPtrs...)
		createInfo.EnabledLayerCount = uint32(len(layerNames))
		createInfo.PpEnabledLayerNames = layerNames
	}

	var pin runtime.Pinner
	pin.Pin(&createInfo)
	pin.Pin(&appInfo)
	if len(extensions) > 0 {
		pin.Pin(&extensions[0])
	}
	if a.cfg.enableValidation && createInfo.EnabledLayerCount > 0 {
		pin.Pin(&createInfo.PpEnabledLayerNames[0])
	}
	defer pin.Unpin()

	var zeroInstance vulkan.Instance
	instanceOut := (*vulkan.Instance)(C.malloc(C.size_t(unsafe.Sizeof(zeroInstance))))
	if instanceOut == nil {
		return fmt.Errorf("create instance: failed to allocate output handle")
	}
	defer C.free(unsafe.Pointer(instanceOut))

	if res := vulkan.CreateInstance(&createInfo, nil, instanceOut); res != vulkan.Success {
		return fmt.Errorf("create instance: %w", vulkan.Error(res))
	}
	a.instance = *instanceOut
	return nil
}

func (a *VulkanApp) validationLayersSupported() bool {
	var count uint32
	if vulkan.EnumerateInstanceLayerProperties(&count, nil) != vulkan.Success {
		return false
	}
	props := make([]vulkan.LayerProperties, count)
	if vulkan.EnumerateInstanceLayerProperties(&count, props) != vulkan.Success {
		return false
	}
	supported := make(map[string]bool)
	for i := range props {
		props[i].Deref()
		name := vulkan.ToString(props[i].LayerName[:])
		supported[name] = true
	}
	for _, l := range validationLayers {
		if !supported[l] {
			return false
		}
	}
	return true
}

func (a *VulkanApp) setupDebugCallback() error {
	if !a.cfg.enableValidation {
		return nil
	}
	createInfo := vulkan.DebugReportCallbackCreateInfo{
		SType: vulkan.StructureTypeDebugReportCallbackCreateInfo,
		Flags: vulkan.DebugReportFlags(
			vulkan.DebugReportErrorBit |
				vulkan.DebugReportWarningBit |
				vulkan.DebugReportPerformanceWarningBit),
		PfnCallback: func(flags vulkan.DebugReportFlags, objectType vulkan.DebugReportObjectType, object uint64, location uint, messageCode int32, layerPrefix string, message string, userData unsafe.Pointer) vulkan.Bool32 {
			log.Printf("[VK][%s][0x%x] %s (code=%d)", layerPrefix, flags, message, messageCode)
			return vulkan.False
		},
	}
	if res := vulkan.CreateDebugReportCallback(a.instance, &createInfo, nil, &a.debugCallback); res != vulkan.Success {
		return fmt.Errorf("create debug callback: %w", vulkan.Error(res))
	}
	return nil
}

func (a *VulkanApp) createSurface() error {
	surfacePtr, err := a.window.CreateWindowSurface(a.instance, nil)
	if err != nil {
		return fmt.Errorf("create window surface: %w", err)
	}
	a.surface = vulkan.SurfaceFromPointer(surfacePtr)
	return nil
}

func (a *VulkanApp) pickPhysicalDevice() error {
	var count uint32
	if res := vulkan.EnumeratePhysicalDevices(a.instance, &count, nil); res != vulkan.Success || count == 0 {
		return fmt.Errorf("enumerate physical devices: %w", vulkan.Error(res))
	}
	devices := make([]vulkan.PhysicalDevice, count)
	if res := vulkan.EnumeratePhysicalDevices(a.instance, &count, devices); res != vulkan.Success {
		return fmt.Errorf("enumerate physical devices list: %w", vulkan.Error(res))
	}

	var selected vulkan.PhysicalDevice
	var selectedQueues queueFamilyIndices
	bestScore := int32(-1)
	for _, dev := range devices {
		q := a.findQueueFamilies(dev)
		if !q.hasGraphics || !q.hasPresent {
			continue
		}
		if !a.deviceExtensionsSupported(dev) {
			continue
		}
		support := a.querySwapchainSupport(dev)
		if len(support.formats) == 0 || len(support.presentModes) == 0 {
			continue
		}
		score := a.deviceScore(dev)
		if score > bestScore {
			bestScore = score
			selected = dev
			selectedQueues = q
		}
	}

	if selected == (vulkan.PhysicalDevice)(unsafe.Pointer(nil)) {
		return errors.New("no suitable GPU found")
	}

	a.physicalDevice = selected
	a.queues = selectedQueues
	return nil
}

func (a *VulkanApp) deviceScore(device vulkan.PhysicalDevice) int32 {
	var props vulkan.PhysicalDeviceProperties
	vulkan.GetPhysicalDeviceProperties(device, &props)
	props.Deref()

	switch props.DeviceType {
	case vulkan.PhysicalDeviceTypeDiscreteGpu:
		return 1000
	case vulkan.PhysicalDeviceTypeIntegratedGpu:
		return 500
	default:
		return 100
	}
}

func (a *VulkanApp) deviceExtensionsSupported(device vulkan.PhysicalDevice) bool {
	var count uint32
	if res := vulkan.EnumerateDeviceExtensionProperties(device, "", &count, nil); res != vulkan.Success {
		return false
	}
	props := make([]vulkan.ExtensionProperties, count)
	if res := vulkan.EnumerateDeviceExtensionProperties(device, "", &count, props); res != vulkan.Success {
		return false
	}
	supported := make(map[string]bool)
	for i := range props {
		props[i].Deref()
		name := vulkan.ToString(props[i].ExtensionName[:])
		supported[name] = true
	}
	for _, ext := range deviceExtensions {
		if !supported[ext] {
			return false
		}
	}
	return true
}

func (a *VulkanApp) findQueueFamilies(device vulkan.PhysicalDevice) queueFamilyIndices {
	var count uint32
	vulkan.GetPhysicalDeviceQueueFamilyProperties(device, &count, nil)
	props := make([]vulkan.QueueFamilyProperties, count)
	vulkan.GetPhysicalDeviceQueueFamilyProperties(device, &count, props)

	var indices queueFamilyIndices
	for i := range props {
		props[i].Deref()
		if props[i].QueueFlags&vulkan.QueueFlags(vulkan.QueueGraphicsBit) != 0 {
			indices.graphicsFamily = uint32(i)
			indices.hasGraphics = true
		}
		var present vulkan.Bool32
		vulkan.GetPhysicalDeviceSurfaceSupport(device, uint32(i), a.surface, &present)
		if present == vulkan.True {
			indices.presentFamily = uint32(i)
			indices.hasPresent = true
		}
		if indices.hasGraphics && indices.hasPresent {
			break
		}
	}
	return indices
}

func (a *VulkanApp) createLogicalDevice() error {
	queueInfos := []vulkan.DeviceQueueCreateInfo{}
	uniqueFamilies := map[uint32]bool{
		a.queues.graphicsFamily: true,
		a.queues.presentFamily:  true,
	}
	priority := float32(1.0)
	for family := range uniqueFamilies {
		queueInfos = append(queueInfos, vulkan.DeviceQueueCreateInfo{
			SType:            vulkan.StructureTypeDeviceQueueCreateInfo,
			QueueFamilyIndex: family,
			QueueCount:       1,
			PQueuePriorities: []float32{priority},
		})
	}

	deviceFeatures := vulkan.PhysicalDeviceFeatures{}
	extNames, extPtrs := makeCStringSlice(deviceExtensions)
	defer freeCStrings(extPtrs)

	createInfo := vulkan.DeviceCreateInfo{
		SType:                   vulkan.StructureTypeDeviceCreateInfo,
		PQueueCreateInfos:       queueInfos,
		QueueCreateInfoCount:    uint32(len(queueInfos)),
		PEnabledFeatures:        []vulkan.PhysicalDeviceFeatures{deviceFeatures},
		PpEnabledExtensionNames: extNames,
		EnabledExtensionCount:   uint32(len(extNames)),
	}

	var layerNames []string
	var layerPtrs []*C.char
	if a.cfg.enableValidation {
		layerNames, layerPtrs = makeCStringSlice(validationLayers)
		createInfo.EnabledLayerCount = uint32(len(layerNames))
		createInfo.PpEnabledLayerNames = layerNames
	}
	defer freeCStrings(layerPtrs)

	var pin runtime.Pinner
	defer pin.Unpin()

	pin.Pin(&createInfo)
	pin.Pin(&queueInfos[0])
	for i := range queueInfos {
		if len(queueInfos[i].PQueuePriorities) > 0 {
			pin.Pin(&queueInfos[i].PQueuePriorities[0])
		}
	}
	pin.Pin(&createInfo.PQueueCreateInfos[0])
	if len(extNames) > 0 {
		pin.Pin(&extNames[0])
	}
	if a.cfg.enableValidation && len(layerNames) > 0 {
		pin.Pin(&layerNames[0])
	}
	pin.Pin(&createInfo.PEnabledFeatures[0])

	var zeroDevice vulkan.Device
	deviceOut := (*vulkan.Device)(C.malloc(C.size_t(unsafe.Sizeof(zeroDevice))))
	if deviceOut == nil {
		return fmt.Errorf("allocate device handle")
	}
	defer C.free(unsafe.Pointer(deviceOut))

	if res := vulkan.CreateDevice(a.physicalDevice, &createInfo, nil, deviceOut); res != vulkan.Success {
		return fmt.Errorf("create logical device: %w", vulkan.Error(res))
	}

	a.device = *deviceOut

	var zeroQueue vulkan.Queue
	queueOut := (*vulkan.Queue)(C.malloc(C.size_t(unsafe.Sizeof(zeroQueue))))
	if queueOut == nil {
		return fmt.Errorf("allocate graphics queue handle")
	}
	defer C.free(unsafe.Pointer(queueOut))
	vulkan.GetDeviceQueue(a.device, a.queues.graphicsFamily, 0, queueOut)
	a.graphicsQueue = *queueOut

	queuePresentOut := (*vulkan.Queue)(C.malloc(C.size_t(unsafe.Sizeof(zeroQueue))))
	if queuePresentOut == nil {
		return fmt.Errorf("allocate present queue handle")
	}
	defer C.free(unsafe.Pointer(queuePresentOut))
	vulkan.GetDeviceQueue(a.device, a.queues.presentFamily, 0, queuePresentOut)
	a.presentQueue = *queuePresentOut
	return nil
}

func (a *VulkanApp) querySwapchainSupport(device vulkan.PhysicalDevice) swapchainSupport {
	var details swapchainSupport
	log.Printf("querySwapchainSupport: capabilities")
	vulkan.GetPhysicalDeviceSurfaceCapabilities(device, a.surface, &details.capabilities)
	details.capabilities.Deref()

	var formatCount uint32
	log.Printf("querySwapchainSupport: formats")
	vulkan.GetPhysicalDeviceSurfaceFormats(device, a.surface, &formatCount, nil)
	if formatCount > 0 {
		details.formats = make([]vulkan.SurfaceFormat, formatCount)
		vulkan.GetPhysicalDeviceSurfaceFormats(device, a.surface, &formatCount, details.formats)
		for i := range details.formats {
			details.formats[i].Deref()
		}
	}

	var presentCount uint32
	log.Printf("querySwapchainSupport: present modes")
	vulkan.GetPhysicalDeviceSurfacePresentModes(device, a.surface, &presentCount, nil)
	if presentCount > 0 {
		details.presentModes = make([]vulkan.PresentMode, presentCount)
		vulkan.GetPhysicalDeviceSurfacePresentModes(device, a.surface, &presentCount, details.presentModes)
	}

	log.Printf("querySwapchainSupport: caps done (formats=%d presentModes=%d)", len(details.formats), len(details.presentModes))
	return details
}

func chooseSwapSurfaceFormat(available []vulkan.SurfaceFormat) vulkan.SurfaceFormat {
	for _, f := range available {
		if f.Format == vulkan.FormatB8g8r8a8Srgb && f.ColorSpace == vulkan.ColorSpaceSrgbNonlinear {
			return f
		}
	}
	return available[0]
}

func chooseSwapPresentMode(available []vulkan.PresentMode) vulkan.PresentMode {
	for _, m := range available {
		if m == vulkan.PresentModeMailbox {
			return m
		}
	}
	return vulkan.PresentModeFifo
}

func chooseSwapExtent(caps vulkan.SurfaceCapabilities, window *glfw.Window) vulkan.Extent2D {
	if caps.CurrentExtent.Width != math.MaxUint32 && caps.CurrentExtent.Width != 0 && caps.CurrentExtent.Height != 0 {
		return caps.CurrentExtent
	}
	w, h := window.GetFramebufferSize()
	extent := vulkan.Extent2D{
		Width:  uint32(w),
		Height: uint32(h),
	}
	if extent.Width == 0 {
		extent.Width = 800
	}
	if extent.Height == 0 {
		extent.Height = 600
	}
	min := caps.MinImageExtent
	max := caps.MaxImageExtent
	if max.Width == 0 {
		max.Width = extent.Width
	}
	if max.Height == 0 {
		max.Height = extent.Height
	}
	if min.Width == 0 {
		min.Width = 1
	}
	if min.Height == 0 {
		min.Height = 1
	}
	extent.Width = uint32(clamp(uint64(extent.Width), uint64(min.Width), uint64(max.Width)))
	extent.Height = uint32(clamp(uint64(extent.Height), uint64(min.Height), uint64(max.Height)))
	return extent
}

func (a *VulkanApp) createSwapchain() error {
	// Ensure the window has a non-zero framebuffer before querying swapchain support.
	for {
		w, h := a.window.GetFramebufferSize()
		if w > 0 && h > 0 {
			break
		}
		glfw.WaitEventsTimeout(0.01)
	}

	log.Printf("createSwapchain: querying support")
	support := a.querySwapchainSupport(a.physicalDevice)

	surfaceFormat := chooseSwapSurfaceFormat(support.formats)
	presentMode := chooseSwapPresentMode(support.presentModes)
	extent := chooseSwapExtent(support.capabilities, a.window)

	imageCount := support.capabilities.MinImageCount + 1
	if support.capabilities.MaxImageCount > 0 && imageCount > support.capabilities.MaxImageCount {
		imageCount = support.capabilities.MaxImageCount
	}

	log.Printf("createSwapchain: creating swapchain extent=%dx%d images=%d format=%v mode=%v", extent.Width, extent.Height, imageCount, surfaceFormat.Format, presentMode)

	createInfo := vulkan.SwapchainCreateInfo{
		SType:            vulkan.StructureTypeSwapchainCreateInfo,
		Surface:          a.surface,
		MinImageCount:    imageCount,
		ImageFormat:      surfaceFormat.Format,
		ImageColorSpace:  surfaceFormat.ColorSpace,
		ImageExtent:      extent,
		ImageArrayLayers: 1,
		ImageUsage:       vulkan.ImageUsageFlags(vulkan.ImageUsageColorAttachmentBit),
		PreTransform:     support.capabilities.CurrentTransform,
		CompositeAlpha:   vulkan.CompositeAlphaOpaqueBit,
		PresentMode:      presentMode,
		Clipped:          vulkan.True,
		OldSwapchain:     vulkan.Swapchain(vulkan.NullHandle),
	}

	if a.queues.graphicsFamily != a.queues.presentFamily {
		indices := []uint32{a.queues.graphicsFamily, a.queues.presentFamily}
		createInfo.ImageSharingMode = vulkan.SharingModeConcurrent
		createInfo.QueueFamilyIndexCount = uint32(len(indices))
		createInfo.PQueueFamilyIndices = indices
	} else {
		createInfo.ImageSharingMode = vulkan.SharingModeExclusive
	}

	var zeroSwapchain vulkan.Swapchain
	swapOut := (*vulkan.Swapchain)(C.malloc(C.size_t(unsafe.Sizeof(zeroSwapchain))))
	if swapOut == nil {
		return fmt.Errorf("allocate swapchain handle")
	}
	defer C.free(unsafe.Pointer(swapOut))

	if res := vulkan.CreateSwapchain(a.device, &createInfo, nil, swapOut); res != vulkan.Success {
		return fmt.Errorf("create swapchain: %w", vulkan.Error(res))
	}
	a.swapchain = *swapOut

	var count uint32
	log.Printf("createSwapchain: querying images")
	vulkan.GetSwapchainImages(a.device, a.swapchain, &count, nil)
	a.swapchainImages = make([]vulkan.Image, count)
	vulkan.GetSwapchainImages(a.device, a.swapchain, &count, a.swapchainImages)
	a.swapchainFormat = surfaceFormat.Format
	a.swapchainExtent = extent
	log.Printf("createSwapchain: created with %d images", count)
	return nil
}

func (a *VulkanApp) createImageViews() error {
	a.swapchainViews = make([]vulkan.ImageView, len(a.swapchainImages))
	for i, img := range a.swapchainImages {
		view, err := a.createImageView(img, a.swapchainFormat, vulkan.ImageAspectFlags(vulkan.ImageAspectColorBit))
		if err != nil {
			return fmt.Errorf("create image view %d: %w", i, err)
		}
		a.swapchainViews[i] = view
	}
	return nil
}

func (a *VulkanApp) createDepthResources() error {
	if a.swapchainExtent.Width == 0 || a.swapchainExtent.Height == 0 {
		return fmt.Errorf("swapchain extent is zero; cannot create depth resources")
	}
	depthFormat, err := a.findDepthFormat()
	if err != nil {
		return err
	}
	a.depthFormat = depthFormat
	image, memory, err := a.createImage(a.swapchainExtent.Width, a.swapchainExtent.Height, depthFormat, vulkan.ImageTilingOptimal, vulkan.ImageUsageFlags(vulkan.ImageUsageDepthStencilAttachmentBit), vulkan.MemoryPropertyDeviceLocalBit)
	if err != nil {
		return fmt.Errorf("create depth image: %w", err)
	}
	view, err := a.createImageView(image, depthFormat, vulkan.ImageAspectFlags(vulkan.ImageAspectDepthBit))
	if err != nil {
		vulkan.DestroyImage(a.device, image, nil)
		vulkan.FreeMemory(a.device, memory, nil)
		return fmt.Errorf("create depth image view: %w", err)
	}
	a.depthImage = image
	a.depthImageMemory = memory
	a.depthImageView = view
	return nil
}

func (a *VulkanApp) createTextureImage() error {
	// Simple 2x2 checker texture.
	texWidth, texHeight := uint32(2), uint32(2)
	pixels := []byte{
		255, 255, 255, 255, 50, 50, 50, 255,
		50, 50, 50, 255, 255, 255, 255, 255,
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

func (a *VulkanApp) findDepthFormat() (vulkan.Format, error) {
	candidates := []vulkan.Format{
		vulkan.FormatD32Sfloat,
		vulkan.FormatD32SfloatS8Uint,
		vulkan.FormatD24UnormS8Uint,
	}
	return a.findSupportedFormat(candidates, vulkan.ImageTilingOptimal, vulkan.FormatFeatureFlags(vulkan.FormatFeatureDepthStencilAttachmentBit))
}

func (a *VulkanApp) findSupportedFormat(candidates []vulkan.Format, tiling vulkan.ImageTiling, features vulkan.FormatFeatureFlags) (vulkan.Format, error) {
	for _, format := range candidates {
		var props vulkan.FormatProperties
		vulkan.GetPhysicalDeviceFormatProperties(a.physicalDevice, format, &props)
		props.Deref()
		if tiling == vulkan.ImageTilingLinear && props.LinearTilingFeatures&features == features {
			return format, nil
		}
		if tiling == vulkan.ImageTilingOptimal && props.OptimalTilingFeatures&features == features {
			return format, nil
		}
	}
	return 0, errors.New("no supported format found")
}

func (a *VulkanApp) createImage(width, height uint32, format vulkan.Format, tiling vulkan.ImageTiling, usage vulkan.ImageUsageFlags, properties vulkan.MemoryPropertyFlagBits) (vulkan.Image, vulkan.DeviceMemory, error) {
	createInfo := vulkan.ImageCreateInfo{
		SType:     vulkan.StructureTypeImageCreateInfo,
		ImageType: vulkan.ImageType2d,
		Extent: vulkan.Extent3D{
			Width:  width,
			Height: height,
			Depth:  1,
		},
		MipLevels:     1,
		ArrayLayers:   1,
		Format:        format,
		Tiling:        tiling,
		InitialLayout: vulkan.ImageLayoutUndefined,
		Usage:         usage,
		Samples:       vulkan.SampleCount1Bit,
		SharingMode:   vulkan.SharingModeExclusive,
	}

	var zeroImage vulkan.Image
	imageOut := (*vulkan.Image)(C.malloc(C.size_t(unsafe.Sizeof(zeroImage))))
	if imageOut == nil {
		return vulkan.Image(vulkan.NullHandle), vulkan.DeviceMemory(vulkan.NullHandle), fmt.Errorf("allocate image handle")
	}
	defer C.free(unsafe.Pointer(imageOut))

	if res := vulkan.CreateImage(a.device, &createInfo, nil, imageOut); res != vulkan.Success {
		return vulkan.Image(vulkan.NullHandle), vulkan.DeviceMemory(vulkan.NullHandle), fmt.Errorf("create image: %w", vulkan.Error(res))
	}

	var memRequirements vulkan.MemoryRequirements
	vulkan.GetImageMemoryRequirements(a.device, *imageOut, &memRequirements)
	memRequirements.Deref()

	memoryType, ok := a.findMemoryType(memRequirements.MemoryTypeBits, properties)
	if !ok {
		vulkan.DestroyImage(a.device, *imageOut, nil)
		return vulkan.Image(vulkan.NullHandle), vulkan.DeviceMemory(vulkan.NullHandle), fmt.Errorf("no suitable memory type for image")
	}

	allocInfo := vulkan.MemoryAllocateInfo{
		SType:           vulkan.StructureTypeMemoryAllocateInfo,
		AllocationSize:  memRequirements.Size,
		MemoryTypeIndex: memoryType,
	}

	var zeroMem vulkan.DeviceMemory
	memoryOut := (*vulkan.DeviceMemory)(C.malloc(C.size_t(unsafe.Sizeof(zeroMem))))
	if memoryOut == nil {
		vulkan.DestroyImage(a.device, *imageOut, nil)
		return vulkan.Image(vulkan.NullHandle), vulkan.DeviceMemory(vulkan.NullHandle), fmt.Errorf("allocate image memory handle")
	}
	defer C.free(unsafe.Pointer(memoryOut))

	if res := vulkan.AllocateMemory(a.device, &allocInfo, nil, memoryOut); res != vulkan.Success {
		vulkan.DestroyImage(a.device, *imageOut, nil)
		return vulkan.Image(vulkan.NullHandle), vulkan.DeviceMemory(vulkan.NullHandle), fmt.Errorf("allocate image memory: %w", vulkan.Error(res))
	}

	if res := vulkan.BindImageMemory(a.device, *imageOut, *memoryOut, 0); res != vulkan.Success {
		vulkan.FreeMemory(a.device, *memoryOut, nil)
		vulkan.DestroyImage(a.device, *imageOut, nil)
		return vulkan.Image(vulkan.NullHandle), vulkan.DeviceMemory(vulkan.NullHandle), fmt.Errorf("bind image memory: %w", vulkan.Error(res))
	}

	return *imageOut, *memoryOut, nil
}

func (a *VulkanApp) findMemoryType(typeFilter uint32, properties vulkan.MemoryPropertyFlagBits) (uint32, bool) {
	var memProps vulkan.PhysicalDeviceMemoryProperties
	vulkan.GetPhysicalDeviceMemoryProperties(a.physicalDevice, &memProps)
	memProps.Deref()

	for i := uint32(0); i < memProps.MemoryTypeCount; i++ {
		memoryType := memProps.MemoryTypes[i]
		memoryType.Deref()
		if typeFilter&(1<<i) != 0 && memoryType.PropertyFlags&vulkan.MemoryPropertyFlags(properties) == vulkan.MemoryPropertyFlags(properties) {
			return i, true
		}
	}
	return 0, false
}

func (a *VulkanApp) createImageView(image vulkan.Image, format vulkan.Format, aspectFlags vulkan.ImageAspectFlags) (vulkan.ImageView, error) {
	viewInfo := vulkan.ImageViewCreateInfo{
		SType:    vulkan.StructureTypeImageViewCreateInfo,
		Image:    image,
		ViewType: vulkan.ImageViewType2d,
		Format:   format,
		Components: vulkan.ComponentMapping{
			R: vulkan.ComponentSwizzleIdentity,
			G: vulkan.ComponentSwizzleIdentity,
			B: vulkan.ComponentSwizzleIdentity,
			A: vulkan.ComponentSwizzleIdentity,
		},
		SubresourceRange: vulkan.ImageSubresourceRange{
			AspectMask:     aspectFlags,
			BaseMipLevel:   0,
			LevelCount:     1,
			BaseArrayLayer: 0,
			LayerCount:     1,
		},
	}
	var zero vulkan.ImageView
	viewOut := (*vulkan.ImageView)(C.malloc(C.size_t(unsafe.Sizeof(zero))))
	if viewOut == nil {
		return vulkan.ImageView(vulkan.NullHandle), fmt.Errorf("allocate image view handle")
	}
	defer C.free(unsafe.Pointer(viewOut))
	if res := vulkan.CreateImageView(a.device, &viewInfo, nil, viewOut); res != vulkan.Success {
		return vulkan.ImageView(vulkan.NullHandle), fmt.Errorf("create image view: %w", vulkan.Error(res))
	}
	return *viewOut, nil
}

func (a *VulkanApp) transitionImageLayout(image vulkan.Image, format vulkan.Format, oldLayout, newLayout vulkan.ImageLayout) error {
	return a.oneTimeCommands(func(cb vulkan.CommandBuffer) {
		subresource := vulkan.ImageSubresourceRange{
			AspectMask:     vulkan.ImageAspectFlags(vulkan.ImageAspectColorBit),
			BaseMipLevel:   0,
			LevelCount:     1,
			BaseArrayLayer: 0,
			LayerCount:     1,
		}
		if newLayout == vulkan.ImageLayoutDepthStencilAttachmentOptimal {
			subresource.AspectMask = vulkan.ImageAspectFlags(vulkan.ImageAspectDepthBit)
		}
		barrier := vulkan.ImageMemoryBarrier{
			SType:               vulkan.StructureTypeImageMemoryBarrier,
			OldLayout:           oldLayout,
			NewLayout:           newLayout,
			SrcQueueFamilyIndex: vulkan.QueueFamilyIgnored,
			DstQueueFamilyIndex: vulkan.QueueFamilyIgnored,
			Image:               image,
			SubresourceRange:    subresource,
		}
		var srcStage, dstStage vulkan.PipelineStageFlags
		switch oldLayout {
		case vulkan.ImageLayoutUndefined:
			barrier.SrcAccessMask = 0
			srcStage = vulkan.PipelineStageFlags(vulkan.PipelineStageTopOfPipeBit)
		case vulkan.ImageLayoutTransferDstOptimal:
			barrier.SrcAccessMask = vulkan.AccessFlags(vulkan.AccessTransferWriteBit)
			srcStage = vulkan.PipelineStageFlags(vulkan.PipelineStageTransferBit)
		default:
			srcStage = vulkan.PipelineStageFlags(vulkan.PipelineStageTopOfPipeBit)
		}
		switch newLayout {
		case vulkan.ImageLayoutTransferDstOptimal:
			barrier.DstAccessMask = vulkan.AccessFlags(vulkan.AccessTransferWriteBit)
			dstStage = vulkan.PipelineStageFlags(vulkan.PipelineStageTransferBit)
		case vulkan.ImageLayoutShaderReadOnlyOptimal:
			barrier.DstAccessMask = vulkan.AccessFlags(vulkan.AccessShaderReadBit)
			dstStage = vulkan.PipelineStageFlags(vulkan.PipelineStageFragmentShaderBit)
		case vulkan.ImageLayoutDepthStencilAttachmentOptimal:
			barrier.DstAccessMask = vulkan.AccessFlags(vulkan.AccessDepthStencilAttachmentReadBit | vulkan.AccessDepthStencilAttachmentWriteBit)
			dstStage = vulkan.PipelineStageFlags(vulkan.PipelineStageEarlyFragmentTestsBit | vulkan.PipelineStageLateFragmentTestsBit)
		default:
			dstStage = vulkan.PipelineStageFlags(vulkan.PipelineStageBottomOfPipeBit)
		}
		vulkan.CmdPipelineBarrier(cb, srcStage, dstStage, 0, 0, nil, 0, nil, 1, []vulkan.ImageMemoryBarrier{barrier})
	})
}

func (a *VulkanApp) copyBufferToImage(buffer vulkan.Buffer, image vulkan.Image, width, height uint32) error {
	return a.oneTimeCommands(func(cb vulkan.CommandBuffer) {
		region := vulkan.BufferImageCopy{
			BufferOffset:      0,
			BufferRowLength:   0,
			BufferImageHeight: 0,
			ImageSubresource: vulkan.ImageSubresourceLayers{
				AspectMask:     vulkan.ImageAspectFlags(vulkan.ImageAspectColorBit),
				MipLevel:       0,
				BaseArrayLayer: 0,
				LayerCount:     1,
			},
			ImageOffset: vulkan.Offset3D{X: 0, Y: 0, Z: 0},
			ImageExtent: vulkan.Extent3D{Width: width, Height: height, Depth: 1},
		}
		vulkan.CmdCopyBufferToImage(cb, buffer, image, vulkan.ImageLayoutTransferDstOptimal, 1, []vulkan.BufferImageCopy{region})
	})
}

func (a *VulkanApp) createRenderPass() error {
	colorAttachment := vulkan.AttachmentDescription{
		Format:         a.swapchainFormat,
		Samples:        vulkan.SampleCount1Bit,
		LoadOp:         vulkan.AttachmentLoadOpClear,
		StoreOp:        vulkan.AttachmentStoreOpStore,
		InitialLayout:  vulkan.ImageLayoutUndefined,
		FinalLayout:    vulkan.ImageLayoutPresentSrc,
		StencilLoadOp:  vulkan.AttachmentLoadOpDontCare,
		StencilStoreOp: vulkan.AttachmentStoreOpDontCare,
	}

	depthAttachment := vulkan.AttachmentDescription{
		Format:         a.depthFormat,
		Samples:        vulkan.SampleCount1Bit,
		LoadOp:         vulkan.AttachmentLoadOpClear,
		StoreOp:        vulkan.AttachmentStoreOpDontCare,
		StencilLoadOp:  vulkan.AttachmentLoadOpDontCare,
		StencilStoreOp: vulkan.AttachmentStoreOpDontCare,
		InitialLayout:  vulkan.ImageLayoutUndefined,
		FinalLayout:    vulkan.ImageLayoutDepthStencilAttachmentOptimal,
	}

	colorRef := vulkan.AttachmentReference{
		Attachment: 0,
		Layout:     vulkan.ImageLayoutColorAttachmentOptimal,
	}
	depthRef := vulkan.AttachmentReference{
		Attachment: 1,
		Layout:     vulkan.ImageLayoutDepthStencilAttachmentOptimal,
	}

	subpass := vulkan.SubpassDescription{
		PipelineBindPoint:       vulkan.PipelineBindPointGraphics,
		ColorAttachmentCount:    1,
		PColorAttachments:       []vulkan.AttachmentReference{colorRef},
		PDepthStencilAttachment: &depthRef,
	}

	dependency := vulkan.SubpassDependency{
		SrcSubpass:    vulkan.SubpassExternal,
		DstSubpass:    0,
		SrcStageMask:  vulkan.PipelineStageFlags(vulkan.PipelineStageColorAttachmentOutputBit | vulkan.PipelineStageEarlyFragmentTestsBit),
		SrcAccessMask: 0,
		DstStageMask:  vulkan.PipelineStageFlags(vulkan.PipelineStageColorAttachmentOutputBit | vulkan.PipelineStageEarlyFragmentTestsBit),
		DstAccessMask: vulkan.AccessFlags(vulkan.AccessColorAttachmentWriteBit | vulkan.AccessDepthStencilAttachmentWriteBit),
	}

	attachments := []vulkan.AttachmentDescription{colorAttachment, depthAttachment}
	createInfo := vulkan.RenderPassCreateInfo{
		SType:           vulkan.StructureTypeRenderPassCreateInfo,
		AttachmentCount: uint32(len(attachments)),
		PAttachments:    attachments,
		SubpassCount:    1,
		PSubpasses:      []vulkan.SubpassDescription{subpass},
		DependencyCount: 1,
		PDependencies:   []vulkan.SubpassDependency{dependency},
	}

	var zeroRenderPass vulkan.RenderPass
	rpOut := (*vulkan.RenderPass)(C.malloc(C.size_t(unsafe.Sizeof(zeroRenderPass))))
	if rpOut == nil {
		return fmt.Errorf("allocate render pass handle")
	}
	defer C.free(unsafe.Pointer(rpOut))

	if res := vulkan.CreateRenderPass(a.device, &createInfo, nil, rpOut); res != vulkan.Success {
		return fmt.Errorf("create render pass: %w", vulkan.Error(res))
	}
	a.renderPass = *rpOut
	return nil
}

func (a *VulkanApp) createUniformBuffers() error {
	bufferSize := vulkan.DeviceSize(unsafe.Sizeof(uniformBufferObject{}))
	count := len(a.swapchainImages)
	a.uniformBuffers = make([]vulkan.Buffer, count)
	a.uniformBuffersMemory = make([]vulkan.DeviceMemory, count)
	for i := 0; i < count; i++ {
		buf, mem, err := a.createBuffer(bufferSize, vulkan.BufferUsageFlags(vulkan.BufferUsageUniformBufferBit), vulkan.MemoryPropertyHostVisibleBit|vulkan.MemoryPropertyHostCoherentBit)
		if err != nil {
			return err
		}
		a.uniformBuffers[i] = buf
		a.uniformBuffersMemory[i] = mem
	}
	return nil
}

func (a *VulkanApp) createDescriptorPool() error {
	poolSizes := []vulkan.DescriptorPoolSize{
		{
			Type:            vulkan.DescriptorTypeUniformBuffer,
			DescriptorCount: uint32(len(a.swapchainImages)),
		},
		{
			Type:            vulkan.DescriptorTypeCombinedImageSampler,
			DescriptorCount: uint32(len(a.swapchainImages)),
		},
	}
	poolInfo := vulkan.DescriptorPoolCreateInfo{
		SType:         vulkan.StructureTypeDescriptorPoolCreateInfo,
		MaxSets:       uint32(len(a.swapchainImages)),
		PoolSizeCount: uint32(len(poolSizes)),
		PPoolSizes:    poolSizes,
	}
	var zero vulkan.DescriptorPool
	out := (*vulkan.DescriptorPool)(C.malloc(C.size_t(unsafe.Sizeof(zero))))
	if out == nil {
		return fmt.Errorf("allocate descriptor pool handle")
	}
	defer C.free(unsafe.Pointer(out))
	if res := vulkan.CreateDescriptorPool(a.device, &poolInfo, nil, out); res != vulkan.Success {
		return fmt.Errorf("create descriptor pool: %w", vulkan.Error(res))
	}
	a.descriptorPool = *out
	return nil
}

func (a *VulkanApp) createDescriptorSets() error {
	layouts := make([]vulkan.DescriptorSetLayout, len(a.swapchainImages))
	for i := range layouts {
		layouts[i] = a.descriptorSetLayout
	}
	allocInfo := vulkan.DescriptorSetAllocateInfo{
		SType:              vulkan.StructureTypeDescriptorSetAllocateInfo,
		DescriptorPool:     a.descriptorPool,
		DescriptorSetCount: uint32(len(a.swapchainImages)),
		PSetLayouts:        layouts,
	}
	count := len(a.swapchainImages)
	if count == 0 {
		return fmt.Errorf("no swapchain images for descriptor sets")
	}
	// Allocate descriptor set array in C memory to avoid Go pointer issues.
	var zeroDS vulkan.DescriptorSet
	cBuf := C.calloc(C.size_t(count), C.size_t(unsafe.Sizeof(zeroDS)))
	if cBuf == nil {
		return fmt.Errorf("allocate descriptor set buffer")
	}
	defer C.free(cBuf)
	sh := &reflect.SliceHeader{
		Data: uintptr(cBuf),
		Len:  count,
		Cap:  count,
	}
	tmp := *(*[]vulkan.DescriptorSet)(unsafe.Pointer(sh))

	if res := vulkan.AllocateDescriptorSets(a.device, &allocInfo, &tmp[0]); res != vulkan.Success {
		return fmt.Errorf("allocate descriptor sets: %w", vulkan.Error(res))
	}
	a.descriptorSets = make([]vulkan.DescriptorSet, count)
	copy(a.descriptorSets, tmp)

	for i := range a.descriptorSets {
		bufferInfo := vulkan.DescriptorBufferInfo{
			Buffer: a.uniformBuffers[i],
			Offset: 0,
			Range:  vulkan.DeviceSize(unsafe.Sizeof(uniformBufferObject{})),
		}
		imageInfo := vulkan.DescriptorImageInfo{
			ImageLayout: vulkan.ImageLayoutShaderReadOnlyOptimal,
			ImageView:   a.textureImageView,
			Sampler:     a.textureSampler,
		}
		write := vulkan.WriteDescriptorSet{
			SType:           vulkan.StructureTypeWriteDescriptorSet,
			DstSet:          a.descriptorSets[i],
			DstBinding:      0,
			DstArrayElement: 0,
			DescriptorType:  vulkan.DescriptorTypeUniformBuffer,
			DescriptorCount: 1,
			PBufferInfo:     []vulkan.DescriptorBufferInfo{bufferInfo},
		}
		writeSampler := vulkan.WriteDescriptorSet{
			SType:           vulkan.StructureTypeWriteDescriptorSet,
			DstSet:          a.descriptorSets[i],
			DstBinding:      1,
			DstArrayElement: 0,
			DescriptorType:  vulkan.DescriptorTypeCombinedImageSampler,
			DescriptorCount: 1,
			PImageInfo:      []vulkan.DescriptorImageInfo{imageInfo},
		}
		vulkan.UpdateDescriptorSets(a.device, 2, []vulkan.WriteDescriptorSet{write, writeSampler}, 0, nil)
	}
	return nil
}

func (a *VulkanApp) updateUniformBuffer(imageIndex uint32) error {
	elapsed := float32(time.Since(a.startTime).Seconds())
	angle := elapsed * mgl32.DegToRad(45)
	model := mgl32.HomogRotate3D(angle, mgl32.Vec3{0, 0, 1})
	view := mgl32.LookAtV(
		mgl32.Vec3{3, 3, 3},
		mgl32.Vec3{0, 0, 0},
		mgl32.Vec3{0, 0, 1},
	)
	proj := mgl32.Perspective(mgl32.DegToRad(45), float32(a.swapchainExtent.Width)/float32(a.swapchainExtent.Height), 0.1, 10.0)
	proj[5] *= -1 // Vulkan clip

	ubo := uniformBufferObject{
		Model: model,
		View:  view,
		Proj:  proj,
	}

	size := vulkan.DeviceSize(unsafe.Sizeof(ubo))
	var data unsafe.Pointer
	if res := vulkan.MapMemory(a.device, a.uniformBuffersMemory[imageIndex], 0, size, 0, &data); res != vulkan.Success {
		return fmt.Errorf("map uniform buffer: %w", vulkan.Error(res))
	}
	dst := (*[1 << 30]byte)(data)[:size:size]
	src := (*[1 << 30]byte)(unsafe.Pointer(&ubo))[:size:size]
	copy(dst, src)
	vulkan.UnmapMemory(a.device, a.uniformBuffersMemory[imageIndex])
	return nil
}

func (a *VulkanApp) createVertexBuffer() error {
	bufferSize := vulkan.DeviceSize(len(cubeVertices)) * vulkan.DeviceSize(unsafe.Sizeof(vertex{}))
	buf, mem, err := a.createBuffer(bufferSize, vulkan.BufferUsageFlags(vulkan.BufferUsageVertexBufferBit), vulkan.MemoryPropertyHostVisibleBit|vulkan.MemoryPropertyHostCoherentBit)
	if err != nil {
		return err
	}
	a.vertexBuffer = buf
	a.vertexBufferMemory = mem

	var data unsafe.Pointer
	if res := vulkan.MapMemory(a.device, mem, 0, bufferSize, 0, &data); res != vulkan.Success {
		return fmt.Errorf("map vertex buffer: %w", vulkan.Error(res))
	}
	dst := (*[1 << 30]byte)(data)[:bufferSize:bufferSize]
	copy(dst, verticesToBytes(cubeVertices))
	vulkan.UnmapMemory(a.device, mem)
	return nil
}

func (a *VulkanApp) createIndexBuffer() error {
	bufferSize := vulkan.DeviceSize(len(cubeIndices) * 4)
	buf, mem, err := a.createBuffer(bufferSize, vulkan.BufferUsageFlags(vulkan.BufferUsageIndexBufferBit), vulkan.MemoryPropertyHostVisibleBit|vulkan.MemoryPropertyHostCoherentBit)
	if err != nil {
		return err
	}
	a.indexBuffer = buf
	a.indexBufferMemory = mem

	var data unsafe.Pointer
	if res := vulkan.MapMemory(a.device, mem, 0, bufferSize, 0, &data); res != vulkan.Success {
		return fmt.Errorf("map index buffer: %w", vulkan.Error(res))
	}
	dst := (*[1 << 30]byte)(data)[:bufferSize:bufferSize]
	copy(dst, indicesToBytes(cubeIndices))
	vulkan.UnmapMemory(a.device, mem)
	return nil
}

func verticesToBytes(verts []vertex) []byte {
	size := len(verts) * int(unsafe.Sizeof(vertex{}))
	out := make([]byte, size)
	hdr := (*[1 << 30]byte)(unsafe.Pointer(&verts[0]))[:size:size]
	copy(out, hdr)
	return out
}

func indicesToBytes(idxs []uint32) []byte {
	size := len(idxs) * 4
	out := make([]byte, size)
	hdr := (*[1 << 30]byte)(unsafe.Pointer(&idxs[0]))[:size:size]
	copy(out, hdr)
	return out
}

func (a *VulkanApp) createDescriptorSetLayout() error {
	uLayoutBinding := vulkan.DescriptorSetLayoutBinding{
		Binding:         0,
		DescriptorType:  vulkan.DescriptorTypeUniformBuffer,
		DescriptorCount: 1,
		StageFlags:      vulkan.ShaderStageFlags(vulkan.ShaderStageVertexBit),
	}
	samplerBinding := vulkan.DescriptorSetLayoutBinding{
		Binding:         1,
		DescriptorType:  vulkan.DescriptorTypeCombinedImageSampler,
		DescriptorCount: 1,
		StageFlags:      vulkan.ShaderStageFlags(vulkan.ShaderStageFragmentBit),
	}
	layoutInfo := vulkan.DescriptorSetLayoutCreateInfo{
		SType:        vulkan.StructureTypeDescriptorSetLayoutCreateInfo,
		BindingCount: 2,
		PBindings:    []vulkan.DescriptorSetLayoutBinding{uLayoutBinding, samplerBinding},
	}
	var zero vulkan.DescriptorSetLayout
	out := (*vulkan.DescriptorSetLayout)(C.malloc(C.size_t(unsafe.Sizeof(zero))))
	if out == nil {
		return fmt.Errorf("allocate descriptor set layout handle")
	}
	defer C.free(unsafe.Pointer(out))
	if res := vulkan.CreateDescriptorSetLayout(a.device, &layoutInfo, nil, out); res != vulkan.Success {
		return fmt.Errorf("create descriptor set layout: %w", vulkan.Error(res))
	}
	a.descriptorSetLayout = *out
	return nil
}

func (a *VulkanApp) createGraphicsPipeline() error {
	vertCode, err := os.ReadFile("shaders/vert.spv")
	if err != nil {
		return fmt.Errorf("read vertex shader: %w", err)
	}
	fragCode, err := os.ReadFile("shaders/frag.spv")
	if err != nil {
		return fmt.Errorf("read fragment shader: %w", err)
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
		Stride:    uint32(unsafe.Sizeof(vertex{})),
		InputRate: vulkan.VertexInputRateVertex,
	}
	attributeDescriptions := []vulkan.VertexInputAttributeDescription{
		{Location: 0, Binding: 0, Format: vulkan.FormatR32g32b32Sfloat, Offset: uint32(unsafe.Offsetof(vertex{}.pos))},
		{Location: 1, Binding: 0, Format: vulkan.FormatR32g32b32Sfloat, Offset: uint32(unsafe.Offsetof(vertex{}.color))},
		{Location: 2, Binding: 0, Format: vulkan.FormatR32g32Sfloat, Offset: uint32(unsafe.Offsetof(vertex{}.uv))},
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
		DepthClampEnable:        vulkan.False,
		RasterizerDiscardEnable: vulkan.False,
		PolygonMode:             vulkan.PolygonModeFill,
		LineWidth:               1.0,
		CullMode:                vulkan.CullModeFlags(vulkan.CullModeBackBit),
		FrontFace:               vulkan.FrontFaceCounterClockwise,
		DepthBiasEnable:         vulkan.False,
	}

	multisampling := vulkan.PipelineMultisampleStateCreateInfo{
		SType:                vulkan.StructureTypePipelineMultisampleStateCreateInfo,
		RasterizationSamples: vulkan.SampleCount1Bit,
	}

	depthStencil := vulkan.PipelineDepthStencilStateCreateInfo{
		SType:                 vulkan.StructureTypePipelineDepthStencilStateCreateInfo,
		DepthTestEnable:       vulkan.True,
		DepthWriteEnable:      vulkan.True,
		DepthCompareOp:        vulkan.CompareOpLess,
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

	pipelineLayoutInfo := vulkan.PipelineLayoutCreateInfo{
		SType:                  vulkan.StructureTypePipelineLayoutCreateInfo,
		SetLayoutCount:         1,
		PSetLayouts:            []vulkan.DescriptorSetLayout{a.descriptorSetLayout},
		PushConstantRangeCount: 0,
	}
	var zeroLayout vulkan.PipelineLayout
	layoutOut := (*vulkan.PipelineLayout)(C.malloc(C.size_t(unsafe.Sizeof(zeroLayout))))
	if layoutOut == nil {
		return fmt.Errorf("allocate pipeline layout handle")
	}
	defer C.free(unsafe.Pointer(layoutOut))
	if res := vulkan.CreatePipelineLayout(a.device, &pipelineLayoutInfo, nil, layoutOut); res != vulkan.Success {
		return fmt.Errorf("create pipeline layout: %w", vulkan.Error(res))
	}
	a.pipelineLayout = *layoutOut

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
		Layout:              a.pipelineLayout,
		RenderPass:          a.renderPass,
		Subpass:             0,
	}

	// Allocate pipeline array in C memory to avoid Go pointer issues.
	var zeroPipeline vulkan.Pipeline
	cBuf := C.calloc(C.size_t(1), C.size_t(unsafe.Sizeof(zeroPipeline)))
	if cBuf == nil {
		vulkan.DestroyPipelineLayout(a.device, a.pipelineLayout, nil)
		return fmt.Errorf("allocate pipeline buffer")
	}
	defer C.free(cBuf)
	sh := &reflect.SliceHeader{
		Data: uintptr(cBuf),
		Len:  1,
		Cap:  1,
	}
	pipelines := *(*[]vulkan.Pipeline)(unsafe.Pointer(sh))

	if res := vulkan.CreateGraphicsPipelines(a.device, vulkan.PipelineCache(vulkan.NullHandle), 1, []vulkan.GraphicsPipelineCreateInfo{pipelineInfo}, nil, pipelines); res != vulkan.Success {
		vulkan.DestroyPipelineLayout(a.device, a.pipelineLayout, nil)
		return fmt.Errorf("create graphics pipeline: %w", vulkan.Error(res))
	}
	a.pipeline = pipelines[0]
	return nil
}

func (a *VulkanApp) createShaderModule(code []byte) (vulkan.ShaderModule, error) {
	codeAligned := bytesToUint32(code)
	createInfo := vulkan.ShaderModuleCreateInfo{
		SType:    vulkan.StructureTypeShaderModuleCreateInfo,
		CodeSize: uint(len(code)),
		PCode:    codeAligned,
	}
	var module vulkan.ShaderModule
	if res := vulkan.CreateShaderModule(a.device, &createInfo, nil, &module); res != vulkan.Success {
		return vulkan.ShaderModule(vulkan.NullHandle), fmt.Errorf("create shader module: %w", vulkan.Error(res))
	}
	return module, nil
}

func bytesToUint32(data []byte) []uint32 {
	if len(data)%4 != 0 {
		panic("shader code length must be multiple of 4")
	}
	hdr := (*[1 << 30]uint32)(unsafe.Pointer(&data[0]))[: len(data)/4 : len(data)/4]
	return hdr
}

func (a *VulkanApp) createBuffer(size vulkan.DeviceSize, usage vulkan.BufferUsageFlags, properties vulkan.MemoryPropertyFlagBits) (vulkan.Buffer, vulkan.DeviceMemory, error) {
	bufferInfo := vulkan.BufferCreateInfo{
		SType:       vulkan.StructureTypeBufferCreateInfo,
		Size:        size,
		Usage:       usage,
		SharingMode: vulkan.SharingModeExclusive,
	}
	var zeroBuffer vulkan.Buffer
	bufferOut := (*vulkan.Buffer)(C.malloc(C.size_t(unsafe.Sizeof(zeroBuffer))))
	if bufferOut == nil {
		return vulkan.Buffer(vulkan.NullHandle), vulkan.DeviceMemory(vulkan.NullHandle), fmt.Errorf("allocate buffer handle")
	}
	defer C.free(unsafe.Pointer(bufferOut))

	if res := vulkan.CreateBuffer(a.device, &bufferInfo, nil, bufferOut); res != vulkan.Success {
		return vulkan.Buffer(vulkan.NullHandle), vulkan.DeviceMemory(vulkan.NullHandle), fmt.Errorf("create buffer: %w", vulkan.Error(res))
	}
	var memReq vulkan.MemoryRequirements
	vulkan.GetBufferMemoryRequirements(a.device, *bufferOut, &memReq)
	memReq.Deref()

	memoryType, ok := a.findMemoryType(memReq.MemoryTypeBits, properties)
	if !ok {
		vulkan.DestroyBuffer(a.device, *bufferOut, nil)
		return vulkan.Buffer(vulkan.NullHandle), vulkan.DeviceMemory(vulkan.NullHandle), fmt.Errorf("no suitable memory type for buffer")
	}

	allocInfo := vulkan.MemoryAllocateInfo{
		SType:           vulkan.StructureTypeMemoryAllocateInfo,
		AllocationSize:  memReq.Size,
		MemoryTypeIndex: memoryType,
	}
	var zeroMem vulkan.DeviceMemory
	bufferMemoryOut := (*vulkan.DeviceMemory)(C.malloc(C.size_t(unsafe.Sizeof(zeroMem))))
	if bufferMemoryOut == nil {
		vulkan.DestroyBuffer(a.device, *bufferOut, nil)
		return vulkan.Buffer(vulkan.NullHandle), vulkan.DeviceMemory(vulkan.NullHandle), fmt.Errorf("allocate buffer memory handle")
	}
	defer C.free(unsafe.Pointer(bufferMemoryOut))

	if res := vulkan.AllocateMemory(a.device, &allocInfo, nil, bufferMemoryOut); res != vulkan.Success {
		vulkan.DestroyBuffer(a.device, *bufferOut, nil)
		return vulkan.Buffer(vulkan.NullHandle), vulkan.DeviceMemory(vulkan.NullHandle), fmt.Errorf("allocate buffer memory: %w", vulkan.Error(res))
	}
	vulkan.BindBufferMemory(a.device, *bufferOut, *bufferMemoryOut, 0)
	return *bufferOut, *bufferMemoryOut, nil
}

func (a *VulkanApp) createFramebuffers() error {
	a.framebuffers = make([]vulkan.Framebuffer, len(a.swapchainViews))
	for i := range a.swapchainViews {
		attachments := []vulkan.ImageView{a.swapchainViews[i], a.depthImageView}
		createInfo := vulkan.FramebufferCreateInfo{
			SType:           vulkan.StructureTypeFramebufferCreateInfo,
			RenderPass:      a.renderPass,
			AttachmentCount: uint32(len(attachments)),
			PAttachments:    attachments,
			Width:           a.swapchainExtent.Width,
			Height:          a.swapchainExtent.Height,
			Layers:          1,
		}
		var zero vulkan.Framebuffer
		fbOut := (*vulkan.Framebuffer)(C.malloc(C.size_t(unsafe.Sizeof(zero))))
		if fbOut == nil {
			return fmt.Errorf("allocate framebuffer handle")
		}
		res := vulkan.CreateFramebuffer(a.device, &createInfo, nil, fbOut)
		if res != vulkan.Success {
			C.free(unsafe.Pointer(fbOut))
			return fmt.Errorf("create framebuffer %d: %w", i, vulkan.Error(res))
		}
		a.framebuffers[i] = *fbOut
		C.free(unsafe.Pointer(fbOut))
	}
	return nil
}

func (a *VulkanApp) createCommandPool() error {
	poolInfo := vulkan.CommandPoolCreateInfo{
		SType:            vulkan.StructureTypeCommandPoolCreateInfo,
		QueueFamilyIndex: a.queues.graphicsFamily,
		Flags:            vulkan.CommandPoolCreateFlags(vulkan.CommandPoolCreateResetCommandBufferBit),
	}
	var zero vulkan.CommandPool
	out := (*vulkan.CommandPool)(C.malloc(C.size_t(unsafe.Sizeof(zero))))
	if out == nil {
		return fmt.Errorf("allocate command pool handle")
	}
	defer C.free(unsafe.Pointer(out))
	if res := vulkan.CreateCommandPool(a.device, &poolInfo, nil, out); res != vulkan.Success {
		return fmt.Errorf("create command pool: %w", vulkan.Error(res))
	}
	a.commandPool = *out
	return nil
}

func (a *VulkanApp) allocateCommandBuffers() error {
	allocInfo := vulkan.CommandBufferAllocateInfo{
		SType:              vulkan.StructureTypeCommandBufferAllocateInfo,
		CommandPool:        a.commandPool,
		Level:              vulkan.CommandBufferLevelPrimary,
		CommandBufferCount: uint32(len(a.framebuffers)),
	}
	count := len(a.framebuffers)
	if count == 0 {
		return fmt.Errorf("no framebuffers for command buffers")
	}
	var zeroCB vulkan.CommandBuffer
	cBuf := C.calloc(C.size_t(count), C.size_t(unsafe.Sizeof(zeroCB)))
	if cBuf == nil {
		return fmt.Errorf("allocate command buffer array")
	}
	defer C.free(cBuf)
	sh := &reflect.SliceHeader{
		Data: uintptr(cBuf),
		Len:  count,
		Cap:  count,
	}
	tmp := *(*[]vulkan.CommandBuffer)(unsafe.Pointer(sh))

	if res := vulkan.AllocateCommandBuffers(a.device, &allocInfo, tmp); res != vulkan.Success {
		return fmt.Errorf("allocate command buffers: %w", vulkan.Error(res))
	}
	a.commandBuffers = make([]vulkan.CommandBuffer, count)
	copy(a.commandBuffers, tmp)
	return nil
}

func (a *VulkanApp) createSyncObjects() error {
	a.imageAvailable = make([]vulkan.Semaphore, maxFramesInFlight)
	a.renderFinished = make([]vulkan.Semaphore, maxFramesInFlight)
	a.inFlightFences = make([]vulkan.Fence, maxFramesInFlight)
	a.imagesInFlight = make([]vulkan.Fence, len(a.swapchainImages))

	semInfo := vulkan.SemaphoreCreateInfo{
		SType: vulkan.StructureTypeSemaphoreCreateInfo,
	}
	fenceInfo := vulkan.FenceCreateInfo{
		SType: vulkan.StructureTypeFenceCreateInfo,
		Flags: vulkan.FenceCreateFlags(vulkan.FenceCreateSignaledBit),
	}

	for i := 0; i < maxFramesInFlight; i++ {
		var zeroSem vulkan.Semaphore
		semOut := (*vulkan.Semaphore)(C.malloc(C.size_t(unsafe.Sizeof(zeroSem))))
		if semOut == nil {
			return fmt.Errorf("allocate semaphore handle")
		}
		res := vulkan.CreateSemaphore(a.device, &semInfo, nil, semOut)
		if res != vulkan.Success {
			C.free(unsafe.Pointer(semOut))
			return fmt.Errorf("create imageAvailable semaphore %d: %w", i, vulkan.Error(res))
		}
		a.imageAvailable[i] = *semOut
		C.free(unsafe.Pointer(semOut))

		semOut2 := (*vulkan.Semaphore)(C.malloc(C.size_t(unsafe.Sizeof(zeroSem))))
		if semOut2 == nil {
			return fmt.Errorf("allocate semaphore handle")
		}
		res = vulkan.CreateSemaphore(a.device, &semInfo, nil, semOut2)
		if res != vulkan.Success {
			C.free(unsafe.Pointer(semOut2))
			return fmt.Errorf("create renderFinished semaphore %d: %w", i, vulkan.Error(res))
		}
		a.renderFinished[i] = *semOut2
		C.free(unsafe.Pointer(semOut2))

		var zeroFence vulkan.Fence
		fenceOut := (*vulkan.Fence)(C.malloc(C.size_t(unsafe.Sizeof(zeroFence))))
		if fenceOut == nil {
			return fmt.Errorf("allocate fence handle")
		}
		res = vulkan.CreateFence(a.device, &fenceInfo, nil, fenceOut)
		if res != vulkan.Success {
			C.free(unsafe.Pointer(fenceOut))
			return fmt.Errorf("create fence %d: %w", i, vulkan.Error(res))
		}
		a.inFlightFences[i] = *fenceOut
		C.free(unsafe.Pointer(fenceOut))
	}
	return nil
}

func (a *VulkanApp) requestSwapchainRecreate() {
	a.framebufferResized = true
}

func (a *VulkanApp) oneTimeCommands(fn func(vulkan.CommandBuffer)) error {
	allocInfo := vulkan.CommandBufferAllocateInfo{
		SType:              vulkan.StructureTypeCommandBufferAllocateInfo,
		CommandPool:        a.commandPool,
		Level:              vulkan.CommandBufferLevelPrimary,
		CommandBufferCount: 1,
	}
	buffers := make([]vulkan.CommandBuffer, 1)
	if res := vulkan.AllocateCommandBuffers(a.device, &allocInfo, buffers); res != vulkan.Success {
		return fmt.Errorf("allocate one-time command buffer: %w", vulkan.Error(res))
	}
	cb := buffers[0]
	beginInfo := vulkan.CommandBufferBeginInfo{
		SType:            vulkan.StructureTypeCommandBufferBeginInfo,
		Flags:            vulkan.CommandBufferUsageFlags(vulkan.CommandBufferUsageOneTimeSubmitBit),
		PInheritanceInfo: nil,
	}
	if res := vulkan.BeginCommandBuffer(cb, &beginInfo); res != vulkan.Success {
		return fmt.Errorf("begin one-time command buffer: %w", vulkan.Error(res))
	}
	fn(cb)
	if res := vulkan.EndCommandBuffer(cb); res != vulkan.Success {
		return fmt.Errorf("end one-time command buffer: %w", vulkan.Error(res))
	}
	submitInfo := vulkan.SubmitInfo{
		SType:              vulkan.StructureTypeSubmitInfo,
		CommandBufferCount: 1,
		PCommandBuffers:    []vulkan.CommandBuffer{cb},
	}
	if res := vulkan.QueueSubmit(a.graphicsQueue, 1, []vulkan.SubmitInfo{submitInfo}, vulkan.Fence(vulkan.NullHandle)); res != vulkan.Success {
		vulkan.FreeCommandBuffers(a.device, a.commandPool, 1, []vulkan.CommandBuffer{cb})
		return fmt.Errorf("submit one-time command buffer: %w", vulkan.Error(res))
	}
	vulkan.QueueWaitIdle(a.graphicsQueue)
	vulkan.FreeCommandBuffers(a.device, a.commandPool, 1, []vulkan.CommandBuffer{cb})
	return nil
}

func (a *VulkanApp) debugClearSwapchainImage(imageIndex uint32) error {
	return a.oneTimeCommands(func(cb vulkan.CommandBuffer) {
		subresource := vulkan.ImageSubresourceRange{
			AspectMask:     vulkan.ImageAspectFlags(vulkan.ImageAspectColorBit),
			BaseMipLevel:   0,
			LevelCount:     1,
			BaseArrayLayer: 0,
			LayerCount:     1,
		}
		barrierToTransfer := vulkan.ImageMemoryBarrier{
			SType:               vulkan.StructureTypeImageMemoryBarrier,
			OldLayout:           vulkan.ImageLayoutPresentSrc,
			NewLayout:           vulkan.ImageLayoutTransferDstOptimal,
			SrcQueueFamilyIndex: vulkan.QueueFamilyIgnored,
			DstQueueFamilyIndex: vulkan.QueueFamilyIgnored,
			Image:               a.swapchainImages[imageIndex],
			SubresourceRange:    subresource,
			SrcAccessMask:       0,
			DstAccessMask:       vulkan.AccessFlags(vulkan.AccessTransferWriteBit),
		}
		vulkan.CmdPipelineBarrier(
			cb,
			vulkan.PipelineStageFlags(vulkan.PipelineStageBottomOfPipeBit),
			vulkan.PipelineStageFlags(vulkan.PipelineStageTransferBit),
			0,
			0, nil,
			0, nil,
			1, []vulkan.ImageMemoryBarrier{barrierToTransfer},
		)

		color := vulkan.ClearColorValue{}
		floats := (*[4]float32)(unsafe.Pointer(&color))
		floats[0] = 1.0
		floats[1] = 0.0
		floats[2] = 1.0
		floats[3] = 1.0
		vulkan.CmdClearColorImage(
			cb,
			a.swapchainImages[imageIndex],
			vulkan.ImageLayoutTransferDstOptimal,
			&color,
			1,
			[]vulkan.ImageSubresourceRange{subresource},
		)

		barrierToColor := vulkan.ImageMemoryBarrier{
			SType:               vulkan.StructureTypeImageMemoryBarrier,
			OldLayout:           vulkan.ImageLayoutTransferDstOptimal,
			NewLayout:           vulkan.ImageLayoutColorAttachmentOptimal,
			SrcQueueFamilyIndex: vulkan.QueueFamilyIgnored,
			DstQueueFamilyIndex: vulkan.QueueFamilyIgnored,
			Image:               a.swapchainImages[imageIndex],
			SubresourceRange:    subresource,
			SrcAccessMask:       vulkan.AccessFlags(vulkan.AccessTransferWriteBit),
			DstAccessMask:       vulkan.AccessFlags(vulkan.AccessColorAttachmentWriteBit),
		}
		vulkan.CmdPipelineBarrier(
			cb,
			vulkan.PipelineStageFlags(vulkan.PipelineStageTransferBit),
			vulkan.PipelineStageFlags(vulkan.PipelineStageColorAttachmentOutputBit),
			0,
			0, nil,
			0, nil,
			1, []vulkan.ImageMemoryBarrier{barrierToColor},
		)
	})
}

func (a *VulkanApp) cleanupSwapchain() {
	if len(a.commandBuffers) > 0 {
		vulkan.FreeCommandBuffers(a.device, a.commandPool, uint32(len(a.commandBuffers)), a.commandBuffers)
		a.commandBuffers = nil
	}
	for _, fb := range a.framebuffers {
		vulkan.DestroyFramebuffer(a.device, fb, nil)
	}
	a.framebuffers = nil
	if a.renderPass != vulkan.RenderPass(vulkan.NullHandle) {
		vulkan.DestroyRenderPass(a.device, a.renderPass, nil)
		a.renderPass = vulkan.RenderPass(vulkan.NullHandle)
	}
	if a.pipeline != vulkan.Pipeline(vulkan.NullHandle) {
		vulkan.DestroyPipeline(a.device, a.pipeline, nil)
		a.pipeline = vulkan.Pipeline(vulkan.NullHandle)
	}
	if a.pipelineLayout != vulkan.PipelineLayout(vulkan.NullHandle) {
		vulkan.DestroyPipelineLayout(a.device, a.pipelineLayout, nil)
		a.pipelineLayout = vulkan.PipelineLayout(vulkan.NullHandle)
	}
	for _, view := range a.swapchainViews {
		vulkan.DestroyImageView(a.device, view, nil)
	}
	a.swapchainViews = nil
	if a.depthImageView != vulkan.ImageView(vulkan.NullHandle) {
		vulkan.DestroyImageView(a.device, a.depthImageView, nil)
		a.depthImageView = vulkan.ImageView(vulkan.NullHandle)
	}
	if a.depthImage != vulkan.Image(vulkan.NullHandle) {
		vulkan.DestroyImage(a.device, a.depthImage, nil)
		a.depthImage = vulkan.Image(vulkan.NullHandle)
	}
	if a.depthImageMemory != vulkan.DeviceMemory(vulkan.NullHandle) {
		vulkan.FreeMemory(a.device, a.depthImageMemory, nil)
		a.depthImageMemory = vulkan.DeviceMemory(vulkan.NullHandle)
	}
	for i := range a.uniformBuffers {
		if a.uniformBuffers[i] != vulkan.Buffer(vulkan.NullHandle) {
			vulkan.DestroyBuffer(a.device, a.uniformBuffers[i], nil)
		}
		if a.uniformBuffersMemory[i] != vulkan.DeviceMemory(vulkan.NullHandle) {
			vulkan.FreeMemory(a.device, a.uniformBuffersMemory[i], nil)
		}
	}
	a.uniformBuffers = nil
	a.uniformBuffersMemory = nil
	if a.descriptorPool != vulkan.DescriptorPool(vulkan.NullHandle) {
		vulkan.DestroyDescriptorPool(a.device, a.descriptorPool, nil)
		a.descriptorPool = vulkan.DescriptorPool(vulkan.NullHandle)
	}
	if a.swapchain != vulkan.Swapchain(vulkan.NullHandle) {
		vulkan.DestroySwapchain(a.device, a.swapchain, nil)
		a.swapchain = vulkan.Swapchain(vulkan.NullHandle)
	}
}

func (a *VulkanApp) recreateSwapchain() error {
	vulkan.DeviceWaitIdle(a.device)
	a.cleanupSwapchain()

	if err := a.createSwapchain(); err != nil {
		return err
	}
	if err := a.createImageViews(); err != nil {
		return err
	}
	if err := a.createDepthResources(); err != nil {
		return err
	}
	if err := a.createRenderPass(); err != nil {
		return err
	}
	if err := a.createGraphicsPipeline(); err != nil {
		return err
	}
	if err := a.createFramebuffers(); err != nil {
		return err
	}
	if err := a.createUniformBuffers(); err != nil {
		return err
	}
	if err := a.createDescriptorPool(); err != nil {
		return err
	}
	if err := a.createDescriptorSets(); err != nil {
		return err
	}
	if err := a.allocateCommandBuffers(); err != nil {
		return err
	}
	a.imagesInFlight = make([]vulkan.Fence, len(a.swapchainImages))
	a.framebufferResized = false
	return nil
}

func (a *VulkanApp) recordCommandBuffer(cb vulkan.CommandBuffer, imageIndex int) error {
	beginInfo := vulkan.CommandBufferBeginInfo{
		SType: vulkan.StructureTypeCommandBufferBeginInfo,
	}
	if res := vulkan.BeginCommandBuffer(cb, &beginInfo); res != vulkan.Success {
		return fmt.Errorf("begin command buffer: %w", vulkan.Error(res))
	}

	clearColor := vulkan.NewClearValue([]float32{0.05, 0.05, 0.1, 1.0})
	clearDepth := vulkan.NewClearDepthStencil(1.0, 0)
	clearValues := []vulkan.ClearValue{clearColor, clearDepth}

	renderPassInfo := vulkan.RenderPassBeginInfo{
		SType:       vulkan.StructureTypeRenderPassBeginInfo,
		RenderPass:  a.renderPass,
		Framebuffer: a.framebuffers[imageIndex],
		RenderArea: vulkan.Rect2D{
			Offset: vulkan.Offset2D{X: 0, Y: 0},
			Extent: a.swapchainExtent,
		},
		ClearValueCount: uint32(len(clearValues)),
		PClearValues:    clearValues,
	}

	vulkan.CmdBeginRenderPass(cb, &renderPassInfo, vulkan.SubpassContentsInline)

	vulkan.CmdBindPipeline(cb, vulkan.PipelineBindPointGraphics, a.pipeline)
	vertexBuffers := []vulkan.Buffer{a.vertexBuffer}
	offsets := []vulkan.DeviceSize{0}
	vulkan.CmdBindVertexBuffers(cb, 0, 1, vertexBuffers, offsets)
	vulkan.CmdBindIndexBuffer(cb, a.indexBuffer, 0, vulkan.IndexTypeUint32)
	vulkan.CmdBindDescriptorSets(cb, vulkan.PipelineBindPointGraphics, a.pipelineLayout, 0, 1, []vulkan.DescriptorSet{a.descriptorSets[imageIndex]}, 0, nil)
	vulkan.CmdDrawIndexed(cb, uint32(len(cubeIndices)), 1, 0, 0, 0)

	vulkan.CmdEndRenderPass(cb)

	if res := vulkan.EndCommandBuffer(cb); res != vulkan.Success {
		return fmt.Errorf("end command buffer: %w", vulkan.Error(res))
	}
	return nil
}

func (a *VulkanApp) DrawFrame() error {
	frame := a.currentFrame % maxFramesInFlight
	if a.debugFrames == 0 {
		log.Printf("DrawFrame start (frame %d)", frame)
	}
	vulkan.WaitForFences(a.device, 1, []vulkan.Fence{a.inFlightFences[frame]}, vulkan.True, vulkan.MaxUint64)

	if a.framebufferResized {
		if err := a.recreateSwapchain(); err != nil {
			return err
		}
	}

	var imageIndex uint32
	res := vulkan.AcquireNextImage(a.device, a.swapchain, vulkan.MaxUint64, a.imageAvailable[frame], vulkan.Fence(vulkan.NullHandle), &imageIndex)
	if res == vulkan.ErrorOutOfDate {
		return a.recreateSwapchain()
	}
	if res != vulkan.Success && res != vulkan.Suboptimal {
		return fmt.Errorf("acquire next image: %w", vulkan.Error(res))
	}

	if err := a.updateUniformBuffer(imageIndex); err != nil {
		return err
	}

	if a.imagesInFlight[imageIndex] != vulkan.Fence(vulkan.NullHandle) {
		vulkan.WaitForFences(a.device, 1, []vulkan.Fence{a.imagesInFlight[imageIndex]}, vulkan.True, vulkan.MaxUint64)
	}
	a.imagesInFlight[imageIndex] = a.inFlightFences[frame]

	vulkan.ResetFences(a.device, 1, []vulkan.Fence{a.inFlightFences[frame]})

	vulkan.ResetCommandBuffer(a.commandBuffers[imageIndex], 0)
	if err := a.recordCommandBuffer(a.commandBuffers[imageIndex], int(imageIndex)); err != nil {
		return err
	}

	waitStages := []vulkan.PipelineStageFlags{vulkan.PipelineStageFlags(vulkan.PipelineStageColorAttachmentOutputBit)}
	submitInfo := vulkan.SubmitInfo{
		SType:                vulkan.StructureTypeSubmitInfo,
		WaitSemaphoreCount:   1,
		PWaitSemaphores:      []vulkan.Semaphore{a.imageAvailable[frame]},
		PWaitDstStageMask:    waitStages,
		CommandBufferCount:   1,
		PCommandBuffers:      []vulkan.CommandBuffer{a.commandBuffers[imageIndex]},
		SignalSemaphoreCount: 1,
		PSignalSemaphores:    []vulkan.Semaphore{a.renderFinished[frame]},
	}
	if res := vulkan.QueueSubmit(a.graphicsQueue, 1, []vulkan.SubmitInfo{submitInfo}, a.inFlightFences[frame]); res != vulkan.Success {
		return fmt.Errorf("queue submit: %w", vulkan.Error(res))
	}

	presentInfo := vulkan.PresentInfo{
		SType:              vulkan.StructureTypePresentInfo,
		WaitSemaphoreCount: 1,
		PWaitSemaphores:    []vulkan.Semaphore{a.renderFinished[frame]},
		SwapchainCount:     1,
		PSwapchains:        []vulkan.Swapchain{a.swapchain},
		PImageIndices:      []uint32{imageIndex},
	}

	res = vulkan.QueuePresent(a.presentQueue, &presentInfo)
	if res == vulkan.ErrorOutOfDate || res == vulkan.Suboptimal || a.framebufferResized {
		return a.recreateSwapchain()
	}
	if res != vulkan.Success {
		return fmt.Errorf("queue present: %w", vulkan.Error(res))
	}

	if a.debugFrames < 5 {
		log.Printf("frame %d presented (image %d, res=%v)", a.debugFrames, imageIndex, res)
	}
	a.debugFrames++

	a.currentFrame = (a.currentFrame + 1) % maxFramesInFlight
	return nil
}

func (a *VulkanApp) Cleanup() {
	vulkan.DeviceWaitIdle(a.device)

	a.cleanupSwapchain()

	for i := 0; i < maxFramesInFlight; i++ {
		vulkan.DestroySemaphore(a.device, a.renderFinished[i], nil)
		vulkan.DestroySemaphore(a.device, a.imageAvailable[i], nil)
		vulkan.DestroyFence(a.device, a.inFlightFences[i], nil)
	}

	if a.commandPool != vulkan.CommandPool(vulkan.NullHandle) {
		vulkan.DestroyCommandPool(a.device, a.commandPool, nil)
	}
	if a.textureSampler != vulkan.Sampler(vulkan.NullHandle) {
		vulkan.DestroySampler(a.device, a.textureSampler, nil)
	}
	if a.textureImageView != vulkan.ImageView(vulkan.NullHandle) {
		vulkan.DestroyImageView(a.device, a.textureImageView, nil)
	}
	if a.textureImage != vulkan.Image(vulkan.NullHandle) {
		vulkan.DestroyImage(a.device, a.textureImage, nil)
	}
	if a.textureImageMemory != vulkan.DeviceMemory(vulkan.NullHandle) {
		vulkan.FreeMemory(a.device, a.textureImageMemory, nil)
	}
	if a.vertexBuffer != vulkan.Buffer(vulkan.NullHandle) {
		vulkan.DestroyBuffer(a.device, a.vertexBuffer, nil)
	}
	if a.vertexBufferMemory != vulkan.DeviceMemory(vulkan.NullHandle) {
		vulkan.FreeMemory(a.device, a.vertexBufferMemory, nil)
	}
	if a.indexBuffer != vulkan.Buffer(vulkan.NullHandle) {
		vulkan.DestroyBuffer(a.device, a.indexBuffer, nil)
	}
	if a.indexBufferMemory != vulkan.DeviceMemory(vulkan.NullHandle) {
		vulkan.FreeMemory(a.device, a.indexBufferMemory, nil)
	}
	if a.descriptorSetLayout != vulkan.DescriptorSetLayout(vulkan.NullHandle) {
		vulkan.DestroyDescriptorSetLayout(a.device, a.descriptorSetLayout, nil)
	}
	if a.device != vulkan.Device(vulkan.NullHandle) {
		vulkan.DestroyDevice(a.device, nil)
	}
	if a.debugCallback != vulkan.DebugReportCallback(vulkan.NullHandle) {
		vulkan.DestroyDebugReportCallback(a.instance, a.debugCallback, nil)
	}
	if a.surface != vulkan.Surface(vulkan.NullHandle) {
		vulkan.DestroySurface(a.instance, a.surface, nil)
	}
	if a.instance != vulkan.Instance(vulkan.NullHandle) {
		vulkan.DestroyInstance(a.instance, nil)
	}
}

func clamp(val, min, max uint64) uint64 {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}
