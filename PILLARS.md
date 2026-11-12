# Engineering Pillars

The engineering standard for this repository and anything built on it. It
inherits **Tiger Style** (TigerBeetle) and adapts it to Go and Vue; where the two
disagree, the adaptation wins. TigerBeetle is a single-binary database where
crashing hard is a feature. KinStack is a web application where it is an outage.

## Two registers

Every rule carries exactly one label.

**[GATE]** — a command decides, in `just verify` or in CI; each gate names which.

**[RITUS]** — a human decides, in review. Every ritual names the question asked
and one real violation from this repository. A rule with no address is
decoration.

**Target** — no gate exists yet. Targets are debt, collected in
[Not yet gated](#not-yet-gated), never dressed up as standards.

Anything that prevents data loss or privilege escalation is a gate, never a
ritual.

## When pillars conflict

The list below is ordered. Higher wins.

1. **Integrity** — data that cannot be recovered, access that cannot be revoked.
2. **Safety** — predictable behavior; failure that is loud, not silent.
3. **Boundaries** — untrusted input, schema versions, secrets, least privilege.
4. **Determinism** — a bug that can be reproduced can be fixed.
5. **Observability** — an outage that can be explained without guessing.
6. **Performance** — designed, not patched.
7. **Developer experience** — readable, and cheap to change.
8. **Aesthetics** — naming, formatting, taste.

When two pillars fight, the higher one wins and the decision is written down in
the change that made it.

---

# Foundation: Simple is Good

Simplicity is the test every pillar below answers to. Every rule, feature, and
design decision passes one question: **does this make the system simpler?**

- Choose the simplest correct solution.
- Complexity is allowed, but it is argued in writing, in the change that
  introduces it.
- Simplicity is the hardest revision, not the first draft. Spend design time
  before writing.

`[RITUS]` *What did this change make simpler? If the answer is "nothing yet",
when will it?*
Real violation: `docs/STYLE.md` was a second copy of this standard. Two copies
kept in agreement are a cost paid for a reader who does not exist.

---

# 1. Safety

Systems must not behave in unexpected ways. A bug fails loudly, at the boundary
or at the assertion, never silently.

- **Explicit errors.** Every error is handled, wrapped with context, or returned.
  Typed sentinels for expected failures; unexpected failures are logged once and
  never leaked to a client.
- **Assert once, at the owner.** An invariant is asserted where it is owned — the
  store, the service, the constructor — not copy-pasted into every caller.
  Duplicated checks are duplication, not safety.
- **No unbounded behavior.** Loops, queues, and recursion all have a fixed upper
  bound. Recursion is permitted only when the bound is written next to it and
  small enough to prove by eye.
- **Explicit widths** where serialization or wire format depends on them. Go's
  `int` is not a wire contract; `u32` is not an idiom. Pick per boundary.
- **Make illegal states unrepresentable** before adding a runtime check.

`[GATE]` `just lint-go` (`go vet`, `staticcheck`), `just test-go`.

`[RITUS]` *Where is the invariant owned, and is it asserted there rather than in
each caller?*
Real violation: `internal/features/notifications/service.go` compared
`recipientPublicID == ""` in four separate methods, each free to drift into a
slightly different check.

# 2. Bounded and Lean

Everything accepted, stored, or returned has an explicit limit, and only what is
needed exists.

- **Bounds are numbers, not adjectives.** Request bodies, uploads, page sizes,
  batches, retries, queues, timeouts, memory. A limit nobody can state is not a
  limit.
- **Timeouts on every external call.** Database, HTTP, mail, storage, queue.
- **Idempotent by default** for anything retried or replayable.
- **Every dependency earns its place** by removing more complexity than it adds,
  including its types, supply-chain risk, and upgrade cost. Prefer the standard
  library and a little copying.
- **Delete what is unused.** A feature, component, or config key with no
  consumer is not optional; it is load.

`[GATE]` `just check-feature-slices` and `just check-modules` (`go mod tidy
-diff`: a dependency nothing imports is load) in verify; `bun audit` in CI. The
bounds themselves are not gated: no script can tell whether a limit is the right
number, only that one exists.

`[RITUS]` *What is the limit, where is it configured, and what happens at it?*
Real violation: `@maizzle/framework`, an email build tool with no runtime use,
sat in `dependencies` beside packages that ship to the browser, and a patched
transitive `decode-uri-component` was left to whoever ran `bun install` next.

# 3. Explicit State

Behavior that depends on "which stage we are in" is a state machine, not hidden
state.

- All states live in one enum. No combinations of flags standing in for a state
  (`is_loading && !has_error && data != null`).
- The transition table is written down: source state, event, destination state.
- One `transition` function is the only place state changes.
- Every state has an invariant asserted on entry. Illegal transitions are bugs
  and fail hard.
- Terminal states are marked as terminal.

`[GATE]` `internal/features/notifications/status_test.go` fails if a transition
leaves the table; `internal/features/notifications/simulation_test.go` asserts
that read is terminal across the whole run, under random operations and repeated
crashes. `Status` is a typed enum mirrored by the schema's `CHECK` constraint, and
`transitionStatus` is the only place the state changes. That is the shape the
next state machine must copy.

`[RITUS]` *Draw the states. Which one has no exit, and which one has two?*
The browser copy lives in `resources/js/lib/dialog.ts`: every modal and
slideover is `closed | editing | saving` (or `closed | confirming | running`),
with the transition table written above the one `transition` function. A
composable that adds a second `showX`/`savingX` boolean instead of a state has
re-created the bug, so the review question is: which state is terminal, and
what happens when a save is cancelled?

# 4. Determinism

Same state plus same events yields the same result.

- **Time, randomness, and I/O are injected**, not called directly from feature
  code. `time.Now()` in a service is a bug; a clock in the constructor is a seam.
- No reliance on iteration order that is not guaranteed. Sort before comparing.
- Every bug reproduces from a seed or a log.
- AI and other non-deterministic services are treated as external, bounded,
  timeout-carrying, failure-capable dependencies — never as pure functions.

The seam is a field, not a framework. A repository or service that needs the
current time takes `now func() time.Time` and defaults it in the constructor;
`internal/features/apitoken/repository_mem.go` is the reference. The memory
repositories that ship with each slice are the substrate a simulation drives,
so that seam is what makes a slice simulable at all.

The engine is `codeberg.org/mrrizkin/dstsim` (v0.4.1, no transitive
dependencies). Two runs exist and are the templates:

- `internal/features/apitoken/simulation_test.go` — seeded RNG, one simulated
  clock shared by service and repository, a lossy link in front of
  authentication, invariants checked after every operation.
- `internal/features/auth/lockout_simulation_test.go` — the guard against
  `fiber.Storage` whose TTL follows the simulated clock, and an independent
  model of the lockout promise as the oracle.

A failure reports the seed, the operation trace, and the shortest trace that
still fails.

One limit to state plainly: `context.WithTimeout` reads the real clock, so
deadline behavior cannot be simulated. A timeout path that must be simulated has
to accept a clock rather than derive a deadline from a context.

`[GATE]` `just check-determinism` greps `internal/features`,
`internal/platform`, `internal/pkg`, `internal/bootstrap`, `config` and `cmd` for
`time.Now`, `time.Since`, `time.Sleep`, `time.After`, `time.Tick` and
`math/rand`. A file not in `scripts/determinism-baseline.tsv` may not contain
one, and a baselined file may not grow. The baseline is empty. Production
defaults carry `//kinstack:allow-nondeterminism` with a reason:
`internal/platform/clock` supplies the system clock, the HTTP access log
measures latency, and the log/observability writers stamp their own time. Every
other clock is a parameter.

`[RITUS]` *Could this test fail on another machine, at another minute, in another
order?*
Real violation: eight memory repositories call `time.Now()` inside the
repository, while `apitoken` alone takes an injected `now`. Only one slice was
replayable, and nothing in the code said so.

# 5. Boundaries

Validate fully at the edge, trust types inside.

- **Parse, don't coerce.** Malformed input fails closed at the boundary;
  "close enough" hides the bug until it is expensive.
- Runtime parsing at every trust boundary: HTTP input, storage reads, route
  params, webhook payloads. A TypeScript type is not a parser; zod is.
- **Schemas are versioned.** Migrations are explicit, tested, and never edited
  after they are applied.
- **No secrets in code.** Least privilege everywhere; authorization rules live in
  the platform layer, never in a handler.
- **Contracts are written**: shape, limits, and the errors a component may
  return.
- Security limits stack across layers (edge validation, parameterized queries,
  rate limits). That is independent defense, not a reason to duplicate internal
  validation.
- The edge is the handler that receives bytes; the parse belongs to the service
  that owns what the bytes mean. A handler that parses a domain value has taken
  a decision §11 says belongs to the service. `c.Params("publicID")` travelling
  as a string to the service is correct for that reason, not by omission.

`[GATE]` `just verify` (types, tests, lint), `just check-secrets` (CI), and the
CI migration job, which applies every migration up before the suite runs.

`[RITUS]` *What is the untrusted edge here, and where exactly is it parsed?*
Real violation: `internal/features/auth/handler_oauth.go` continued the callback
when the PKCE verifier was missing, so a replayed state reached the provider with
an unprotected exchange.

# 6. Verification

Unit tests alone do not find bugs; bugs are hunted on purpose.

- Property-based tests for invariants that must hold for every input.
- Fuzzing on parsers and system edges.
- Simulation that randomizes event order and failures, deterministically, from a
  seed. Every failure prints the seed; replaying the seed reproduces the exact
  op sequence. This is what `dstsim` exists to do, and a simulation test that
  cannot be replayed from its seed is not a simulation.
- Every bug that escapes becomes a regression test. No exceptions.

`[GATE]` `just test-go` (runs `go test -race`), `just test-frontend`,
`just check-feature-slices`, `just check-secrets` (CI: gitleaks over the
committed tree), and `just fuzz` — the same target in CI with a 30-second
budget, so coverage-guided fuzzing cannot quietly stop happening.

Fault injection has three shapes, all deterministic:

- **Network** — `dstsim.Network` drops, duplicates, reorders, and loses replies
  in front of a call (`apitoken`, `auth`).
- **Process death** — `dstsim` crashes and restarts mid-run;
  `internal/features/notifications/simulation_test.go` asserts the feed is
  identical before and after, so the service holds no per-process state.
- **Storage corruption** —
  `TestPendingTwoFactorActorRejectsCorruptChallenge` feeds every truncation of a
  valid session value, arbitrary bytes, and an expired payload into
  `PendingTwoFactorActor`; none may yield an actor and none may panic.

Database torn writes and fsync are outside the process: durability there belongs
to Postgres, and pretending to simulate it inside the service would be a lie
about what is being tested.

Property suites use `testing/quick` and live next to the code they describe:
`internal/pkg/pagination/property_test.go` (windows tile the result set, pages
stay in bounds) and `internal/features/audit/filter_property_test.go` (any
accepted time is an absolute UTC instant). Coverage is reported, never required:
a number invites writing tests for the number.

`[RITUS]` *Which invariant in this change is only checked by a happy-path
example?*
Real violation: in `internal/features/apitoken`, flipping `result.RevokedAt !=
nil` to `false &&` passed the handler tests. The simulation caught it at
operation 10 of 300.

Rule of honesty: a pillar whose gate does not exist yet does not get to be
described in the past tense. See [Not yet gated](#not-yet-gated).

# 7. Observability

When it fails, the cause is knowable without guessing.

- Structured logs, `request_id` on every request, one log per unexpected failure.
- Metrics leave the process: request count and latency are exported over OTLP
  whenever telemetry is configured.
- Correlation IDs cross components. Exported trace IDs are attached to errors,
  not invented at the client.
- Error messages carry enough context to diagnose without a debugger.

`[GATE]` `internal/pkg/log/middleware_test.go` fails if either middleware line
loses its `request_id`; `internal/platform/observability/observability_test.go`
fails if telemetry stops exporting metrics. `just check-observability` fails if a
handler passes
`"request_id"` by hand (that is the binder's job) or makes a structured log call
that did not come from `log.ForRequest(logger, c.RequestID())`. Bootstrap logs —
startup, shutdown, background workers — have no request and are out of scope.

`[RITUS]` *Simulate this failing in production. Can you tell why from telemetry
alone?*
Real violation: the OAuth callback redirected to `/login` on five failure paths
without logging why, leaving a failed login to be guessed at from the outside.

# 8. Function Shape

This is a quality bar, not a line budget. **Line count is not a gate.** A linear
90-line function beats three 30-line functions that pass six parameters and
operate each other's state. Extraction does not remove complexity; it moves it to
the call site.

A function is too large when it is a **GOD function**, judged by shape, not size:

- **Name test.** It cannot be named without "and" / "then". `syncUserAndNotify`
  fails before you read the body. This is the same rule as naming: one subject,
  one action.
- **One level of abstraction.** `db.Query` sitting next to `strings.TrimSpace`
  in the same body means two levels were mixed. That is the real reason a
  function becomes unreadable.
- **Nesting depth ≤ 3.** Extraction does not reduce nesting, so this cannot be
  gamed.
- **Cognitive complexity over cyclomatic complexity.** Splitting a function
  leaves total cognitive load unchanged and adds context switches. Ceiling 15.
- **More than four parameters** is a signal that the function knows too much,
  not a violation in itself.

Line count survives only as a review signal: around 70 lines for a Go function,
around 300 for a component. `just shape` reports both and never fails a build.

`[GATE]` `just format-go-check` (gofumpt, goimports, golines at 100 columns),
`just format-check`, `just lint-check`, `just type-check`, `just check-icons`,
`just check-complexity` (`gocognit` at 15, `nestif` weight at 8).

`scripts/complexity-baseline.tsv` may name a function that predates the ceiling,
with its number. The file only ratchets down: a listed function may be deleted
once it drops under the ceiling, never raised, and a function that is not listed
may not cross 15 at all.

`[RITUS]` *Can you say what this function does in one sentence, without "and"?*
Real violation: `config.(AppConfig).Validate` reached cognitive complexity 31 in
one function that validated identity, limits, network, and production at once.

# 9. Naming

Names carry units, ownership, and call history. They are the cheapest
documentation and the only kind that cannot go stale.

- **Big-endian order**: the most significant part (domain, subject) first; the
  qualifier and the unit move to the back. `latencyMSMax`, not `maxLatencyMS`.
  Related names then sort into groups — `latencyMSMin`, `latencyMSMax` — and are
  searchable by prefix.
- Casing follows the language, not this document: Go initialisms stay uppercase,
  Vue and TypeScript use camelCase, database columns use snake_case. Where
  big-endian ordering fights the language convention, the language wins and the
  ordering survives.
- No abbreviations. No accents or decoration in identifiers.
- Paired names should read parallel and be the same length where possible
  (`source` / `target`). This is a technique, not a law: `input` / `output` and
  `read` / `write` are correct even when the lengths differ.

`[RITUS]` *Read the names in this diff with the comments deleted. Do they still
explain the system? Does any name put the qualifier before its subject, or bury
its unit in the middle?*

Real violation (fixed): `maxUploadSize`, `maxCSPReportBytes`, `maxLogLineBytes`,
and `maxDocumentBytes` named the qualifier first, so the unit sat in the middle
of the identifier and the family never sorted together. The family now reads
`uploadSizeMax`, `cspReportBytesMax`, `logLineBytesMax`, `documentBytesMax`,
`perPageMax`, and `pageHintMax`.

# 10. Show, Don't Tell

Code, names, and types replace comments. Examples replace explanation. A comment
exists only for *why* — a constraint that the code cannot express.

- No documentation that restates the code. No TODO explaining what the diff
  already says.
- Prefer deleting a comment over maintaining a stale one.
- Documentation states invariants and operations for the application you are
  building, not its status, roadmap, or history. Decisions and task state live
  in the tracker, not the repository.

`[RITUS]` *Delete every comment in this diff. What did you lose? If nothing, why
was it written?*
Real violation: `docs/STYLE.md` restated this document's foundation almost line
for line, so one rule had two homes and one of them went stale.

Note this document applies to itself: prose that restates a rule already stated
elsewhere is deleted, not merged.

# 11. Developer Experience

Readable beats clever; the next reader is not the author.

- Variables live in the smallest scope that works.
- Branching is centralized in the orchestrator; leaf functions stay small and
  single-purpose.
- Composition API with `<script setup lang="ts">` only. Use `ref<T>()`;
  `reactive<T>()` is not part of the standard.
- No `any`; narrow `unknown`. Assertions only where a parsed payload crosses a
  boundary.
- Be explicit: `watch([...])` instead of multi-dependency `watchEffect`; props
  and events instead of `provide`/`inject` for business logic.
- Reuse what exists. Do not create a second helper, component, or platform
  primitive next to one that already does the job.
- New behavior goes in `internal/features/<name>` as a slice, following the
  handler/service/repository boundary. Handlers do not decide; services do;
  repositories persist through interfaces they own. Shared infrastructure goes
  in `internal/platform`.
- Register every permission code in `internal/platform/authz/catalog.go` and
  every guarded route in `routePolicy`. Startup fails without the route entry.
- Icons are semantic keys from `resources/js/lib/icons.ts` (`icons.delete`),
  never glyph names. Key by intent, not by glyph: `icons.access` may change
  shape, `icons.lock` may not. The registry is referenced from `ui.icons` in
  `vite.config.ts`.
- Server messages go through `internal/platform/i18n`; client shell copy lives
  in `resources/js/i18n`; page copy is written in English.

Frontend specifics:

- Declare Inertia layouts on the page with
  `defineOptions({ layout: [BaseLayout, XLayout] })`; `BaseLayout` is the only
  `<UApp>` holder.
- Wrap a Nuxt UI component only when the wrapper adds behavior or removes
  duplication.
- Render table cells through cell components, not `resolveComponent()`: Nuxt UI
  tags are auto-imported only inside `.vue` files. A `columns.ts` module maps a
  row to cell props and never builds markup.
- `components/` never imports `@/pages/`. Shared cells take flat structural
  props, not feature DTOs, and stay usable from `h(Cell, props)`.
- Render timestamps through the shared time cell so every surface keeps the
  machine-readable `<time datetime>` attribute.
- Send JSON through `requestJson` in `resources/js/lib/http.ts`, which uses
  Inertia's HTTP client. Do not call `fetch` directly.

`[GATE]` `just verify`, `just check-feature-slices`, `just check-icons`.

`[RITUS]` *What existing thing did this change fail to reuse?*
Real violation: `shrink` in the pagination property test and `bounded` in the
audit property test were the same function, so a fix to one would leave the other
wrong.

---

# The Review Ritual

Rituals are enforced by repetition at a fixed moment, not by a pass/fail. The
moment is the pull request.

Answer in prose, not checkboxes. The questions are the `[RITUS]` lines above —
one per pillar, asked where the rule is stated, so nothing is repeated here.

Two questions are not tied to a pillar:

- On the destructive path, does it say what it destroys?
- Count the steps and decisions from arrival to first success.

# Deviations

A pillar may be broken deliberately. The cost is writing it down.

```
Deviation: <pillar>
Reason: <the constraint that makes the rule wrong here>
Exit: <what would make the rule right again, or when this is revisited>
```

- Recorded in the change that introduces it, not in a wiki.
- Anything touching pillars 1–3 needs a second reviewer.
- An open-ended deviation is not a deviation; it is a missing revision.

Deviation: §1 Safety (§5 Boundaries)
Reason: `github.com/crewjam/saml` GO-2026-6909 reports two defects with no
patched release in any published version (v0.6.3 is the newest and still
vulnerable): a nil-pointer panic in `validateAssertion`, and a namespace-blind
`Subject.NameID`. The panic is reachable from an unauthenticated SAML response,
so `parseResponseSafely` recovers it into `ErrInvalidResponse` and the login is
rejected instead of the process ending. The namespace-blind subject cannot be
corrected from outside the library: the parser discards the namespace before the
value reaches this repository, so no check here can tell a real assertion from
one that renamed its element. Exposure is bounded by SAML being an opt-in
provider configured by an administrator, not a self-service signup path.
Exit: upgrade when a fixed release ships — remove the `-exclude` from `just
audit` in the same change, and delete this block. Revisited every release, since
the scan fails loudly the moment the advisory gains a fix.

# Not yet gated

A rule this document states but cannot yet enforce belongs here, named as a
target in the order it earns its cost. An empty list is the goal.

Nothing. Every rule in this document names a command that decides it, or a
review question that a human answers — and each of those questions carries a
violation that actually happened in this repository.

Slices deliberately not simulated, with the reason:

| Slice | Why not |
|---|---|
| `media` | The interesting failure is the object store, which the network model already covers in front of the call; the slice itself is a pass-through. |
| `audit` | Append-only. There is no state to get wrong. |
| `passkeys` | The state that matters lives in the browser authenticator. A simulation without a real WebAuthn ceremony would test the mock. |
| `profile`, `system` | Read and render paths. The property suites cover their parsing; there is no transition to replay. |
| `users` | The risky part is the authorization boundary (who may grant which role), tested directly in `platform/authz` and the users handler tests. A simulation would spend most of its code building a fixture for a single guard. |

# Revision

- Review this standard quarterly. Delete every rule that has not been used. A
  rule nobody has cited is a rule nobody needed.
- A ritual rule with no example of a real violation does not get to stay.
- A backlash rule — one added after an incident — keeps the incident reference,
  so the next reader can judge whether it still applies.
- Automate anything that can be automated, then move it out of the ritual list.

# Appendix A: interface and UX notes

Working notes. Not gates, not rules, no verification. They live here because no
real violation in this repository could be cited for them, and a ritual rule
without one does not get to stay in the main body.

The interface rules, when designing a flow:

- Errors say what is wrong and what to do next. Both, always.
- The default is the safest choice; a destructive action states what is lost.
- One primary action per surface; the rest recede.
- Follow convention: `Ctrl+C` cancels, `-h` shows help.
- Few options up front, depth on demand.
- Progress is visible, and interrupted work can be resumed.
- No dark patterns: no hidden opt-outs, no false urgency, no friction invented
  to extract a concession.

The psychology catalogue:

| Note | Use | Example |
|---|---|---|
| Smart default | Most users never change it | `timeout_ms = 5_000`; CLI runs with no flags |
| Goal gradient | Show progress, start easy | `Step 2 of 3`; first run succeeds with one command |
| Reciprocity | Give value before asking | Show the result before asking for an account |
| Endowment / IKEA | Let the user shape the outcome | Pre-filled template, project named by the user |
| Loss aversion | Name what a destructive action destroys | `Delete 42 files? This cannot be undone.` |
| Contrast | One option stands out | One primary button, the rest secondary |
| Hick's law | Few options up front | Common commands in `--help`, the rest in `--help-all` |
| Jakob's law | Follow what is already known | `Ctrl+C` cancels; `-h` shows help |
| Peak-end | Design the success moment | `Done. 3 files created.` |
| Zeigarnik | Preserve interrupted work | `Resume from step 3?` |
| Cognitive load | One decision per step | One question per screen |
| Fitts's law | Frequent actions are big, near, short | Primary button in thumb reach; `ls`, not `list-directory-contents` |

Reciprocity is the one to watch. "Show value before asking for an account" is one
edit away from a dark pattern; if the value is fabricated to create obligation,
it is misuse, not a note.
