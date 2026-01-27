---
id: TASK-009
title: Display FPS overlay
status: Todo
assignee: []
created_date: '2026-01-27 21:05'
labels: []
dependencies:
  - TASK-007
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Add an FPS indicator rendered in the top-left using the overlay pipeline (`vk_overlay.go` + `overlay.vert/frag`). Build a tiny font atlas as bit patterns, expand to screen-space quads, convert to NDC, and write into an overlay vertex buffer; issue a single indirect draw. Compute FPS over a 1s window (frame count / elapsed), update each frame, and ensure the overlay pipeline uses no depth and an opaque color blend so it draws over the scene.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 FPS value is drawn in the top-left corner each frame and updates at least once per second.
- [ ] #2 Overlay does not disrupt existing rendering (cube, background, depth) and works after resize/swapchain recreation.
- [ ] #3 Calculation uses measured frame times (or frame counts over a 1s window), not a hard-coded number.
- [ ] #4 Minimal visual styling (legible text or simple digits) and code is documented for how FPS is derived.
<!-- AC:END -->
