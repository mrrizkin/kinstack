# Style

KinStack is a base. Applications built on it inherit the standards below; this
page is the contract for the code you add. The base already follows these
standards, so its own slices are the reference implementation to copy from.

## Principles

Design goals, in order. Apply them to the features you add, not just to code
that ships with the base.

**Simple is good.** Simplicity is the hardest revision, not the first draft.
Spend design time before writing, choose the simpler signature, and do not reach
for generics, reflection, decorative interfaces, or dependencies until they earn
their place.

**Lean software.** A dependency must remove more complexity than it adds,
including its types, supply-chain risk, and upgrade cost. Prefer the standard
library and a little copying over a package. Keep the application tidy enough to
iterate on and maintain for years without extra documentation.

**Bound the system.** Everything you accept, store, or return has an explicit
limit: request bodies, uploads, list pages, batches, retries, loops. Validate at
the boundary, use typed errors, and assert invariants where they are assumed.

**Learnable by default.** Every action shows immediate feedback, mistakes are
cheap to recover, and depth is revealed progressively. Consistency beats
novelty; a flow that silently confuses a first-time user is a defect.

## Validation boundaries

Validate where untrusted data enters, then trust the types inside your own code.
Do not re-check invariants that the boundary and the type system already
guarantee.

- Make illegal states unrepresentable (a union of states, not a free-form
  string) instead of adding runtime checks at every consumer.
- Assert an invariant once, where it is owned: the store, the service, the
  constructor. Copy-pasting the check into each caller is duplication, not
  safety.
- Parse instead of coercing. A malformed payload fails loudly at the boundary;
  silently falling back to "close enough" hides the bug until it is expensive.
- Security limits stack across layers (edge validation, parameterized queries,
  rate limits). That is independent defense, not a reason to duplicate internal
  validation.

## Adding a feature

- Put application behavior in `internal/features/<name>` as a slice. Keep the
  handler/service/repository boundary; `internal/features/users` is the most
  complete example to copy.
- Register every new permission code in `internal/platform/authz/catalog.go`.
- Register every guarded route in `routePolicy` in
  `internal/bootstrap/web_routes.go`. Startup fails without an entry.
- Parse transport data in the handler, decide in the service, persist through a
  repository interface owned by the slice. Handlers do not make decisions.
- Reuse `internal/platform` for pagination, audit, authorization, sessions, and
  documents instead of adding local variants.
- Add a Goose migration for schema changes; never edit an applied migration.
- Add the copy you need to the existing i18n boundary in the same change.

## Go

- Bound anything you read, list, batch, or retry. Reject invalid input instead
  of guessing; past-end pages clamp, malformed input fails closed.
- Return typed sentinel errors for expected failures. Log unexpected failures
  once, with `request_id`. Never return raw storage or database errors to a
  client.
- Assert invariants that must never break, at startup or at the decision point.
- Keep authorization rules in the platform layer (rank, permission subset,
  self-change, session requirements), never in a handler.
- Prefer small functions and centralize branching in the orchestrator. Around 70
  lines is a review signal, not a gate.
- Names carry units, ownership, and call history; acronyms stay uppercase; Go
  naming, never `snake_case`.
- Comments explain why and which constraint holds; the code explains what.

## Vue and TypeScript

- Composition API with `<script setup lang="ts">` only. Use `ref<T>()`;
  `reactive<T>()` is not part of the standard.
- Parse every trust boundary at runtime with zod, not just a TypeScript type.
  Inbound API payloads, storage reads, and route params are boundaries.
- No `any`; narrow `unknown`. Type assertions are limited to the place where a
  parsed payload crosses the boundary.
- Be explicit: `watch([...])` instead of multi-dependency `watchEffect`, and
  props/events instead of `provide`/`inject` for business logic.
- Declare Inertia layouts on the page with
  `defineOptions({ layout: [BaseLayout, XLayout] })`; `BaseLayout` is the only
  `<UApp>` holder.
- Reuse existing components. Wrap a Nuxt UI component only when the wrapper adds
  behavior or removes duplication.
- Around 300 lines per component is a review signal, not a gate.
- Route user-facing copy through the i18n boundary, in English.

## Documentation

- Write documentation in English, professionally, and timelessly. Document
  invariants and operations for the application you are building, not its
  status, roadmap, or history.
- Keep decisions, rationale, and task state out of the repository; they belong
  in agent memory.
- Agent instructions live in `AGENTS.md`, never in `docs/`.

## Gates

- `just verify` must pass before a change is complete.
- `just shape` reports long functions, large components, and casts to inform
  review. It never fails a build.
