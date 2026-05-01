# Task Agent Guide

## Agent Assignment Criteria

| Agent | Specialty | Example Tasks | Tools |
|----------|----------|----------|------|
| **developer** | Feature implementation, bug fixes | API implementation, service logic | Read, Write, Edit, Bash, Glob, Grep |
| **devops** | Infrastructure, CI/CD | Docker, GitHub Actions | Read, Write, Edit, Bash, Glob, Grep |
| **architect** | Design, architecture | ADRs, architecture documents | Read, Write, Edit, Glob, Grep |
| **qa-engineer** | Testing, quality verification | Integration tests, E2E | Read, Write, Edit, Bash, Glob, Grep |
| **tech-writer** | Documentation | README, API docs | Read, Write, Edit, Glob, Grep |

---

## How Agents Are Invoked

When a Task is started (`/hstl-oss:task:start`), invoke the agent via:

```
1. Identify the assigned agent from the Task file
   │
   ▼
2. Invoke the agent via the Task tool
   - subagent_type: {assigned agent}
   - prompt: Task requirements + completion criteria
   │
   ▼
3. The agent does the work
   │
   ▼
4. Review the output and complete the Task
```

---

## Specifying the Agent in the Task File

```markdown
## Metadata

| Item | Value |
|------|-----|
| Assigned agent | developer |  ← required
```

**If the agent is not specified**: assign one before starting the Task.

---

## Tasks That Suit Each Agent

### developer

- Implement API endpoints
- Build service logic
- Fix bugs
- Refactor code
- Write unit tests

### devops

- Docker configuration
- CI/CD pipelines
- Environment variable setup
- Deploy scripts
- Monitoring setup

### architect

- Author ADRs
- Architecture design documents
- Tech stack decisions
- System design review
- Domain model design

### qa-engineer

- Write integration tests
- Write E2E tests
- Performance tests
- Test automation
- Quality reports

### tech-writer

- Write / update README
- API documentation
- User guides
- Tutorials
- Release notes
