---
id: TASK-007
title: 'Milestone: Texturing and vkcube visual parity'
status: In Progress
assignee: []
created_date: '2026-01-27 20:35'
labels: []
dependencies: [TASK-005]
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Add a textured cube and adjust the background/visuals to mirror the classic `vkcube` demo. Steps: embed the `lunarg.ppm` texture (`texture_embed.go`), parse it to RGBA in `vk_texture.go`, create image/view/sampler (R8G8B8A8 SRGB, shader-read layout). Use 24-vertex cube layout with per-face UVs and indices. Update shaders to read position/color/UV and sample the texture. Descriptor set layout binds UBO at 0 and combined image sampler at 1. Disable culling to keep all faces visible, keep depth testing on, and clear to the vkcube-style green background. Input controls: ESC to close, SPACE to pause rotation.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Cube renders with texture sampling (combined image sampler bound alongside the uniform buffer).
- [ ] #2 Background clear color matches the intended vkcube dark backdrop.
- [ ] #3 Depth testing is enabled and faces do not disappear during rotation.
- [ ] #4 Shaders and SPIR-V are updated to consume position, color, and UV inputs.
- [ ] #5 Cube uses the classic vkcube `lunarg.ppm` texture with correct UV mapping per face.
- [ ] #6 Runtime controls: ESC closes the window; SPACE toggles cube rotation without hitching.
<!-- AC:END -->
