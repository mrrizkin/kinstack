# Operations & runbook

This document complements [`ARCHITECTURE.md`](./ARCHITECTURE.md) and
[`OBSERVABILITY.md`](./OBSERVABILITY.md). It contains the operational baseline
for a KinStack deployment: backup, alerting, deployment, rollback, and incident
response.

## Service targets

| Signal | Initial target |
|---|---|
| HTTP availability | 99.9% per month |
| Document request p95 latency | < 500 ms |
| API p95 latency | < 250 ms |
| 5xx error rate | < 0.5% |
| Readiness | 100% during the service window |
| PostgreSQL RPO | ≤ 6 hours |
| PostgreSQL RTO | ≤ 2 hours |
| Audit retention | ≥ 1 year (adjust to policy) |

These numbers are an internal baseline, not a contractual SLA.

## Backup and restore

### PostgreSQL

Take daily backups in custom format and encrypt them at the storage layer:

```bash
pg_dump --format=custom --no-owner --no-privileges "$DATABASE_URL" \
  | age -r "$BACKUP_AGE_RECIPIENT" > "kinstack-$(date -u +%Y%m%dT%H%M%SZ).dump.age"
```

Minimum rules:

- back up every 6 hours once application data starts arriving;
- keep 7 daily, 4 weekly, and 12 monthly backups;
- store backups separately from the database;
- test a restore at least once a month to a temporary database;
- do not rely only on container volume snapshots.

Restore:

```bash
createdb kinstack_restore
age -d -i "$BACKUP_AGE_IDENTITY" kinstack-....dump.age \
  | pg_restore --no-owner --no-privileges --dbname kinstack_restore
```

After restoring, run `just migrate-status` and smoke test login and the
`/dashboard` page.

### Object storage (RustFS/S3)

- Enable versioning and a lifecycle policy on the media bucket.
- Back up PostgreSQL metadata and object bytes consistently.
- Test restoring one object and one `media_assets` row before an incident.

### Valkey

Valkey stores sessions, lockouts, and temporary data. Do not make it the only
source of truth. Losing Valkey forces every user to sign in again; document that
as accepted behavior.

## Minimum alerts

| Alert | Initial condition | Action |
|---|---|---|
| HTTP 5xx | > 1% for 5 minutes | Check GlitchTip, `request_id` logs, DB/storage |
| Latency | p95 > 1s for 10 minutes | Check DB pool, slow queries, Gotenberg/S3 |
| Readiness | 3 consecutive probe failures | Check PostgreSQL and Valkey |
| DB pool | active connections = max 10 | Look for slow queries or increase the pool |
| Login lockout | spike in `locked`/429 | Check for brute force and source IPs |
| Media storage | `media storage unavailable` error | Check RustFS/S3 and credentials |
| CSP | new `csp violation report` entries | Check blocked script/style sources |
| Certificate/TLS | < 14 days | Renew the reverse proxy certificate |

Send alerts to the on-call channel. GlitchTip/OpenObserve are diagnosis tools,
not a replacement for alerting.

## Deployment

1. Run `just verify` on the commit being released.
2. Build the image and run migrations before or alongside the release:
   ```bash
   docker compose -f deploy/compose.yml run --rm migrate
   ```
3. Deploy one instance, check `/health` and `/ready`.
4. Continue the rollout gradually.
5. Monitor 5xx, latency, and media/email errors for at least 15 minutes.

Migrations must be backward compatible during the rollout. Avoid direct
`DROP COLUMN`; use the expand → migrate → contract pattern.

## Rollback

1. Stop the rollout.
2. Roll the image back to the previous tag.
3. Do not automatically run `goose down` in production. Only roll a migration
   back after analysis confirms the data is safe.
4. If data is corrupted, restore the database from the latest backup and then
   reconcile data written after that backup.

## Incident runbook

### Database unavailable

- Check `/ready`; `database: unavailable` confirms the condition.
- Check PostgreSQL connectivity, disk, and connection limits.
- Do not restart the application repeatedly without investigating the cause.
- After recovery, confirm the migration status and pool are back to normal.

### Valkey unavailable

- Sessions and lockouts cannot be read.
- Users are asked to sign in again; lockout fails open by design.
- Check memory, eviction, and connectivity.

### Email delivery failure

- Magic links, password resets, and invitations will fail.
- Check the SMTP relay, credentials, and Mailpit status (development only).
- Never print tokens in logs.

### Media failure

- Check the S3/RustFS endpoint, bucket, credentials, and the 12 MiB body limit.
- Media uploads are limited to 10 MiB and validated by magic bytes.

### CSP blocks assets

- Temporarily set `CSP_REPORT_ONLY=true` instead of disabling CSP.
- Read the `csp violation report` entries, then add origins through
  `CSP_EXTRA_SCRIPT_SRC`/`CSP_EXTRA_CONNECT_SRC`/`CSP_EXTRA_IMG_SRC`.
- Run `just smoke-csp` before returning to enforcing mode.

## Secret rotation

- Break-glass `ACCESS_KEY`/`SECRET_KEY`: rotate periodically and after any
  suspected leak. Use at least 32 random characters.
- `AUTH_STATE_SECRET`: rotate only when no OAuth login is in progress.
- SMTP, S3/RustFS, and database credentials: keep them in a secret manager, not
  in the repository.
- After rotation, smoke test login, email, and media.

## Capacity and future work

- The DB pool is currently 10 open / 5 idle; measure before increasing it.
- Email and PDF rendering are still called inline. Before volume grows, move them
  to a worker queue with retries and a dead-letter queue.
- Audit is not yet tamper-evident; consider a hash chain/WORM if compliance
  requirements appear.
- Uploads do not pass through antivirus/re-encoding yet; consider adding it for
  deployments that accept files from external users.
