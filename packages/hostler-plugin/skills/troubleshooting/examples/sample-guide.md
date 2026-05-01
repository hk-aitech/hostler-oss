# Backend authentication troubleshooting

**Status**: complete
**Created**: 2026-01-03
**Last updated**: 2026-01-03
**Related commits**: `173d3fa`, `55aa7b3`, `bb9f2d5`

---

## 1. Overview

Captures authentication-related issues that surface in Backstage's new backend system and their solutions.

### 1.1 Versions

- Backstage: 1.35+
- Backend System: New Backend (`@backstage/backend-defaults`)
- Node.js: 22.x

### 1.2 Related technologies

- HTTP Router Service
- Auth Service
- CatalogClient
- GitLab Integration

---

## 2. 401 Unauthorized error

### 2.1 Symptoms

```
Error: Request failed with 401 Unauthorized
AuthenticationError: Missing credentials
```

- 401 response when calling API endpoints
- Frontend pages fail to load data
- Inter-service internal calls fail

### 2.2 Cause analysis

#### Cause 1: no Auth Policy

In the new backend system every HTTP endpoint requires authentication by default.

```typescript
// Default behavior: every request requires auth
httpRouter.use(router);
```

#### Cause 2: missing CatalogClient auth

Inter-service calls created the CatalogClient without an auth token:

```typescript
// Wrong example
this.catalogApi = new CatalogClient({
  discoveryApi: options.discovery,
  // fetchApi missing - calls have no auth token
});
```

#### Cause 3: missing external-API token

Calls to external APIs (e.g. GitLab) didn't include the auth header:

```typescript
// Wrong example
const response = await fetch(url, {
  headers: { 'Accept': 'application/json' },
  // PRIVATE-TOKEN missing
});
```

### 2.3 Solutions

#### Solution 1: addAuthPolicy

Add an unauthenticated exception policy for public endpoints:

```typescript
// In the plugin's init
httpRouter.use(router);

// Public endpoint policy
httpRouter.addAuthPolicy({
  path: '/health',
  allow: 'unauthenticated',
});

httpRouter.addAuthPolicy({
  path: '/dashboard/domains',
  allow: 'unauthenticated',
});
```

**Caveats:**
- `path: '/dashboard/*'` triggers a path-to-regexp error
- Specify each path explicitly or use `path: '/dashboard/:endpoint'`
- `path: '/'` allows the entire plugin (use with care)

#### Solution 2: add an auth-aware fetchApi to CatalogClient

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

#### Solution 3: include the external-API token

```typescript
const gitlabToken = config.getOptionalString('integrations.gitlab[0].token');

const headers: Record<string, string> = {
  'Accept': 'application/json',
};
if (gitlabToken) {
  headers['PRIVATE-TOKEN'] = gitlabToken;
}

const response = await fetch(url, { headers });
```

---

## 3. Missing core.tokenManager service

### 3.1 Symptoms

```
Error: Service or extension point dependencies of plugin 'tech-insights'
are missing for the following ref(s): serviceRef{core.tokenManager}
```

- Plugin startup fails
- The whole backend fails to start

### 3.2 Cause

- `@backstage/plugin-tech-insights-backend` requires the legacy `core.tokenManager` service
- The new backend system doesn't provide it by default

### 3.3 Workaround (disable the plugin)

```typescript
// packages/backend/src/index.ts

// Tech Insights plugin for project standards compliance
// FIXME: Disabled due to core.tokenManager service missing error
// Need to update to new backend system API or add tokenManager service
// backend.add(import('@backstage/plugin-tech-insights-backend'));
// backend.add(import('./plugins/techInsights'));
```

### 3.4 Permanent fixes

1. **Wait for the upstream update**: until the official plugin migrates to the new backend API
2. **Implement the tokenManager service**: register `ServerTokenManager` from `@backstage/backend-common` as a service
3. **Custom plugin wrapper**: build a custom plugin that wraps tech-insights

---

## 4. Related files

### 4.1 Modified files

| File | Change |
|------|--------|
| `plugins/cicd-backend/src/plugin.ts` | Add addAuthPolicy |
| `plugins/cicd-backend/src/services/DomainStatsService.ts` | Add CatalogClient auth |
| `packages/backend/src/plugins/health.ts` | Use GitLab token, add addAuthPolicy |
| `packages/backend/src/index.ts` | Disable tech-insights |

### 4.2 Configuration

The GitLab token comes from `app-config.yaml`:

```yaml
integrations:
  gitlab:
    - host: gitlab.example.com
      token: ${GITLAB_TOKEN}
```

---

## 5. Debugging guide

### 5.1 Log inspection

```bash
# Inspect the backend log for auth errors
grep -iE "(401|unauthorized|auth)" /tmp/backstage-dev.log
```

### 5.2 Direct API tests

```bash
# Public endpoint test
curl -s http://localhost:7007/api/health/health | jq .

# Authenticated endpoint test
curl -s http://localhost:7007/api/cicd/dashboard/domains | jq .
```

### 5.3 Plugin initialization check

```bash
# Inspect plugin initialization logs
grep "Plugin initialization" /tmp/backstage-dev.log
```

---

## 6. Checklist

Auth-related checklist when developing a new plugin:

- [ ] Public endpoints have `addAuthPolicy`
- [ ] Inter-service calls use an authenticated `fetchApi`
- [ ] External API calls include the token
- [ ] Legacy service dependencies (e.g. tokenManager) checked

---

## 7. Related docs

- [Backstage Backend System](https://backstage.io/docs/backend-system/)
- [HTTP Router Service](https://backstage.io/docs/backend-system/core-services/http-router)
- [Auth Service](https://backstage.io/docs/backend-system/core-services/auth)

---

*Created: 2026-01-03*
*Author: Claude Code*
