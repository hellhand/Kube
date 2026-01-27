---
id: TASK-005.01
title: Record draw commands for rotating cube
status: Done
assignee: []
created_date: '2026-01-27 16:40'
labels: []
dependencies:
  - TASK-004.02
parent_task_id: TASK-005
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Record command buffers that bind the cube pipeline/resources, draw the indexed cube, and drive a render loop that updates rotation over time. Bind pipeline, vertex/index buffers, descriptor sets per swapchain image, issue `vkCmdDrawIndexed`, and handle clear values for color+depth.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Command buffers bind pipeline, vertex/index buffers, descriptor sets, and issue indexed draw for cube.
- [x] #2 Per-frame uniform updates apply rotation and basic camera/view/projection transforms.
- [x] #3 Render loop presents rotating cube frames without validation errors.
<!-- AC:END -->
