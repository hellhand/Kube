---
id: TASK-009
title: Display FPS overlay
status: Todo
assignee: []
created_date: '2026-01-27 21:05'
labels: []
dependencies:
  - TASK-007
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Add an FPS indicator rendered in the top-left of the window so we can monitor frame timing during the vkcube clone. Keep it lightweight and non-intrusive.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 FPS value is drawn in the top-left corner each frame and updates at least once per second.
- [ ] #2 Overlay does not disrupt existing rendering (cube, background, depth) and works after resize/swapchain recreation.
- [ ] #3 Calculation uses measured frame times (or frame counts over a 1s window), not a hard-coded number.
- [ ] #4 Minimal visual styling (legible text or simple digits) and code is documented for how FPS is derived.
<!-- AC:END -->
