---
id: TASK-003
title: 'Milestone: Swapchain and frame scheduling'
status: Done
assignee: []
created_date: '2026-01-27 16:39'
labels: []
dependencies: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Parent milestone for swapchain creation, framebuffers, command buffers, and sync needed for a basic clear-color loop: create swapchain with present/graphics sharing as needed, image views + depth buffer, render pass/framebuffers, primary command buffers, and per-frame semaphores/fences for acquire/submit/present.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 All child tasks are done and the app can acquire/present frames reliably (even if it only clears the screen).
<!-- AC:END -->
