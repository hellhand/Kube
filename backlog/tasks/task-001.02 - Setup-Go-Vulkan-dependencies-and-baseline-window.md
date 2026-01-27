---
id: TASK-001.02
title: Setup Go Vulkan dependencies and baseline window
status: In Progress
assignee: []
created_date: '2026-01-27 16:40'
labels: []
dependencies:
  - TASK-001.01
parent_task_id: TASK-001
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Add Go dependencies (vulkan-go, glfw, math), clean out placeholder code, and ship a minimal GLFW window loop as the starting point.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 go.mod/go.sum include vulkan-go/vulkan, vulkan-go/glfw, and chosen math library.
- [ ] #2 Placeholder sample code removed; main opens a window and processes close/escape events cleanly.
- [ ] #3 "go run ." builds/runs the window with no validation errors.
<!-- AC:END -->
