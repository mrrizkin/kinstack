# Renovate (Forgejo)

This repository uses **Renovate** as its only dependency updater (based on the
platform-agnostic `renovate.json`: GitHub, GitLab, Forgejo, or Gitea).
Dependabot has been removed so there are no duplicate PRs.

## Why Renovate

- Forgejo does not support Dependabot. Renovate supports the `forgejo` (and
  `gitea`) platform natively.
- One configuration for Go modules, Bun/npm, GitHub Actions, Dockerfile, and
  Docker Compose.
- PR groups, schedules, and exceptions (for example, TypeScript major versions)
  are managed in `renovate.json`.

## Running on Forgejo

Renovate runs as a separate process/container from the application. A minimal
example (self-hosted Forgejo):

```bash
docker run --rm \
  -e RENOVATE_PLATFORM=forgejo \
  -e RENOVATE_ENDPOINT=https://git.example.com/api/v1 \
  -e RENOVATE_TOKEN=glpat-xxxxxxxxxxxxxxxxxxxx \
  -e RENOVATE_REPOSITORIES=owner/kinstack \
  -e RENOVATE_GIT_AUTHOR="Renovate Bot <renovate@example.com>" \
  -e LOG_LEVEL=info \
  -v "$PWD/renovate-config.js:/usr/src/app/config.js:ro" \
  ghcr.io/renovatebot/renovate:latest
```

Notes:

- `RENOVATE_ENDPOINT` must point to the Forgejo **API**, not the web page:
  `https://<host>/api/v1` (no trailing slash).
- `RENOVATE_TOKEN` is a Forgejo access token with the `repo` scope (read/write)
  so Renovate can create branches and PRs.
- For multiple repositories, use `RENOVATE_REPOSITORIES` or a `config.js` with
  `repositories: [...]`.
- Keep the token in a secret manager or runner environment, not in the
  repository.

### Optional config.js

```js
module.exports = {
  platform: "forgejo",
  endpoint: "https://git.example.com/api/v1",
  gitAuthor: "Renovate Bot <renovate@example.com>",
  onboarding: false,
  requireConfig: "required",
  repositories: ["owner/kinstack"],
};
```

## Configuration behavior

| Manager | Scope | PR group |
|---|---|---|
| `gomod` | `go.mod` / `go.sum` | `go dependencies` |
| `npm` | `package.json`, `bun.lock` | `frontend dependencies` |
| `github-actions` | `.github/workflows/*` | `ci actions` |
| `dockerfile` / `docker-compose` | `Containerfile`, `deploy/compose.yml` | `container images` |

- Schedule: every Monday before 05:00 `Asia/Jakarta`.
- `rangeStrategy: bump` raises the minimum version instead of merely widening
  the range.
- **TypeScript major** upgrades are disabled because `vue-tsc` does not support
  TypeScript 7 yet. This rule lives in `renovate.json` and must be removed once
  `vue-tsc` is compatible.

## CI notes

Dependabot is no longer configured. CI currently uses GitHub Actions
(`.github/workflows`). Forgejo Actions reads the `.forgejo/workflows` directory;
if you fully migrate to Forgejo, copy the workflows to `.forgejo/workflows/` and
adjust the runner.
