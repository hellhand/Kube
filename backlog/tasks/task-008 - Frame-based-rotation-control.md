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
Replace time-based spin with frame-based angular steps (as now done via `debugFrames` counter in `updateUniformBuffer`); keep SPACE pause/resume by gating the frame counter, and allow tuning degrees-per-frame (currently ~0.75°/frame for 45°/s at 60 fps). Document the choice and keep animation accumulation deterministic.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Rotation angle advances by a fixed per-frame delta (e.g., configurable degrees per frame) instead of elapsed time.
- [ ] #2 Pause/resume (SPACE) leaves the angle frozen while paused and resumes from the same orientation without jumps.
- [ ] #3 Behavior is documented in code comments or README so the chosen per-frame rate is discoverable.
- [ ] #4 Manual run shows consistent rotation speed regardless of CPU load or vsync timing jitter.
<!-- AC:END -->
