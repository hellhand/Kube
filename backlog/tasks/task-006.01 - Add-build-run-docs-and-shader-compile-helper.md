---
id: TASK-006.01
title: Add build/run docs and shader compile helper
status: To Do
assignee: []
created_date: '2026-01-27 16:41'
labels: []
dependencies:
  - TASK-005.02
parent_task_id: TASK-006
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Provide developer docs and scripts for building/running the cube, including shader compilation (glslc or equivalent) and any required env vars.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 README (or docs) documents build/run steps, required env vars, and platform notes.
- [ ] #2 Script or Make target compiles GLSL shaders to SPIR-V deterministically.
- [ ] #3 Convenient run target/command launches the app using the built shaders.
<!-- AC:END -->
