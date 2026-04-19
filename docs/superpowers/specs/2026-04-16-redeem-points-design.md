# Redeem Points Design

**Date:** 2026-04-16

**Goal:** Add a redeem-points system into student primary data, with automatic contest awards and manual non-contest awards.

## Scope

The feature adds a second scoring system alongside `rating`, but stores its cumulative total directly on `students`.

- Automatic points:
  - Foundation monthly exams
  - Brand monthly contests
  - Brand finals
- Manual points:
  - Progress awards
  - Event organization awards

The feature does not change the existing `rating` algorithm or replay pipeline.

## Core Rules

### Contest Types

Each contest gets a `reward_type`:

- `none`
- `foundation_exam`
- `brand_monthly`
- `brand_final`

### Automatic Contest Awards

Awards are based on participant count `N` and rank `R`.

- First prize: `R <= floor(N * 1%)`
- Second prize: `floor(N * 1%) < R <= floor(N * 2%)`
- Third prize: `floor(N * 2%) < R <= floor(N * 3%)`
- Participation prize: everyone with a contest result record who did not get first/second/third prize

If a threshold evaluates to `0`, that prize tier has no winners.

Qualification awards based on score are out of scope for now.

### Point Tables

- `foundation_exam`
  - first: 100
  - second: 80
  - third: 50
  - participation: 5
- `brand_monthly`
  - first: 50
  - second: 30
  - third: 20
  - participation: 2
- `brand_final`
  - first: 100
  - second: 80
  - third: 50
  - participation: 5

### Manual Awards

Manual entries use the same ledger as automatic entries.

- `progress_award`
- `organizer_reward`

## Data Model

### Student

Add `redeem_points` to `students`.

### Contest

Add `reward_type` to `contests`.

### Point Records

Add `point_records` table:

- `id`
- `student_id`
- `contest_id` nullable
- `category`
- `source`
- `points`
- `title`
- `description`
- `created_at`

`category` values:

- `contest_award`
- `progress_award`
- `organizer_reward`

`source` values:

- `auto`
- `manual`

## Backend Behavior

### Contest Upload

Contest upload accepts `reward_type`.

After contest creation and result import:

- if `reward_type == none`, do nothing
- otherwise, generate automatic point records for that contest
- then recompute affected students' `redeem_points`

### Contest Edit

Contest edit accepts `reward_type`.

If the contest reward type changes:

- delete auto-generated point records for that contest
- regenerate them from current contest results
- recompute affected students' totals

### Contest Delete

Deleting a contest also deletes its auto-generated point records, then recomputes totals.

### Manual Entry

Admin can create manual point records for:

- progress awards
- organization awards

Manual records update student totals immediately.

## Frontend Changes

- Contest upload form: add reward type selector
- Contest edit modal: add reward type selector
- Student admin table: show `redeem_points`
- Optional later extension: point-record management page

## Non-Goals

- No score-based qualification-award automation
- No integration into `rating replay`
- No automatic progress-award calculation yet
