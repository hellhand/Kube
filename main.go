package main

import (
	"log"
	"runtime"
	"time"

	mgl32 "github.com/go-gl/mathgl/mgl32"
	"github.com/vulkan-go/glfw/v3.3/glfw"
	_ "github.com/vulkan-go/vulkan"
)

func init() {
	// GLFW (and Vulkan later) expect to run on the main OS thread.
	runtime.LockOSThread()
}

func main() {
	if err := glfw.Init(); err != nil {
		log.Fatalf("init glfw: %v", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI) // no OpenGL context; Vulkan will be used later
	window, err := glfw.CreateWindow(800, 600, "Kube Vulkan (baseline window)", nil, nil)
	if err != nil {
		log.Fatalf("create window: %v", err)
	}
	defer window.Destroy()

	window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		if key == glfw.KeyEscape && action == glfw.Press {
			w.SetShouldClose(true)
		}
	})

	// Placeholder math usage; keeps math library available for upcoming transforms.
	_ = mgl32.Ident4()

	for !window.ShouldClose() {
		glfw.PollEvents()
		time.Sleep(16 * time.Millisecond) // simple throttling to avoid pegging the CPU
	}
}
