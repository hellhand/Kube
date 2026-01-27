---
id: TASK-002.01
title: Create Vulkan instance with validation layers and debug messenger
status: To Do
assignee: []
created_date: '2026-01-27 16:40'
labels: []
dependencies:
  - TASK-001.02
parent_task_id: TASK-002
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Initialize Vulkan instance for GLFW surface creation, request validation layers conditionally, and wire a debug messenger to log validation output.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Instance enables GLFW-required extensions and optional validation layers via flag/env.
- [ ] #2 Debug messenger installed and logs validation messages to stdout/stderr.
- [ ] #3 Startup fails clearly if required extensions/layers are unavailable.
<!-- AC:END -->
