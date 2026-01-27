# Kube: Vulkan vkcube (Go)

This project will reimplement `vkcube` in Go using Vulkan. The notes below capture the platform prerequisites and how to verify your environment before we begin coding.

## Prerequisites
- Go 1.22+ (set `GOVERSION` to a matching toolchain if you use `go env -w GOVERSION=1.22.0`)
- A Vulkan-capable GPU and up-to-date drivers that expose at least Vulkan 1.1
- Vulkan SDK installed (for headers, loader, validation layers, and tools)
- Build essentials (C toolchain) to compile GLFW/Vulkan C bindings when needed

## Install and verify Vulkan SDK
Linux:
- Install vendor GPU drivers (NVIDIA/AMD/Intel) with Vulkan support.
- Install the LunarG Vulkan SDK (tarball or your distro packages). Typical install sets `VULKAN_SDK=/path/to/vulkan/<version>/x86_64`.

macOS:
- Install Xcode command-line tools.
- Install Vulkan SDK with MoltenVK (LunarG installer). `VULKAN_SDK` is set by the installer; MoltenVK provides the ICD.

Windows:
- Install GPU drivers with Vulkan.
- Install the LunarG Vulkan SDK (sets `VULKAN_SDK` and adds tools like `vulkaninfo`).

## Environment variables (validation layers and ICDs)
- `VULKAN_SDK` (Linux/macOS/Windows): root of the SDK.
- `VK_LAYER_PATH`: point to the SDK layers if the loader cannot find them.
  - Linux: `${VULKAN_SDK}/share/vulkan/explicit_layer.d`
  - macOS: `${VULKAN_SDK}/share/vulkan/explicit_layer.d`
  - Windows: `%VULKAN_SDK%\\Bin` is usually registered automatically; override with `%VULKAN_SDK%\\Bin`
- `VK_INSTANCE_LAYERS=VK_LAYER_KHRONOS_validation` enables validation layers when present.
- `VK_ICD_FILENAMES` (only if you need to override ICD discovery):
  - Linux example: `/usr/share/vulkan/icd.d/nvidia_icd.json` (choose your driver’s JSON)
  - macOS uses MoltenVK’s ICD from the SDK: `${VULKAN_SDK}/share/vulkan/icd.d/MoltenVK_icd.json`

## Verification before coding
- `vulkaninfo` should exit 0 and print the GPU name and `Vulkan Instance Version` (expect 1.1+).
- `vkcube` (from the SDK demos) should open a spinning cube window without validation errors.
- With validation enabled (`VK_INSTANCE_LAYERS=VK_LAYER_KHRONOS_validation` and, if needed, `VK_LAYER_PATH`), `vkcube` should run and emit validation logs to stdout/stderr without errors.
- If `vulkaninfo` fails, check driver install and `VK_ICD_FILENAMES`. If validation fails to load, check `VK_LAYER_PATH`.

These checks ensure the system can create instances, load validation layers, and present surfaces before we wire up the Go application.
