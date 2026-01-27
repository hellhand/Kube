---
id: TASK-006.02
title: Add smoke test and validation toggles
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
Ship a lightweight smoke test (or self-check mode) plus flags/env toggles for validation layers and debug logging.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Smoke test or self-check runs via go test / CLI flag to verify instance/device creation and exits cleanly.
- [ ] #2 Validation layer toggles are exposed via CLI flag/env var and documented.
- [ ] #3 Logs/reporting clearly show selected device, swapchain format, and validation status.
<!-- AC:END -->
