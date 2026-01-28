package main

import (
	"log"
	"runtime"
	"time"

	"github.com/vulkan-go/glfw/v3.3/glfw"
)

func init() {
	// init locks the OS thread because GLFW/Vulkan expect calls from a single thread.
	runtime.LockOSThread()
}

// main boots GLFW, creates the window, wires input callbacks, and runs the render loop.
func main() {
	if err := glfw.Init(); err != nil {
		log.Fatalf("init glfw: %v", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	glfw.WindowHint(glfw.Resizable, glfw.True)
	window, err := glfw.CreateWindow(800, 600, "Kube Vulkan (baseline window)", nil, nil)
	if err != nil {
		log.Fatalf("create window: %v", err)
	}
	defer window.Destroy()

	var app *VulkanApp
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
		if key == glfw.KeySpace && action == glfw.Press && app != nil {
			app.togglePause()
		}
	})

	app, err = newVulkanApp(window)
	if err != nil {
		log.Fatalf("init vulkan: %v", err)
	}
	defer app.Cleanup()

	window.SetFramebufferSizeCallback(func(w *glfw.Window, width int, height int) {
		app.requestSwapchainRecreate()
	})

	log.Printf("Entering main loop")

	for !window.ShouldClose() {
		frameStart := time.Now()
		glfw.PollEvents()
		if err := app.DrawFrame(); err != nil {
			log.Fatalf("draw frame: %v", err)
		}
		if app.cfg.maxFPS > 0 {
			target := time.Second / time.Duration(app.cfg.maxFPS)
			if sleep := target - time.Since(frameStart); sleep > 0 {
				time.Sleep(sleep)
			}
		} else {
			time.Sleep(1 * time.Millisecond) // small throttle to avoid busy loop
		}
	}
}
