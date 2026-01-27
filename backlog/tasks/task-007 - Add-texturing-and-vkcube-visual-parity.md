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
Add a textured cube and adjust the background/visuals to mirror the classic `vkcube` demo (textured faces, consistent depth/culling, and dark backdrop).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Cube renders with texture sampling (combined image sampler bound alongside the uniform buffer).
- [ ] #2 Background clear color matches the intended vkcube dark backdrop.
- [ ] #3 Depth testing is enabled and faces do not disappear during rotation.
- [ ] #4 Shaders and SPIR-V are updated to consume position, color, and UV inputs.
<!-- AC:END -->
