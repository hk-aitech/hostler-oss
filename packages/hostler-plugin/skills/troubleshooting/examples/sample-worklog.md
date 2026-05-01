# Worklog: addAuthPolicy and dev-check.sh improvements

**Date**: 2026-01-03
**Status**: resolved
**Severity**: High
**Time spent**: ~2 hours

---

## Problem context

### Symptoms

A 401 Unauthorized error on the Backstage dev server when visiting the `/cicd` and `/health` pages.

```
Error: Request failed with 401 Unauthorized
  at handleResponse (catalog-client.esm.js:45:11)
```

In the browser console:
- `GET /api/cicd/dashboard/domains` → 401
- `GET /api/health/gitlab` → 401

### Reproduction

1. Start the dev server with `yarn dev`
2. Open `http://localhost:3000/cicd` in a browser
3. The page errors during load

### Environment

- OS: Ubuntu 22.04 (WSL2)
- Node.js: 22.x
- Backstage: 1.35+
- Backend: New Backend System

---

## Investigation

### 10:15 - Initial inspection

Suspected a Node.js version issue, checked the version:

```bash
node --version
# v22.12.0
```

Version was fine. Inspected the backend log:

```bash
grep "401\|unauthorized" /tmp/backstage-dev.log
```

Result: many auth-related errors.

### 10:25 - UI check via Playwright

Checked directly in the browser:

```bash
# Capture a screenshot of /cicd
```

Findings:
- The CICD page shows "Error loading data"
- The Network tab shows 401 responses

### 10:35 - Backend code analysis

**Hypothesis**: missing auth policy in the new backend system.

Inspected `plugins/cicd-backend/src/plugin.ts`:

```typescript
httpRouter.use(router);
// no addAuthPolicy call
```

**Verification**: Backstage docs confirmed
- The new backend authenticates every endpoint by default
- `addAuthPolicy` is needed to declare exceptions

### 10:45 - CatalogClient analysis

Inspected `DomainStatsService.ts`:

```typescript
this.catalogApi = new CatalogClient({
  discoveryApi: options.discovery,
  // no fetchApi!
});
```

Findings:
- Inter-service calls created the CatalogClient without an auth token
- Internal API calls also returned 401

### 10:55 - GitLab health-check analysis

Inspected `packages/backend/src/plugins/health.ts`:

```typescript
const response = await fetch(gitlabUrl, {
  headers: { 'Accept': 'application/json' },
  // no PRIVATE-TOKEN!
});
```

Findings:
- GitLab API calls were missing the token
- Self-hosted GitLab requires authentication

---

## Attempts

### Attempt 1: addAuthPolicy in cicd-backend

```typescript
// plugin.ts
httpRouter.use(router);
httpRouter.addAuthPolicy({
  path: '/',
  allow: 'unauthenticated',
});
```

Result: success — fixed the /cicd 401.

### Attempt 2: add CatalogClient fetchApi

```typescript
this.catalogApi = new CatalogClient({
  discoveryApi: options.discovery,
  fetchApi: {
    fetch: async (input, init) => {
      const { token } = await options.auth.getPluginRequestToken({
        onBehalfOf: await options.auth.getOwnServiceCredentials(),
        targetPluginId: 'catalog',
      });
      const headers = new Headers(init?.headers);
      headers.set('Authorization', `Bearer ${token}`);
      return fetch(input, { ...init, headers });
    },
  },
});
```

Result: success — internal Catalog calls work.

### Attempt 3: include the GitLab token (health.ts)

```typescript
const gitlabToken = gitlabInstances.length > 0
  ? gitlabInstances[0].getOptionalString('token')
  : undefined;

const headers: Record<string, string> = {
  'Accept': 'application/json',
};
if (gitlabToken) {
  headers['PRIVATE-TOKEN'] = gitlabToken;
}
```

Result: success — GitLab health check works.

### Attempt 4: addAuthPolicy with a wildcard

```typescript
httpRouter.addAuthPolicy({
  path: '/dashboard/*',
  allow: 'unauthenticated',
});
```

Result: failed — path-to-regexp error.

```
TypeError: Missing parameter name at index 12: /dashboard/*
```

Fix: declare paths explicitly or use `path: '/'`.

---

## Final resolution

### Root cause

1. The new Backstage backend system authenticates every endpoint by default
2. Inter-service calls must explicitly forward an auth token
3. External API (GitLab) calls must include their token

### Resolution

Modified three files:

1. **cicd-backend/plugin.ts**: add addAuthPolicy
2. **DomainStatsService.ts**: add an auth token to fetchApi
3. **health.ts**: add the GitLab token and an addAuthPolicy

### Verification

```bash
# Restart the dev server
yarn dev

# API test
curl -s http://localhost:7007/api/cicd/dashboard/domains | jq .
# normal response confirmed

curl -s http://localhost:7007/api/health/health | jq .
# GitLab status: "healthy" confirmed
```

In the browser, the `/cicd` and `/health` pages now load correctly.

---

## Additional findings

### tech-insights plugin error

Found during investigation:

```
Error: Service or extension point dependencies of plugin 'tech-insights'
are missing for the following ref(s): serviceRef{core.tokenManager}
```

Workaround: disable the tech-insights plugin

```typescript
// backend.add(import('@backstage/plugin-tech-insights-backend'));
```

→ Track separately (TD-OPS-01).

---

## Related commits

- `173d3fa`: add addAuthPolicy in cicd-backend
- `55aa7b3`: add an authenticated fetchApi in DomainStatsService
- `bb9f2d5`: add the GitLab token and auth policy in health.ts

---

## Lessons

1. **Understand the new Backstage backend system**: the default auth policy changed
2. **Auth in inter-service communication matters**: internal API calls also need a token
3. **External API token management**: read tokens from config and use them
4. **path-to-regexp limits**: be careful with wildcard patterns

---

## References

- [Backstage Backend System Docs](https://backstage.io/docs/backend-system/)
- [HTTP Router Service](https://backstage.io/docs/backend-system/core-services/http-router)

---

*Authored: 2026-01-03*
*Author: Claude Code*
