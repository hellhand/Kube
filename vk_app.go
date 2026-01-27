package main

import (
	"errors"
	"fmt"
	"log"
	"math"
	"os"
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
}

var cubeVertices = []vertex{
	{pos: mgl32.Vec3{-1, -1, -1}, color: mgl32.Vec3{1, 0, 0}},
	{pos: mgl32.Vec3{1, -1, -1}, color: mgl32.Vec3{0, 1, 0}},
	{pos: mgl32.Vec3{1, 1, -1}, color: mgl32.Vec3{0, 0, 1}},
	{pos: mgl32.Vec3{-1, 1, -1}, color: mgl32.Vec3{1, 1, 0}},
	{pos: mgl32.Vec3{-1, -1, 1}, color: mgl32.Vec3{1, 0, 1}},
	{pos: mgl32.Vec3{1, -1, 1}, color: mgl32.Vec3{0, 1, 1}},
	{pos: mgl32.Vec3{1, 1, 1}, color: mgl32.Vec3{1, 1, 1}},
	{pos: mgl32.Vec3{-1, 1, 1}, color: mgl32.Vec3{0.2, 0.6, 1}},
}

var cubeIndices = []uint32{
	0, 1, 2, 2, 3, 0, // back
	4, 5, 6, 6, 7, 4, // front
	4, 5, 1, 1, 0, 4, // bottom
	7, 6, 2, 2, 3, 7, // top
	4, 0, 3, 3, 7, 4, // left
	5, 1, 2, 2, 6, 5, // right
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
	if err := vulkan.InitInstance(a.instance); err != nil {
		return fmt.Errorf("vkInitInstance: %w", err)
	}
	if err := a.setupDebugCallback(); err != nil {
		return err
	}
	if err := a.createSurface(); err != nil {
		return err
	}
	if err := a.pickPhysicalDevice(); err != nil {
		return err
	}
	if err := a.createLogicalDevice(); err != nil {
		return err
	}
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
	if err := a.createDescriptorSetLayout(); err != nil {
		return err
	}
	if err := a.createGraphicsPipeline(); err != nil {
		return err
	}
	if err := a.createFramebuffers(); err != nil {
		return err
	}
	if err := a.createCommandPool(); err != nil {
		return err
	}
	if err := a.createVertexBuffer(); err != nil {
		return err
	}
	if err := a.createIndexBuffer(); err != nil {
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
	if err := a.createSyncObjects(); err != nil {
		return err
	}
	a.startTime = time.Now()
	return nil
}

func (a *VulkanApp) createInstance() error {
	if a.cfg.enableValidation && !a.validationLayersSupported() {
		return errors.New("requested validation layers not available")
	}

	if !glfw.VulkanSupported() {
		return errors.New("GLFW Vulkan loader not found")
	}

	appInfo := vulkan.ApplicationInfo{
		SType:              vulkan.StructureTypeApplicationInfo,
		PApplicationName:   "Kube Vulkan",
		ApplicationVersion: vulkan.MakeVersion(0, 1, 0),
		PEngineName:        "No Engine",
		EngineVersion:      vulkan.MakeVersion(0, 1, 0),
		ApiVersion:         vulkan.MakeVersion(1, 1, 0),
	}

	extensions := a.window.GetRequiredInstanceExtensions()
	if a.cfg.enableValidation {
		extensions = append(extensions, "VK_EXT_debug_report")
	}

	createInfo := vulkan.InstanceCreateInfo{
		SType:                   vulkan.StructureTypeInstanceCreateInfo,
		PApplicationInfo:        &appInfo,
		EnabledExtensionCount:   uint32(len(extensions)),
		PpEnabledExtensionNames: extensions,
	}
	if a.cfg.enableValidation {
		createInfo.EnabledLayerCount = uint32(len(validationLayers))
		createInfo.PpEnabledLayerNames = validationLayers
	}

	if res := vulkan.CreateInstance(&createInfo, nil, &a.instance); res != vulkan.Success {
		return fmt.Errorf("create instance: %w", vulkan.Error(res))
	}
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
	createInfo := vulkan.DeviceCreateInfo{
		SType:                   vulkan.StructureTypeDeviceCreateInfo,
		PQueueCreateInfos:       queueInfos,
		QueueCreateInfoCount:    uint32(len(queueInfos)),
		PEnabledFeatures:        []vulkan.PhysicalDeviceFeatures{deviceFeatures},
		PpEnabledExtensionNames: deviceExtensions,
		EnabledExtensionCount:   uint32(len(deviceExtensions)),
	}
	if a.cfg.enableValidation {
		createInfo.EnabledLayerCount = uint32(len(validationLayers))
		createInfo.PpEnabledLayerNames = validationLayers
	}

	if res := vulkan.CreateDevice(a.physicalDevice, &createInfo, nil, &a.device); res != vulkan.Success {
		return fmt.Errorf("create logical device: %w", vulkan.Error(res))
	}

	vulkan.GetDeviceQueue(a.device, a.queues.graphicsFamily, 0, &a.graphicsQueue)
	vulkan.GetDeviceQueue(a.device, a.queues.presentFamily, 0, &a.presentQueue)
	return nil
}

func (a *VulkanApp) querySwapchainSupport(device vulkan.PhysicalDevice) swapchainSupport {
	var details swapchainSupport
	vulkan.GetPhysicalDeviceSurfaceCapabilities(device, a.surface, &details.capabilities)
	details.capabilities.Deref()

	var formatCount uint32
	vulkan.GetPhysicalDeviceSurfaceFormats(device, a.surface, &formatCount, nil)
	if formatCount > 0 {
		details.formats = make([]vulkan.SurfaceFormat, formatCount)
		vulkan.GetPhysicalDeviceSurfaceFormats(device, a.surface, &formatCount, details.formats)
		for i := range details.formats {
			details.formats[i].Deref()
		}
	}

	var presentCount uint32
	vulkan.GetPhysicalDeviceSurfacePresentModes(device, a.surface, &presentCount, nil)
	if presentCount > 0 {
		details.presentModes = make([]vulkan.PresentMode, presentCount)
		vulkan.GetPhysicalDeviceSurfacePresentModes(device, a.surface, &presentCount, details.presentModes)
	}

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
	if caps.CurrentExtent.Width != math.MaxUint32 {
		return caps.CurrentExtent
	}
	w, h := window.GetFramebufferSize()
	extent := vulkan.Extent2D{
		Width:  uint32(w),
		Height: uint32(h),
	}
	min := caps.MinImageExtent
	max := caps.MaxImageExtent
	extent.Width = uint32(clamp(uint64(extent.Width), uint64(min.Width), uint64(max.Width)))
	extent.Height = uint32(clamp(uint64(extent.Height), uint64(min.Height), uint64(max.Height)))
	return extent
}

func (a *VulkanApp) createSwapchain() error {
	support := a.querySwapchainSupport(a.physicalDevice)

	surfaceFormat := chooseSwapSurfaceFormat(support.formats)
	presentMode := chooseSwapPresentMode(support.presentModes)
	extent := chooseSwapExtent(support.capabilities, a.window)

	imageCount := support.capabilities.MinImageCount + 1
	if support.capabilities.MaxImageCount > 0 && imageCount > support.capabilities.MaxImageCount {
		imageCount = support.capabilities.MaxImageCount
	}

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

	if res := vulkan.CreateSwapchain(a.device, &createInfo, nil, &a.swapchain); res != vulkan.Success {
		return fmt.Errorf("create swapchain: %w", vulkan.Error(res))
	}

	var count uint32
	vulkan.GetSwapchainImages(a.device, a.swapchain, &count, nil)
	a.swapchainImages = make([]vulkan.Image, count)
	vulkan.GetSwapchainImages(a.device, a.swapchain, &count, a.swapchainImages)
	a.swapchainFormat = surfaceFormat.Format
	a.swapchainExtent = extent
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
		return fmt.Errorf("create depth image view: %w", err)
	}
	a.depthImage = image
	a.depthImageMemory = memory
	a.depthImageView = view
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

	var image vulkan.Image
	if res := vulkan.CreateImage(a.device, &createInfo, nil, &image); res != vulkan.Success {
		return vulkan.Image(vulkan.NullHandle), vulkan.DeviceMemory(vulkan.NullHandle), fmt.Errorf("create image: %w", vulkan.Error(res))
	}

	var memRequirements vulkan.MemoryRequirements
	vulkan.GetImageMemoryRequirements(a.device, image, &memRequirements)
	memRequirements.Deref()

	allocInfo := vulkan.MemoryAllocateInfo{
		SType:           vulkan.StructureTypeMemoryAllocateInfo,
		AllocationSize:  memRequirements.Size,
		MemoryTypeIndex: a.findMemoryType(memRequirements.MemoryTypeBits, properties),
	}

	var memory vulkan.DeviceMemory
	if res := vulkan.AllocateMemory(a.device, &allocInfo, nil, &memory); res != vulkan.Success {
		return vulkan.Image(vulkan.NullHandle), vulkan.DeviceMemory(vulkan.NullHandle), fmt.Errorf("allocate image memory: %w", vulkan.Error(res))
	}

	if res := vulkan.BindImageMemory(a.device, image, memory, 0); res != vulkan.Success {
		return vulkan.Image(vulkan.NullHandle), vulkan.DeviceMemory(vulkan.NullHandle), fmt.Errorf("bind image memory: %w", vulkan.Error(res))
	}

	return image, memory, nil
}

