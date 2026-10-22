# Operations

## Service targets

| Signal | Target |
|---|---|
| HTTP availability | 99.9% per month |
| Document request p95 | < 500 ms |
| API p95 | < 250 ms |
| 5xx error rate | < 0.5% |
| PostgreSQL RPO | ≤ 6 hours |
| PostgreSQL RTO | ≤ 2 hours |
| Audit retention | ≥ 1 year (enforced by `prune-audit`) |

These are internal targets, not a contractual SLA.

## Backup and restore

### PostgreSQL

Back up daily in custom format and encrypt at the storage layer:

```bash
pg_dump --format=custom --no-owner --no-privileges "$DATABASE_URL" \
  | age -r "$BACKUP_AGE_RECIPIENT" > "kinstack-$(date -u +%Y%m%dT%H%M%SZ).dump.age"
```

Operational rules:

- back up every 6 hours when application data changes frequently;
- keep 7 daily, 4 weekly, and 12 monthly backups;
- store backups separately from the database;
- test a restore monthly;
- do not rely only on container volume snapshots.

Restore:

```bash
createdb kinstack_restore
age -d -i "$BACKUP_AGE_IDENTITY" kinstack-....dump.age \
  | pg_restore --no-owner --no-privileges --dbname kinstack_restore
```

After restoring, run `just migrate-status` and test login and `/dashboard`.

### Object storage

- Enable versioning and a lifecycle policy on the media bucket.
- Back up PostgreSQL metadata and object bytes consistently.
- Test restoring one object and one `media_assets` row.

### Valkey

Valkey stores sessions, lockouts, and temporary state. Losing it forces users to
sign in again.

## Migration rollback test

`TestMigrationLatestRollbackRoundTrip` applies and reverses every migration. It
is skipped unless `KINSTACK_MIGRATION_ROLLBACK_DATABASE_URL` points at a
disposable database:

```bash
PGPASSWORD=kinstack createdb -h localhost -p 5433 -U kinstack kinstack_rollback
export KINSTACK_MIGRATION_ROLLBACK_DATABASE_URL="postgres://kinstack:kinstack@localhost:5433/kinstack_rollback?sslmode=disable"
just test-go
```

The test drops and recreates the schema; never point it at data you want to keep.

## Observability

Error tracking, analytics, and server telemetry are opt-in. Each integration
stays disabled until its environment values are set, so a core-only environment
needs no extra services. `just infra-up` starts the optional local profile;
local ports and defaults live in `deploy/compose.yml` and `.env.example`.

## Alerts

| Alert | Condition | Action |
|---|---|---|
| HTTP 5xx | > 1% for 5 minutes | Check request logs, database, and storage |
| Latency | p95 > 1s for 10 minutes | Check database pool, slow queries, documents, and storage |
| Readiness | 3 consecutive failures | Check PostgreSQL and Valkey |
| Database pool | active connections = `DB_MAX_CONNS` | Look for slow queries or raise the pool |
| Login lockout | spike in locked accounts or 429 | Check brute-force sources |
| Media storage | storage unavailable errors | Check object storage and credentials |
| TLS certificate | < 14 days | Renew at the reverse proxy |

## Reverse proxy and client address

Rate limiting, the auth lockout, CSP reports, and audit records key on the
client address. Set `TRUSTED_PROXIES` to the proxy addresses only; every other
peer is treated as the client.

- A proxy on the same host or private network needs no configuration.
- A load balancer or CDN that answers from a public address must be listed
  (`TRUSTED_PROXIES=203.0.113.10,198.51.100.0/24`), otherwise all requests look
  like they come from the balancer: one attacker can exhaust the shared
  rate-limit bucket and lock out every account.
- Never list `0.0.0.0/0`: any client could then forge its own address.
- Behind TLS termination, the proxy must send `X-Forwarded-Proto: https` so
  cookies stay `Secure` and HSTS is emitted.

## Deployment

1. Run `just verify` on the release commit.
2. Build the image.
3. Run migrations before or alongside the release.
4. Deploy one instance and check `/health` and `/ready`.
5. Continue the rollout.
6. Monitor errors, latency, and storage for at least 15 minutes.

Migrations must be backward compatible during rollout. Use the
expand → migrate → contract pattern.

## Rollback

1. Stop the rollout.
2. Roll the image back to the previous tag.
3. Do not run `goose down` automatically in production.
4. Restore from backup only after confirming data corruption.

## Incident response

### Database unavailable

- `/ready` reports `database: unavailable`.
- Check connectivity, disk, and connection limits.
- Do not restart the application repeatedly without a cause.
- After recovery, confirm migration status and pool health.

### Valkey unavailable

- Sessions and lockouts cannot be read.
- Users must sign in again.
- Check memory, eviction, and connectivity.

### Email delivery failure

- Magic links, password resets, and invitations fail.
- Check the SMTP relay and credentials.
- Never print tokens in logs.

### Media failure

- Check the object storage endpoint, bucket, credentials, and body limit.
- Media uploads are limited to 10 MiB and validated before storage.

## Secret rotation

- Generate local secrets with `just setup-secrets` (idempotent; it only
  replaces empty or example values and keeps `DATABASE_URL` in sync with
  `POSTGRES_PASSWORD`).
- The credential defaults in `.env.example` are development-only. Startup
  validation rejects them when `APP_ENV=production`, so a copied example file
  cannot go live with published credentials.
- `ACCESS_KEY`/`SECRET_KEY`: rotate periodically and after any suspected leak.
  Use at least 32 random characters.
- `AUTH_STATE_SECRET`: rotate during a maintenance window.
- `AUTH_2FA_KEY`: envelope key for TOTP secrets at rest. When unset the
  application derives it from `SECRET_KEY`, so rotating `SECRET_KEY` also
  invalidates stored TOTP secrets (users must re-enroll). Set a dedicated
  32+ byte `AUTH_2FA_KEY` to rotate the two independently, and keep it with the
  other production secrets.
- After rotating the two-factor key, run `kinstack backfill-2fa` only if rows
  still hold plaintext secrets; sealed rows cannot be re-encrypted by that
  command and require re-enrollment.
- SMTP, object storage, and database credentials: keep them in a secret manager.
- After rotation, test login, email, and media.

## Authentication

- TOTP secrets are sealed with AES-256-GCM under a key derived from
  `AUTH_2FA_KEY` (or `SECRET_KEY` when unset) and never logged.
- `kinstack backfill-2fa` seals remaining plaintext secrets (idempotent,
  batched by 100).
- Recovery codes are stored as Argon2id hashes.
- A TOTP code is accepted once per timestep; replaying a code inside its
  validity window is rejected.
- Passkeys require user verification (`VerificationRequired`); enroll from the
  profile page after signing in with password and a second factor.
- OAuth and SAML accounts are bound to `(provider, subject)`. The SAML IdP must
  assert a persistent NameID; transient identifiers are rejected.

## Capacity

- PostgreSQL pool: `DB_MAX_CONNS` open (default 10), `DB_MAX_IDLE_CONNS` idle
  (default 5). Size the sum of all replicas below the server
  `max_connections` budget.
- Email delivery and PDF rendering run synchronously in the request path.
- Audit records are append-only and kept indefinitely by default. A scheduled
  run of `kinstack prune-audit 8760h` enforces the one-year target; schedule it
  from cron or a Kubernetes CronJob.
- Media uploads are validated by magic bytes and image dimensions before storage.
