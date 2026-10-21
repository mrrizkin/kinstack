# Style

KinStack is a base, not a finished application. The code is the reference
implementation; this page states the conventions a new slice must match. When a
rule has a working example, the example is the specification.

## Principles

**Simple is good.** Simplicity is the hardest revision, not the first draft.
Spend design time up front, prefer the simpler signature, and do not reach for
generics, reflection, extra interfaces, or dependencies until they earn their
place.

**Lean software.** Every dependency must save more complexity than it adds,
including its types, supply-chain risk, and upgrade cost. Prefer the standard
library; a little copying beats a little dependency. Keep the base tidy to
develop, iterate, and maintain, so the code stays readable years later without
extra documentation.

**Bound the system.** Every loop, queue, batch, read, and list has an explicit
limit. Input is validated at the boundary, errors are typed, and invariants are
asserted close to where they are assumed.

**Learnable by default.** Every action has visible, immediate feedback, mistakes
are cheap to recover, and depth is disclosed progressively. Consistency beats
novelty; silent confusion is a defect.

## Go

- Bound resources: request bodies, uploads, list pages, batches, retries.
  Examples: `internal/features/media/handler.go`,
  `internal/platform/audit/repository_pg.go`, `internal/pkg/pagination`.
- Reject invalid input instead of guessing. Page requests fail closed in
  `internal/features/users/service_users.go`; past-end pages clamp in
  `internal/pkg/pagination/pagination.go`.
- Expected failures are typed sentinel errors; unexpected ones are logged once
  with `request_id`. See `internal/features/users/errors.go`.
- Assert invariants that must never break, at startup or at the decision point.
  See `internal/pkg/assert` and `internal/platform/authz/authority.go`.
- Authorization rules live in the platform, not in handlers: rank, permission
  subset, self-change, and session requirements are enforced by `authz`.
- Guarded routes need a `routePolicy` entry; startup fails without one and the
  policy test covers it (`internal/bootstrap/route_policy_test.go`).
- Prefer small functions and centralize branching in the orchestrator. Around 70
  lines is a review signal, never a gate.
- Names carry units, ownership, and call history; acronyms stay uppercase; Go
  naming, not `snake_case`.
- Comments explain why and which constraint holds; the code explains what.

## Vue and TypeScript

- Composition API with `<script setup lang="ts">` only. `ref<T>()` everywhere;
  `reactive<T>()` is not used.
- Parse every trust boundary at runtime with zod, not just a TypeScript type:
  `resources/js/lib/http.ts` and `resources/js/lib/contracts`.
- No `any`; narrow `unknown`. Type assertions are allowed only where a parsed
  payload crosses the boundary, with the parse right there.
- Explicit dependencies only: `watch([...])` instead of multi-dependency
  `watchEffect`, props/events instead of `provide`/`inject` for business logic.
- Inertia layouts are declared by the page with
  `defineOptions({ layout: [BaseLayout, XLayout] })`; `resources/js/layouts/BaseLayout.vue`
  is the only `<UApp>` holder.
- Reuse existing components; wrap a Nuxt UI component only when the wrapper adds
  behavior or removes duplication. See `resources/js/components/AppPagination.vue`
  and `AppConfirmDialog.vue`.
- Around 300 lines per component is a review signal, never a gate.
- Copy goes through the existing i18n boundary, in English.

## Documentation

- English, professional, timeless: no status, roadmap, history, or agent
  instructions in `docs/`.
- Document invariants and operations only. Decisions, rationale, and task state
  live in agent memory, not in the repository.
- Documentation explains KinStack as a base for building applications.
- `AGENTS.md` holds agent instructions; humans read `README.md` and `docs/`.

## Gates

- `just verify` must pass before a change is complete.
- `just shape` reports long functions, large components, and casts. It informs
  review and never fails a build.
