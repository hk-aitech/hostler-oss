# Troubleshooting checklists

> Per-situation problem-solving checklists

---

## Table of contents

1. [Immediate-incident checklist](#immediate-incident-checklist)
2. [Investigation checklist](#investigation-checklist)
3. [Resolution checklist](#resolution-checklist)
4. [Closure checklist](#closure-checklist)
5. [Per-technology checklist](#per-technology-checklist)
6. [Per-severity checklist](#per-severity-checklist)

---

## Immediate-incident checklist

What to confirm right after an incident:

### Information gathering

- [ ] Copy the entire error message (screenshot or text)
- [ ] Record the time of the error
- [ ] Record reproduction steps
- [ ] Map the impact (who/what is affected)

### Environment info

- [ ] Confirm OS and version
- [ ] Confirm runtime version (Node.js, Python, etc.)
- [ ] Confirm framework / library versions
- [ ] Confirm configuration files

### Change history

- [ ] Inspect recent code changes (`git log -5`)
- [ ] Inspect recent configuration changes
- [ ] Inspect recent deployments
- [ ] Inspect dependency updates

### Classification

- [ ] Decide severity (Critical/High/Medium/Low)
- [ ] Decide priority (P0/P1/P2/P3)
- [ ] Decide whether to escalate

---

## Investigation checklist

### Log analysis

- [ ] Inspect application logs
- [ ] Inspect system logs
- [ ] Inspect logs around the error time
- [ ] Inspect logs of related services

### Hypothesis formation

- [ ] List possible causes
- [ ] Rank likelihood of each cause
- [ ] Order them so the most testable hypothesis goes first

### Hypothesis validation

- [ ] Validate the most likely hypothesis first
- [ ] Record validation results
- [ ] Record ruled-out hypotheses
- [ ] Revise hypotheses based on new information

### Isolation testing

- [ ] Confirm whether the problem only occurs in a specific environment
- [ ] Confirm whether it only occurs with specific input
- [ ] Confirm whether it only occurs at specific times
- [ ] Derive the minimum reproduction case

---

## Resolution checklist

### Pick a solution

- [ ] Decide between workaround and root-cause fix
- [ ] Assess side effects
- [ ] Assess risk
- [ ] Confirm whether rollback is possible

### Verify in test environment

- [ ] Apply the fix in the test environment
- [ ] Confirm the original problem is solved
- [ ] Confirm related features still work
- [ ] Confirm no side effects

### Prepare for application

- [ ] Prepare a rollback plan
- [ ] Document rollback commands
- [ ] Define rollback trigger conditions
- [ ] Define monitoring items

### Production application

- [ ] Apply the change
- [ ] Monitor immediately
- [ ] Confirm the problem is resolved
- [ ] Confirm no new errors in logs

### Post-application validation

- [ ] Re-check after 15 minutes
- [ ] Re-check after 1 hour
- [ ] Confirm related metrics are within normal range
- [ ] Confirm user feedback (when applicable)

---

## Closure checklist

### Documentation decisions

- [ ] Could it recur? → write a topic guide
- [ ] Was the investigation complex? → write a worklog
- [ ] Is this useful to others? → document it

### Documenting

- [ ] Capture symptoms accurately
- [ ] Capture cause analysis
- [ ] Capture the resolution
- [ ] Capture preventive measures

### Retro

- [ ] What did we learn?
- [ ] How can we prevent it?
- [ ] Does the process need to change?
- [ ] Do tools need improvement?

### Knowledge sharing

- [ ] Update index.md
- [ ] Update related guides
- [ ] Share with the team (when needed)

---

## Per-technology checklist

### Node.js issues

- [ ] Confirm Node.js version (`node --version`)
- [ ] Confirm npm/yarn version
- [ ] Confirm node_modules state
- [ ] Confirm package-lock.json sync
- [ ] Confirm native-module compatibility

```bash
# Version check
node --version
npm --version

# Clean reinstall
rm -rf node_modules
npm install

# Clear cache
npm cache clean --force
```

### Docker issues

- [ ] Confirm the Docker daemon is running
- [ ] Confirm the image exists
- [ ] Inspect container logs
- [ ] Confirm volume mounts
- [ ] Confirm network configuration

```bash
# Docker status
docker ps
docker images

# Container logs
docker logs {container_id}

# Network check
docker network ls
```

### Database issues

- [ ] Confirm the DB server is running
- [ ] Confirm the connection string
- [ ] Confirm credentials
- [ ] Confirm query performance
- [ ] Confirm index state

```bash
# PostgreSQL status
pg_isready -h localhost -p 5432

# Connection test
psql -h localhost -U username -d database -c "SELECT 1"
```

### API issues

- [ ] Confirm the endpoint URL
- [ ] Confirm the HTTP method
- [ ] Confirm headers (especially auth)
- [ ] Confirm the request body format
- [ ] Confirm the response status code

```bash
# API test
curl -v -X GET http://localhost:3000/api/endpoint

# With auth
curl -v -H "Authorization: Bearer {token}" http://localhost:3000/api/endpoint
```

### Auth issues

- [ ] Confirm token validity
- [ ] Confirm token expiration
- [ ] Confirm auth header format
- [ ] Confirm role / permission
- [ ] Confirm CORS configuration

### Build issues

- [ ] Confirm the build command
- [ ] Confirm environment variables
- [ ] Confirm dependencies are installed
- [ ] Confirm TypeScript compile errors
- [ ] Confirm the build output path

```bash
# TypeScript errors
npx tsc --noEmit

# Run build
npm run build

# Inspect build artifacts
ls -la dist/
```

---

## Per-severity checklist

### Critical (P0)

Requires immediate response:

- [ ] Start investigation immediately
- [ ] Notify all owners
- [ ] Share progress every 15 minutes
- [ ] Apply a workaround first
- [ ] Prepare rollback
- [ ] Document immediately after resolution

### High (P1)

Requires same-day response:

- [ ] Start investigation within 1 hour
- [ ] Notify stakeholders
- [ ] Share progress every 2 hours
- [ ] Prefer the root-cause fix; fall back to workaround if needed
- [ ] Document within 24 hours

### Medium (P2)

Resolve within this Sprint:

- [ ] Start investigation within 24 hours
- [ ] Record daily progress
- [ ] Resolve at the planned time
- [ ] Document after resolution

### Low (P3)

Backlog management:

- [ ] File an issue
- [ ] Plan according to priority
- [ ] Bundle with other work
- [ ] Document on resolution (optional)

---

## Quick reference checklists

### 5-minute check (immediately after incident)

1. [ ] Capture error messages
2. [ ] Classify severity
3. [ ] Decide whether to escalate
4. [ ] Begin investigation

### 15-minute check (initial investigation)

1. [ ] Log inspection complete
2. [ ] Recent-change check complete
3. [ ] At least one hypothesis formed
4. [ ] Progress recorded

### 30-minute check (in-flight investigation)

1. [ ] Validating hypotheses
2. [ ] Re-evaluate escalation
3. [ ] Estimate time to resolution
4. [ ] Consider workarounds

### Resolution check

1. [ ] Confirm problem resolved
2. [ ] Confirm no side effects
3. [ ] Documentation complete
4. [ ] Knowledge base updated
