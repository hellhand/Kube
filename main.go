package main

import (
	"log"
	"runtime"
	"time"

	"github.com/vulkan-go/glfw/v3.3/glfw"
)

func init() {
	// GLFW/Vulkan require the main thread.
	runtime.LockOSThread()
}

func main() {
	if err := glfw.Init(); err != nil {
		log.Fatalf("init glfw: %v", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	window, err := glfw.CreateWindow(800, 600, "Kube Vulkan (baseline window)", nil, nil)
	if err != nil {
		log.Fatalf("create window: %v", err)
	}
	defer window.Destroy()

	// Ensure the framebuffer has a non-zero size before initializing Vulkan.
	for {
		w, h := window.GetFramebufferSize()
		if w > 0 && h > 0 {
			break
		}
		glfw.WaitEventsTimeout(0.01)
	}

	window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		if key == glfw.KeyEscape && action == glfw.Press {
			w.SetShouldClose(true)
		}
	})

	app, err := newVulkanApp(window)
	if err != nil {
		log.Fatalf("init vulkan: %v", err)
	}
	defer app.Cleanup()

	window.SetFramebufferSizeCallback(func(w *glfw.Window, width int, height int) {
		app.requestSwapchainRecreate()
	})

	log.Printf("Entering main loop")

	for !window.ShouldClose() {
		glfw.PollEvents()
		if err := app.DrawFrame(); err != nil {
			log.Fatalf("draw frame: %v", err)
		}
		time.Sleep(1 * time.Millisecond) // small throttle to avoid busy loop
	}
}
