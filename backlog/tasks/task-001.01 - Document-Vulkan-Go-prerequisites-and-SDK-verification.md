---
id: TASK-001.01
title: Document Vulkan/Go prerequisites and SDK verification
status: Done
assignee: []
created_date: '2026-01-27 16:40'
labels: []
dependencies: []
parent_task_id: TASK-001
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Capture OS-specific Vulkan SDK setup, Go toolchain versions, required headers/libraries (GLFW/X11 dev packages), and how to verify the runtime before coding. Include sample commands (`vulkaninfo`, `glslc --version`, minimal `go run` window with `glfw.NoAPI`), expected output, and env vars for validation layers (e.g., `VK_INSTANCE_LAYERS`, layer/ICD paths).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 README documents Vulkan SDK install/driver requirements and Go version.
- [x] #2 Verification command (e.g., vulkaninfo or vkcube) documented with expected success output.
- [x] #3 Notes on enabling validation layers (layer path/ICD env vars) recorded.
<!-- AC:END -->
