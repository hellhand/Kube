module Kube

go 1.24.4

require (
	github.com/go-gl/mathgl v1.2.0
	github.com/vulkan-go/glfw v0.0.0-20210402172934-58379a80228d
	github.com/vulkan-go/vulkan v0.0.0-20221209234627-c0a353ae26c8
)

replace github.com/vulkan-go/vulkan => ./third_party/vulkan
