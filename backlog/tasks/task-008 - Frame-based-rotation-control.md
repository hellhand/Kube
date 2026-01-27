---
id: TASK-008
title: Frame-based rotation control
status: Todo
assignee: []
created_date: '2026-01-27 21:01'
labels: []
dependencies:
  - TASK-007
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Replace time-based spin with frame-based angular steps (see `updateUniformBuffer` using `debugFrames` as the counter). Gate increments when paused (SPACE) so the angle freezes, then resumes smoothly. Expose/tune degrees-per-frame (e.g., 45°/s at 60 fps → 0.75° per frame), and wrap/normalize angle as needed. Document the approach so another dev can tweak the per-frame delta.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Rotation angle advances by a fixed per-frame delta (e.g., configurable degrees per frame) instead of elapsed time.
- [ ] #2 Pause/resume (SPACE) leaves the angle frozen while paused and resumes from the same orientation without jumps.
- [ ] #3 Behavior is documented in code comments or README so the chosen per-frame rate is discoverable.
- [ ] #4 Manual run shows consistent rotation speed regardless of CPU load or vsync timing jitter.
<!-- AC:END -->
