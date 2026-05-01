# Operations document templates

> Templates for runbooks, deployment, monitoring, and incident response

## Table of contents

1. [Runbook](#runbook)
2. [Deployment playbook](#deployment-playbook)
3. [Monitoring setup guide](#monitoring-setup-guide)

## Runbook

```markdown
# {System name} Runbook

> Last updated: YYYY-MM-DD
> Audience: {operators/developers}

## System overview

| Item | Value |
|------|-------|
| Process | {process name} |
| Run command | `{command}` |
| Port | {port} |
| Logs | {log path} |
| Monitoring | {dashboard URL} |

## Start / Stop

### Start
```bash
{start command}
```

### Stop
```bash
{stop command}
```

### Restart
```bash
{restart command}
```

## Healthcheck

```bash
{healthcheck command}
```

| State | Meaning | Action |
|-------|---------|--------|
| Healthy | {description} | None |
| Warning | {description} | {action} |
| Failed | {description} | {action} |

## Incident response

### Symptom 1: {symptom}
**Cause**: {cause}
**Action**: {step-by-step actions}

## Configuration changes

| Setting | File | How to change |
|---------|------|---------------|
| {setting} | {file path} | {procedure} |
```

## Deployment playbook

```markdown
# {System name} Deployment Playbook

> Last updated: YYYY-MM-DD

## Preconditions

- [ ] {condition 1}
- [ ] {condition 2}

## Deployment procedure

### 1. Pre-checks
```bash
{validation command}
```

### 2. Deploy
```bash
{deployment command}
```

### 3. Post-checks
```bash
{validation command}
```

## Rollback

```bash
{rollback command}
```

## Change history

| Date | Version | Change |
|------|---------|--------|
```

## Monitoring setup guide

```markdown
# {System name} Monitoring Setup

> Based on Prometheus + Grafana

## Metrics

| Metric | Type | Description | Alert condition |
|--------|------|-------------|-----------------|
| {metric_name} | Counter/Gauge/Histogram | {description} | {condition} |

## Dashboard

| Panel | PromQL | Threshold |
|-------|--------|-----------|
| {panel name} | `{PromQL}` | {green/yellow/red} |

## Alert rules

| Rule | Condition | Channel | Severity |
|------|-----------|---------|----------|
| {rule name} | {condition} | Slack #{channel} | critical/warning |
```
