---
id: TASK-004.02
title: 'Create cube buffers, uniform data, and descriptors'
status: Done
assignee: []
created_date: '2026-01-27 16:40'
labels: []
dependencies:
  - TASK-004.01
parent_task_id: TASK-004
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Define cube vertex/index data (24 vertices for per-face UVs), upload via host-visible buffers, set up per-frame uniform buffers (MVP), and allocate/update descriptor sets. Ensure descriptor set layout matches bindings (0=UBO, 1=sampler), and stage indices/vertices into device buffers with coherent memory flags.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Cube vertex/index buffers allocated and populated (with staging if needed).
- [x] #2 Uniform buffer(s) hold MVP matrix per frame; descriptor sets bind the uniform to the pipeline layout.
- [x] #3 Buffer/descriptor resources cleaned up correctly on shutdown and swapchain/pipeline recreation.
<!-- AC:END -->
