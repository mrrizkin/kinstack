# Renovate

KinStack uses Renovate for dependency updates through `renovate.json`. The
configuration is platform-agnostic and supports GitHub, GitLab, Forgejo, and
Gitea.

## Coverage

| Manager | Scope | Group |
|---|---|---|
| `gomod` | `go.mod`, `go.sum` | `go dependencies` |
| `npm` | `package.json`, `bun.lock` | `frontend dependencies` |
| `github-actions` | `.github/workflows/*` | `ci actions` |
| `dockerfile` / `docker-compose` | `Containerfile`, `deploy/compose.yml` | `container images` |

The schedule is Monday before 05:00 `Asia/Jakarta`. `rangeStrategy: bump` raises
minimum versions instead of only widening ranges. TypeScript major upgrades are
disabled because `vue-tsc` requires the current TypeScript line.

## Forgejo

Renovate runs outside the application as a separate process or container.

```bash
docker run --rm \
  -e RENOVATE_PLATFORM=forgejo \
  -e RENOVATE_ENDPOINT=https://git.example.com/api/v1 \
  -e RENOVATE_TOKEN=<token> \
  -e RENOVATE_REPOSITORIES=owner/kinstack \
  -e RENOVATE_GIT_AUTHOR="Renovate Bot <renovate@example.com>" \
  -e LOG_LEVEL=info \
  -v "$PWD/renovate-config.js:/usr/src/app/config.js:ro" \
  ghcr.io/renovatebot/renovate:latest
```

- `RENOVATE_ENDPOINT` points to the Forgejo API, not the web UI, and has no
  trailing slash.
- `RENOVATE_TOKEN` needs the `repo` scope.
- Keep the token in a secret manager or runner environment.

Optional `config.js`:

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

## CI

The repository uses GitHub Actions. Forgejo Actions reads `.forgejo/workflows`;
copy the workflows there and adjust the runners when migrating.
