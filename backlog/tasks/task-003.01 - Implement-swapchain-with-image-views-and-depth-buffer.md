---
id: TASK-003.01
title: Implement swapchain with image views and depth buffer
status: To Do
assignee: []
created_date: '2026-01-27 16:40'
labels: []
dependencies:
  - TASK-002.02
parent_task_id: TASK-003
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Create the swapchain, color image views, and a depth buffer suitable for a 3D cube, with recreation support on resize or surface changes.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Swapchain chooses format/present mode, image count, and extent from surface capabilities.
- [ ] #2 Color image views and a depth image with view allocated/cleaned alongside the swapchain.
- [ ] #3 Resize/surface-loss triggers safe swapchain (and depth) recreation without leaks.
<!-- AC:END -->
