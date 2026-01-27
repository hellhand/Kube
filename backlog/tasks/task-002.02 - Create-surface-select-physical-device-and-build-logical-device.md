---
id: TASK-002.02
title: 'Create surface, select physical device, and build logical device'
status: Done
assignee: []
created_date: '2026-01-27 16:40'
labels: []
dependencies:
  - TASK-002.01
parent_task_id: TASK-002
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Create the window surface, choose a GPU with graphics+present support, and create the logical device/queues for rendering. Ensure swapchain support query is logged, queue family indices are copied into C memory for swapchain creation, and the device enables `VK_KHR_swapchain` plus both graphics and present queues.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Surface created from GLFW window and cleaned up on shutdown.
- [x] #2 Physical device selection checks swapchain + graphics/present queue families and prefers discrete GPU when available.
- [x] #3 Logical device exposes graphics and present queues; chosen device details logged.
<!-- AC:END -->
