---
id: TASK-005.02
title: Implement resize-safe swapchain/pipeline recreation and cleanup
status: Done
assignee: []
created_date: '2026-01-27 16:40'
labels: []
dependencies:
  - TASK-005.01
parent_task_id: TASK-005
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Detect out-of-date/resize events, tear down and rebuild swapchain-dependent resources (framebuffers, pipeline, depth), and ensure orderly cleanup on exit.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Resize or VK_ERROR_OUT_OF_DATE_KHR triggers swapchain + depth + framebuffer rebuild without crashes or leaks.
- [x] #2 Graphics pipeline and descriptor bindings remain valid after recreation (or are rebuilt as needed).
- [x] #3 Shutdown path destroys Vulkan resources in correct order with validation clean.
<!-- AC:END -->