func (a *VulkanApp) findMemoryType(typeFilter uint32, properties vulkan.MemoryPropertyFlagBits) uint32 {
	var memProps vulkan.PhysicalDeviceMemoryProperties
	vulkan.GetPhysicalDeviceMemoryProperties(a.physicalDevice, &memProps)
	memProps.Deref()

	for i := uint32(0); i < memProps.MemoryTypeCount; i++ {
		memoryType := memProps.MemoryTypes[i]
		memoryType.Deref()
		if typeFilter&(1<<i) != 0 && memoryType.PropertyFlags&vulkan.MemoryPropertyFlags(properties) == vulkan.MemoryPropertyFlags(properties) {
			return i
		}
	}
	return 0
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
	var view vulkan.ImageView
	if res := vulkan.CreateImageView(a.device, &viewInfo, nil, &view); res != vulkan.Success {
		return vulkan.ImageView(vulkan.NullHandle), fmt.Errorf("create image view: %w", vulkan.Error(res))
	}
	return view, nil
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

	if res := vulkan.CreateRenderPass(a.device, &createInfo, nil, &a.renderPass); res != vulkan.Success {
		return fmt.Errorf("create render pass: %w", vulkan.Error(res))
	}
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
	poolSize := vulkan.DescriptorPoolSize{
		Type:            vulkan.DescriptorTypeUniformBuffer,
		DescriptorCount: uint32(len(a.swapchainImages)),
	}
	poolInfo := vulkan.DescriptorPoolCreateInfo{
		SType:         vulkan.StructureTypeDescriptorPoolCreateInfo,
		MaxSets:       uint32(len(a.swapchainImages)),
		PoolSizeCount: 1,
		PPoolSizes:    []vulkan.DescriptorPoolSize{poolSize},
	}
	if res := vulkan.CreateDescriptorPool(a.device, &poolInfo, nil, &a.descriptorPool); res != vulkan.Success {
		return fmt.Errorf("create descriptor pool: %w", vulkan.Error(res))
	}
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
	a.descriptorSets = make([]vulkan.DescriptorSet, len(a.swapchainImages))
	if res := vulkan.AllocateDescriptorSets(a.device, &allocInfo, &a.descriptorSets[0]); res != vulkan.Success {
		return fmt.Errorf("allocate descriptor sets: %w", vulkan.Error(res))
	}

	for i := range a.descriptorSets {
		bufferInfo := vulkan.DescriptorBufferInfo{
			Buffer: a.uniformBuffers[i],
			Offset: 0,
			Range:  vulkan.DeviceSize(unsafe.Sizeof(uniformBufferObject{})),
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
		vulkan.UpdateDescriptorSets(a.device, 1, []vulkan.WriteDescriptorSet{write}, 0, nil)
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
	layoutInfo := vulkan.DescriptorSetLayoutCreateInfo{
		SType:        vulkan.StructureTypeDescriptorSetLayoutCreateInfo,
		BindingCount: 1,
		PBindings:    []vulkan.DescriptorSetLayoutBinding{uLayoutBinding},
	}
	if res := vulkan.CreateDescriptorSetLayout(a.device, &layoutInfo, nil, &a.descriptorSetLayout); res != vulkan.Success {
		return fmt.Errorf("create descriptor set layout: %w", vulkan.Error(res))
	}
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
	if res := vulkan.CreatePipelineLayout(a.device, &pipelineLayoutInfo, nil, &a.pipelineLayout); res != vulkan.Success {
		return fmt.Errorf("create pipeline layout: %w", vulkan.Error(res))
	}

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

	pipelines := make([]vulkan.Pipeline, 1)
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
	var buffer vulkan.Buffer
	if res := vulkan.CreateBuffer(a.device, &bufferInfo, nil, &buffer); res != vulkan.Success {
		return vulkan.Buffer(vulkan.NullHandle), vulkan.DeviceMemory(vulkan.NullHandle), fmt.Errorf("create buffer: %w", vulkan.Error(res))
	}
	var memReq vulkan.MemoryRequirements
	vulkan.GetBufferMemoryRequirements(a.device, buffer, &memReq)
	memReq.Deref()
	allocInfo := vulkan.MemoryAllocateInfo{
		SType:           vulkan.StructureTypeMemoryAllocateInfo,
		AllocationSize:  memReq.Size,
		MemoryTypeIndex: a.findMemoryType(memReq.MemoryTypeBits, properties),
	}
	var bufferMemory vulkan.DeviceMemory
	if res := vulkan.AllocateMemory(a.device, &allocInfo, nil, &bufferMemory); res != vulkan.Success {
		vulkan.DestroyBuffer(a.device, buffer, nil)
		return vulkan.Buffer(vulkan.NullHandle), vulkan.DeviceMemory(vulkan.NullHandle), fmt.Errorf("allocate buffer memory: %w", vulkan.Error(res))
	}
	vulkan.BindBufferMemory(a.device, buffer, bufferMemory, 0)
	return buffer, bufferMemory, nil
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
		if res := vulkan.CreateFramebuffer(a.device, &createInfo, nil, &a.framebuffers[i]); res != vulkan.Success {
			return fmt.Errorf("create framebuffer %d: %w", i, vulkan.Error(res))
		}
	}
	return nil
}

func (a *VulkanApp) createCommandPool() error {
	poolInfo := vulkan.CommandPoolCreateInfo{
		SType:            vulkan.StructureTypeCommandPoolCreateInfo,
		QueueFamilyIndex: a.queues.graphicsFamily,
		Flags:            vulkan.CommandPoolCreateFlags(vulkan.CommandPoolCreateResetCommandBufferBit),
	}
	if res := vulkan.CreateCommandPool(a.device, &poolInfo, nil, &a.commandPool); res != vulkan.Success {
		return fmt.Errorf("create command pool: %w", vulkan.Error(res))
	}
	return nil
}

func (a *VulkanApp) allocateCommandBuffers() error {
	allocInfo := vulkan.CommandBufferAllocateInfo{
		SType:              vulkan.StructureTypeCommandBufferAllocateInfo,
		CommandPool:        a.commandPool,
		Level:              vulkan.CommandBufferLevelPrimary,
		CommandBufferCount: uint32(len(a.framebuffers)),
	}
	a.commandBuffers = make([]vulkan.CommandBuffer, len(a.framebuffers))
	if res := vulkan.AllocateCommandBuffers(a.device, &allocInfo, a.commandBuffers); res != vulkan.Success {
		return fmt.Errorf("allocate command buffers: %w", vulkan.Error(res))
	}
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
		if res := vulkan.CreateSemaphore(a.device, &semInfo, nil, &a.imageAvailable[i]); res != vulkan.Success {
			return fmt.Errorf("create imageAvailable semaphore %d: %w", i, vulkan.Error(res))
		}
		if res := vulkan.CreateSemaphore(a.device, &semInfo, nil, &a.renderFinished[i]); res != vulkan.Success {
			return fmt.Errorf("create renderFinished semaphore %d: %w", i, vulkan.Error(res))
		}
		if res := vulkan.CreateFence(a.device, &fenceInfo, nil, &a.inFlightFences[i]); res != vulkan.Success {
			return fmt.Errorf("create fence %d: %w", i, vulkan.Error(res))
		}
	}
	return nil
}

func (a *VulkanApp) requestSwapchainRecreate() {
	a.framebufferResized = true
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

	clearColor := vulkan.NewClearValue([]float32{0.05, 0.05, 0.08, 1.0})
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
