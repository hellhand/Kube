---
id: TASK-003.02
title: 'Render pass, framebuffers, and frame loop with sync'
status: To Do
assignee: []
created_date: '2026-01-27 16:40'
labels: []
dependencies:
  - TASK-003.01
parent_task_id: TASK-003
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Build the render pass (color+depth), framebuffers, command pool/buffers, and per-frame sync objects to run a clear-color acquire/draw/present loop.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Render pass defines color+depth attachments compatible with the swapchain.
- [ ] #2 Framebuffers, command pool, and command buffers allocated per swapchain image.
- [ ] #3 Per-frame semaphores/fences support at least double buffering and the clear-color loop presents without stalling.
<!-- AC:END -->
