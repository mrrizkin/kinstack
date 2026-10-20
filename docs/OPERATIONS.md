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
| Audit retention | ≥ 1 year |

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
sign in again. This is accepted behavior.

## Alerts

| Alert | Condition | Action |
|---|---|---|
| HTTP 5xx | > 1% for 5 minutes | Check request logs, database, and storage |
| Latency | p95 > 1s for 10 minutes | Check database pool, slow queries, documents, and storage |
| Readiness | 3 consecutive failures | Check PostgreSQL and Valkey |
| Database pool | active connections = 10 | Look for slow queries or raise the pool |
| Login lockout | spike in locked accounts or 429 | Check brute-force sources |
| Media storage | storage unavailable errors | Check object storage and credentials |
| TLS certificate | < 14 days | Renew at the reverse proxy |

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
- SMTP, object storage, and database credentials: keep them in a secret manager.
- After rotation, test login, email, and media.

## Capacity

- PostgreSQL pool: 10 open, 5 idle.
- Email delivery and PDF rendering run synchronously in the request path.
- Audit records use the configured PostgreSQL retention policy.
- Media uploads are validated by magic bytes and image dimensions before storage.
