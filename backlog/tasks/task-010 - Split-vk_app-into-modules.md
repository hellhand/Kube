---
id: TASK-010
title: Split vk_app.go into logical modules
status: Todo
assignee: []
created_date: '2026-01-27 21:20'
labels: []
dependencies:
  - TASK-007
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Refactor the monolithic `vk_app.go` into logical Go files (types/init, swapchain/pipeline, resources, overlay, helpers, etc.) while preserving behavior. Keep all code in package `main`, ensure imports stay minimal, and maintain build/test parity.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 `vk_app.go` is reduced to core orchestration, with related concerns moved into clearly named files (e.g., textures, overlay/HUD, swapchain/pipeline helpers).
- [ ] #2 Build succeeds (`go build ./...`) with no behavioral changes (cube, texture, overlay, input controls still work).
- [ ] #3 New files have focused responsibilities and minimal imports; no duplicated code or dead functions.
- [ ] #4 Cleanup/recreation paths still free resources correctly across files.
<!-- AC:END -->
