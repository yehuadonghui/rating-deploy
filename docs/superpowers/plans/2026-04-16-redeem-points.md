# Redeem Points Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add redeem points to student primary data with automatic contest awards and manual ledger entries.

**Architecture:** Extend contest and student models, add a point-record ledger plus a small service layer for award generation and total recomputation, then surface the new fields in existing admin flows. Keep redeem points separate from rating replay.

**Tech Stack:** Go, Gin, GORM, SQLite, React, TypeScript, Ant Design

---

### Task 1: Backend Schema

**Files:**
- Modify: `backend/internal/models/student.go`
- Modify: `backend/internal/models/contest.go`
- Modify: `backend/internal/models/config.go`
- Modify: `backend/internal/database/migrations.go`

- [ ] **Step 1: Write the failing test**

Add backend tests that expect `redeem_points`, `reward_type`, and `point_records` schema-backed behavior.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/services/...`

- [ ] **Step 3: Write minimal implementation**

Add model fields and migration support.

- [ ] **Step 4: Run test to verify it passes**

Run the targeted backend tests again.

- [ ] **Step 5: Commit**

Commit schema changes once tests pass.

### Task 2: Auto Contest Award Service

**Files:**
- Create: `backend/internal/services/points/service.go`
- Create: `backend/internal/services/points/service_test.go`

- [ ] **Step 1: Write the failing test**

Cover prize-tier cutoff behavior, participation fallback, and contest-type point table selection.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/services/points -v`

- [ ] **Step 3: Write minimal implementation**

Implement ledger generation and student-total recomputation helpers.

- [ ] **Step 4: Run test to verify it passes**

Run the same package tests.

- [ ] **Step 5: Commit**

Commit the service layer once tests pass.

### Task 3: Admin Contest Integration

**Files:**
- Modify: `backend/internal/handlers/admin.go`

- [ ] **Step 1: Write the failing test**

Add handler/service integration tests for contest upload/edit/delete point side effects where practical.

- [ ] **Step 2: Run test to verify it fails**

Run the targeted backend test package.

- [ ] **Step 3: Write minimal implementation**

Accept `reward_type` in contest create/update and trigger point-record regeneration.

- [ ] **Step 4: Run test to verify it passes**

Re-run the targeted backend tests.

- [ ] **Step 5: Commit**

Commit contest integration changes.

### Task 4: Manual Point Entry API

**Files:**
- Modify: `backend/internal/handlers/admin_students.go`
- Modify: `frontend/src/api/admin.ts`
- Modify: `frontend/src/types.ts`

- [ ] **Step 1: Write the failing test**

Add tests for manual point-record creation and student total refresh.

- [ ] **Step 2: Run test to verify it fails**

Run the targeted backend test package.

- [ ] **Step 3: Write minimal implementation**

Add admin endpoint and frontend client method.

- [ ] **Step 4: Run test to verify it passes**

Run the targeted tests again.

- [ ] **Step 5: Commit**

Commit the manual-entry API.

### Task 5: Admin UI Surface

**Files:**
- Modify: `frontend/src/pages/admin/Contests.tsx`
- Modify: `frontend/src/pages/admin/Students.tsx`
- Modify: `frontend/src/types.ts`

- [ ] **Step 1: Write the failing test**

Add UI or type-level coverage if test harness exists; otherwise verify via build.

- [ ] **Step 2: Run test to verify it fails**

Run: `npm run build`

- [ ] **Step 3: Write minimal implementation**

Add reward-type selectors, show student redeem points, and add a manual-point entry flow in students admin.

- [ ] **Step 4: Run test to verify it passes**

Run: `npm run build`

- [ ] **Step 5: Commit**

Commit UI changes.

### Task 6: Final Verification

**Files:**
- Review: `backend/internal/services/points/service_test.go`
- Review: `backend/internal/handlers/...`
- Review: `frontend/src/pages/admin/...`

- [ ] **Step 1: Run targeted backend tests**

Run: `go test ./internal/services/points ./internal/services/rating -run "TestReplayAll_AppliesHistoryDecayToOlderContests|TestPreviewAll_MatchesReplayAllWhenRatingsDrop|Test.*Points.*" ./...`

- [ ] **Step 2: Run frontend build**

Run: `npm run build`

- [ ] **Step 3: Review diffs**

Check that unrelated dirty files were not reverted.

- [ ] **Step 4: Summarize residual risks**

Document that qualification-award automation is still pending.
