---
id: TASK-002.01
title: Create Vulkan instance with validation layers and debug messenger
status: Done
assignee: []
created_date: '2026-01-27 16:40'
labels: []
dependencies:
  - TASK-001.02
parent_task_id: TASK-002
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Initialize Vulkan instance for GLFW surface creation, request validation layers conditionally (env override allowed), and wire a debug messenger to log validation output. Include C-allocated C-strings for app/engine names and layer/extension lists to avoid Go pointer issues when validation is enabled.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Instance enables GLFW-required extensions and optional validation layers via flag/env.
- [x] #2 Debug messenger installed and logs validation messages to stdout/stderr.
- [x] #3 Startup fails clearly if required extensions/layers are unavailable.
<!-- AC:END -->
