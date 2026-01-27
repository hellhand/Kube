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
Add a textured cube and adjust the background/visuals to mirror the classic `vkcube` demo: bind UBO + combined image sampler, use the embedded `lunarg.ppm` texture (see `texture_embed.go`), ensure UV-correct faces via 24-vertex layout, disable culling for visibility, keep depth testing on, and use the vkcube-style green backdrop.
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
