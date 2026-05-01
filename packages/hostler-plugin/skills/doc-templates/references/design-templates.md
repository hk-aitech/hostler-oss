# Design document templates

> Eight templates for various design documents (§8 added during the Spike-result standardization)

---

## Table of contents

1. [Architecture Overview](#1-architecture-overview-template)
2. [Domain Design](#2-domain-design-template)
3. [Feature Design](#3-feature-design-template)
4. [API Design](#4-api-design-template)
5. [Configuration Design](#5-configuration-design-template)
6. [Deployment Design](#6-deployment-design-template)
7. [IaC Design](#7-iac-design-template)
8. [Spike-result doc — §4 follow-up Task input labelling mandate (`docs/07-knowledge/architecture/patterns.md#A027`)](#8-spike-result-doc-template)

---

## 1. Architecture Overview template

```markdown
# Architecture Overview

## Overview

| Item | Content |
|------|---------|
| Document version | 1.0 |
| Last updated | {YYYY-MM-DD} |
| Status | Draft / Review / Approved |

---

## 1. System overview

{Briefly describe the system's purpose and major features}

---

## 2. Architecture principles

| Principle | Description |
|-----------|-------------|
| {principle 1} | {description} |
| {principle 2} | {description} |

---

## 3. System architecture

### 3.1 High-level architecture

```
┌─────────────────────────────────────────────────────────┐
│                      Client Layer                        │
├─────────────────────────────────────────────────────────┤
│                      API Gateway                         │
├─────────────────────────────────────────────────────────┤
│                    Service Layer                         │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐   │
│  │Service A│  │Service B│  │Service C│  │Service D│   │
│  └─────────┘  └─────────┘  └─────────┘  └─────────┘   │
├─────────────────────────────────────────────────────────┤
│                     Data Layer                           │
└─────────────────────────────────────────────────────────┘
```

### 3.2 Component structure

| Component | Role | Tech |
|-----------|------|------|
| {component 1} | {role} | {tech} |
| {component 2} | {role} | {tech} |

---

## 4. Communication patterns

### 4.1 Synchronous

- {pattern 1}

### 4.2 Asynchronous

- {pattern 2}

---

## 5. Data architecture

### 5.1 Data stores

| Store | Purpose | Tech |
|-------|---------|------|
| {store 1} | {purpose} | {tech} |

---

## 6. Deployment architecture

### 6.1 Environments

| Environment | Purpose | Notes |
|-------------|---------|-------|
| Development | Development | {notes} |
| Staging | Testing | {notes} |
| Production | Production | {notes} |

---

## 7. Security architecture

- {security concern 1}
- {security concern 2}

---

## 8. Scalability considerations

- {scalability point 1}
- {scalability point 2}
```

---

## 2. Domain Design template

```markdown
# {Domain name} Domain Design

## Overview

| Item | Content |
|------|---------|
| Domain | {domain name} |
| Bounded Context | {context name} |
| Version | 1.0 |
| Last updated | {YYYY-MM-DD} |

---

## 1. Domain overview

### 1.1 Purpose

{Domain purpose and responsibilities}

### 1.2 Core concepts

| Concept | Description |
|---------|-------------|
| {concept 1} | {description} |
| {concept 2} | {description} |

---

## 2. Aggregate design

### 2.1 Design principles

- **Consistency boundary**: {description}
- **Transaction boundary**: {description}
- **Reference rules**: {description}
- **Small size**: {description}

### 2.2 Aggregate list

| Aggregate | Root | Responsibility |
|-----------|------|----------------|
| {Aggregate 1} | {RootClass} | {responsibility} |

---

## 3. Entity definitions

### 3.1 {Entity Name}

```
{Entity name}
├── id: {ID type}
├── {attribute 1}: {type}
├── {attribute 2}: {type}
└── {method}()
```

**Business rules:**
- {rule 1}

---

## 4. Value Object definitions

### 4.1 {Value Object Name}

| Attribute | Type | Description |
|-----------|------|-------------|
| {attribute} | {type} | {description} |

**Validation rules:**
- {rule 1}

---

## 5. Domain Events

| Event | Description | Publisher | Subscribers |
|-------|-------------|-----------|-------------|
| {Event 1} | {description} | {publisher} | {subscribers} |

---

## 6. Repository pattern

### 6.1 Repository interface

```
I{Aggregate}Repository
├── GetByIdAsync(id): Task<{Aggregate}>
├── SaveAsync(entity): Task
└── DeleteAsync(id): Task
```

### 6.2 Repository implementation guide

- Define a Repository per Aggregate (not per Entity)
- Consider separating Read and Write models
- Caching strategy belongs inside the Repository
```

---

## 3. Feature Design template

```markdown
# {Feature name} Design Document

## Overview

| Item | Content |
|------|---------|
| Version | 1.0 |
| Status | Draft / Design Complete / Implemented |
| Phase | Phase {N} |
| Date | {YYYY-MM-DD} |

---

## 1. Overview

### 1.1 Document purpose

{The purpose of this document}

### 1.2 Background

{Why the feature is needed}

### 1.3 Scope

**Included:**
- {included item}

**Excluded:**
- {excluded item}

---

## 2. Requirements

### 2.1 Functional requirements

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-001 | {requirement} | P0/P1/P2 |
| FR-002 | {requirement} | P0/P1/P2 |

### 2.2 Non-functional requirements

| ID | Requirement | Target |
|----|-------------|--------|
| NFR-001 | Response time | < {X}ms |
| NFR-002 | Availability | {X}% |

---

## 3. Architecture design

### 3.1 Component structure

{component diagram}

### 3.2 Sequence diagram

{sequence diagram}

---

## 4. Data model

### 4.1 Entities

| Entity | Description | Attributes |
|--------|-------------|------------|
| {entity} | {description} | {attribute list} |

---

## 5. API design

### 5.1 Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/{resource} | {description} |
| POST | /api/{resource} | {description} |

---

## 6. Implementation plan

### 6.1 Task breakdown

| Task | Description | Size | Owner |
|------|-------------|------|-------|
| {Task 1} | {description} | S/M/L | {owner} |

---

## 7. Test plan

| Type | Scope | Criterion |
|------|-------|-----------|
| Unit tests | {scope} | Coverage {X}% |
| Integration tests | {scope} | {criterion} |
```

---

## 4. API Design template

```markdown
# {API name} API Design

## Overview

| Item | Content |
|------|---------|
| API name | {API name} |
| Version | v1 |
| Base URL | `/api/v1/{resource}` |
| Auth | {auth method} |

---

## 1. Endpoint list

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | `/{resource}` | List | Required |
| GET | `/{resource}/{id}` | Get one | Required |
| POST | `/{resource}` | Create | Required |
| PUT | `/{resource}/{id}` | Update | Required |
| DELETE | `/{resource}/{id}` | Delete | Required |

---

## 2. Common

### 2.1 Authentication

**Header:**
```
Authorization: Bearer {token}
```

### 2.2 Common response format

**Success:**
```json
{
  "success": true,
  "data": { ... },
  "meta": {
    "timestamp": "2025-01-01T00:00:00Z"
  }
}
```

**Failure:**
```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Error message"
  }
}
```

### 2.3 Error codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| UNAUTHORIZED | 401 | Authentication required |
| FORBIDDEN | 403 | No permission |
| NOT_FOUND | 404 | Resource not found |
| VALIDATION_ERROR | 400 | Validation failed |

---

## 3. Endpoint details

### 3.1 GET `/{resource}`

**Description:** list {resources}

**Query parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| page | integer | N | Page number (default: 1) |
| size | integer | N | Page size (default: 20) |
| sort | string | N | Sort key |

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "uuid",
      "field": "value"
    }
  ],
  "meta": {
    "page": 1,
    "size": 20,
    "total": 100
  }
}
```

### 3.2 POST `/{resource}`

**Description:** create a {resource}

**Request body:**
```json
{
  "field1": "value1",
  "field2": "value2"
}
```

**Validation:**

| Field | Rule |
|-------|------|
| field1 | required, max:100 |
| field2 | required |

**Response:** (201 Created)
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "field1": "value1"
  }
}
```
```

---

## 5. Configuration Design template

```markdown
# Configuration Design

## Overview

| Item | Content |
|------|---------|
| Version | 1.0 |
| Last updated | {YYYY-MM-DD} |

---

## 1. Configuration structure

### 1.1 File locations

```
config/
├── appsettings.json           # Default settings
├── appsettings.Development.json
├── appsettings.Staging.json
├── appsettings.Production.json
└── secrets/                   # Secrets (gitignored)
```

### 1.2 Configuration layers

```
Default config (appsettings.json)
    │
    ▼
Per-environment config (appsettings.{Environment}.json)
    │
    ▼
Environment variables
    │
    ▼
Secrets (Secret Manager / Vault)
```

---

## 2. Configuration items

### 2.1 Application config

| Key | Description | Default | Required |
|-----|-------------|---------|----------|
| App:Name | App name | {value} | Y |
| App:Version | Version | {value} | Y |

### 2.2 Database config

| Key | Description | Per-env |
|-----|-------------|---------|
| Database:ConnectionString | DB connection | Y |
| Database:MaxPoolSize | Pool size | N |

---

## 3. Per-environment config

### 3.1 Development

```json
{
  "Database": {
    "ConnectionString": "localhost..."
  },
  "Logging": {
    "Level": "Debug"
  }
}
```

### 3.2 Production

```json
{
  "Database": {
    "ConnectionString": "${DATABASE_URL}"
  },
  "Logging": {
    "Level": "Warning"
  }
}
```

---

## 4. Secrets management

### 4.1 Secret list

| Secret | Use | Storage |
|--------|-----|---------|
| DB Password | DB access | Secret Manager |
| API Key | External API | Vault |

---

## 5. Validation

### 5.1 Required-config validation

| Setting | Rule |
|---------|------|
| Database:ConnectionString | Not empty |
| App:Name | Not empty |
```

---

## 6. Deployment Design template

```markdown
# Deployment Design

## Overview

| Item | Content |
|------|---------|
| Version | 1.0 |
| Last updated | {YYYY-MM-DD} |

---

## 1. Deployment architecture

### 1.1 Environments

| Environment | Purpose | URL |
|-------------|---------|-----|
| Development | Development | dev.example.com |
| Staging | Testing | staging.example.com |
| Production | Production | example.com |

### 1.2 Infrastructure diagram

```
┌─────────────────────────────────────────────┐
│                 Load Balancer                │
└─────────────────────────────────────────────┘
                      │
        ┌─────────────┼─────────────┐
        ▼             ▼             ▼
   ┌─────────┐   ┌─────────┐   ┌─────────┐
   │ App 1   │   │ App 2   │   │ App 3   │
   └─────────┘   └─────────┘   └─────────┘
        │             │             │
        └─────────────┼─────────────┘
                      ▼
              ┌─────────────┐
              │  Database   │
              └─────────────┘
```

---

## 2. Container layout

### 2.1 Docker images

| Service | Image | Port |
|---------|-------|------|
| {service 1} | {image} | {port} |

### 2.2 Docker Compose

```yaml
version: '3.8'
services:
  app:
    image: {image}
    ports:
      - "{port}:{port}"
    environment:
      - ENV_VAR=value
```

---

## 3. CI/CD pipeline

### 3.1 Workflow

```
┌────────┐    ┌────────┐    ┌────────┐    ┌────────┐
│  Push  │ ─▶ │ Build  │ ─▶ │  Test  │ ─▶ │ Deploy │
└────────┘    └────────┘    └────────┘    └────────┘
```

### 3.2 Deploy triggers

| Branch | Environment | Trigger |
|--------|-------------|---------|
| develop | Development | Auto |
| main | Staging | Auto |
| release/* | Production | Manual approval |

---

## 4. Scaling strategy

### 4.1 Horizontal scaling

| Condition | Action |
|-----------|--------|
| CPU > 70% | Add instance |
| CPU < 30% | Remove instance |

---

## 5. Monitoring

| Item | Tool | Alert condition |
|------|------|-----------------|
| Server health | {tool} | On down |
| Response time | {tool} | > {X}ms |
| Error rate | {tool} | > {X}% |

---

## 6. Rollback procedure

1. {step 1}
2. {step 2}
3. {step 3}
```

---

## 7. IaC Design template

```markdown
# Infrastructure as Code (IaC) Design

## Overview

| Item | Content |
|------|---------|
| IaC tool | {Terraform / Pulumi / CloudFormation} |
| Version | 1.0 |
| Last updated | {YYYY-MM-DD} |

---

## 1. IaC structure

### 1.1 Directory layout

```
deploy/
├── terraform/
│   ├── modules/
│   │   ├── network/
│   │   ├── compute/
│   │   └── database/
│   ├── environments/
│   │   ├── dev/
│   │   ├── staging/
│   │   └── prod/
│   └── main.tf
└── scripts/
    └── setup.sh
```

### 1.2 Modules

| Module | Description | Resources |
|--------|-------------|-----------|
| network | Networking | VPC, Subnet, Security Group |
| compute | Compute | EC2, ECS, Lambda |
| database | Database | RDS, DynamoDB |

---

## 2. Resource definitions

### 2.1 Network

| Resource | Name | Settings |
|----------|------|----------|
| VPC | {name}-vpc | CIDR: 10.0.0.0/16 |
| Subnet | {name}-subnet-{az} | CIDR: 10.0.{N}.0/24 |

---

## 3. Per-environment config

### 3.1 Variable files

**dev.tfvars:**
```hcl
environment = "dev"
instance_type = "t3.micro"
```

**prod.tfvars:**
```hcl
environment = "prod"
instance_type = "t3.large"
```

---

## 4. State management

### 4.1 Backend configuration

```hcl
terraform {
  backend "s3" {
    bucket = "{state-bucket}"
    key    = "{project}/terraform.tfstate"
    region = "{region}"
  }
}
```

---

## 5. Deployment procedure

### 5.1 Manual deploy

```bash
# Init
terraform init

# Plan
terraform plan -var-file=environments/{env}.tfvars

# Apply
terraform apply -var-file=environments/{env}.tfvars
```

---

## 6. Tagging conventions

| Tag | Description | Example |
|-----|-------------|---------|
| Environment | Environment | dev, prod |
| Project | Project | {project-name} |
| Owner | Owner | {team} |

---

## 8. Spike-result doc template

> **Convention mandate (standardized in `docs/07-knowledge/architecture/patterns.md#A027`)**: a Spike-type Task's
> design-doc artifact MUST include a §4 "Follow-up Task input dependencies" sub-section. Label every
> follow-up Task ID that the spike feeds, so the follow-up Task can read just §4 — not the entire spike
> body — to pick up the right input.
>
> Reference example: `docs/03-design/sample-spike.md` §4.1
> (a real spike with 5 follow-up Task labels).

```markdown
---
uuid: {UUID v7 — hstl-oss uuid generate}
title: {Spike topic} — {YYYY-MM-DD}
date: {YYYY-MM-DD}
sprint: {sprint-NN}
task: T{NNN}
related_adr: ADR-{NNN}  # when applicable
followups: [T{NNN}, T{NNN}, ...]  # all follow-up Task IDs the spike feeds — must align 1:1 with §4 label matrix
doc_type: spike-result
---

# {Spike topic}

## 1. Context

{Why the spike was needed — options to decide / open questions, 1–3 items}

## 2. Option comparison (measurements + trade-offs)

### Option A — {name}
- Pros: {…}
- Cons: {…}
- Measurements (when applicable): {numbers / benchmarks / PoC results}

### Option B — {name}
- Same structure

## 3. Chosen option line (used to update ADR-{NNN} §{Q-decision})

> {one-line statement of the chosen option + 1-paragraph rationale}. This line becomes the SSOT for the ADR amendment.

## 4. Follow-up Task input dependencies (mandatory — `docs/07-knowledge/architecture/patterns.md#A027`)

### 4.1 Label matrix

| Follow-up Task ID | Input dependency | Spike § reference |
|-------------------|------------------|-------------------|
| T{NNN} | {input label — e.g. "JSONL persistence chosen + envelope schema v1"} | §3 / appendix |
| T{NNN} | {input label} | §… |

### 4.2 Label-writing rules

- Label = a one-line statement of "which decision / number / artifact from this spike does the follow-up Task consume as input"
- If a follow-up Task depends on multiple inputs, separate them with commas (e.g. "JSONL chosen, envelope schema v1, 30-day retention")
- One row per follow-up Task ID — must match the frontmatter `followups` field exactly

## 5. Open issues / deferred to next Sprint

{Out-of-scope items + follow-up spike candidates}

## 6. References

- ADR: `docs/02-architecture/adrs/ADR-{NNN}.md` §{section}
- Related Task / Sprint: T{NNN} / sprint-{NN}
- KB: A027 (this convention), A028 (handling spike → ADR amendment → PoC in the same sprint)
```

### Good vs bad examples

**Good** (spike `protocol-spike-2026-04-27.md` §4.1):

```
| | JSONL persistence chosen + envelope schema v1 + retention policy | §3.6, appendix A |
| | Topic list (5 channels) consumed by `hstl-oss worker` subcommand | §2.3 |
| | (sample) external client polling topic for decision requests | §2.4 |
| | system.audit topic where audit hash chain applies | §2.5 |
| | ADR §D4 Q3 amendment input — JSONL Stage 1 / SQLite Stage 2 | §3.6 chosen-option line |
```

**Bad** (no labels — follow-up Task forced to read the entire spike body to extract its input):

```
## 4. Follow-up work
T..., T..., T... will use the spike result.
```

→ Doesn't state which input each follow-up Task consumes. Each follow-up Task ends up with a manual cross-link line in its body.
```
