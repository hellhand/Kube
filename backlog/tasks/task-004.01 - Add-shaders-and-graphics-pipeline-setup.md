---
id: TASK-004.01
title: Add shaders and graphics pipeline setup
status: Done
assignee: []
created_date: '2026-01-27 16:40'
labels: []
dependencies:
  - TASK-003.02
parent_task_id: TASK-004
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Create shader sources (vertex/fragment) with a build step to SPIR-V, define descriptor/pipeline layouts, and build the graphics pipeline against the swapchain render pass.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 GLSL shader sources stored with a reproducible compile script or documented command.
- [x] #2 Descriptor set layout and pipeline layout include MVP uniform binding for the cube.
- [x] #3 Graphics pipeline (viewport, rasterization, depth, blend state) builds successfully and recreates on swapchain resize.
<!-- AC:END -->
