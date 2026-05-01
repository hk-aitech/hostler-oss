# First Workflow Example

> An example showing the full flow for a project using the hostler plugin for the first time.

---

## Step 1: Initialize the project

Create the project with `/hstl-oss:project:init`.

```
/hstl-oss:project:init Build a user authentication API in Python with FastAPI.
JWT-based auth, PostgreSQL. It's the auth service in a microservice architecture.
```

**Expected outcome:**
- Standard documentation structure is generated under `docs/` (`00-project/`, `02-architecture/`, `03-design/`, etc.).
- A project definition draft is written into `docs/00-project/pdd.md`.
- A Phase/Sprint roadmap draft is written into `docs/00-project/roadmap.md`.
- `works/`, `.claude/`, `CLAUDE.md`, `README.md`, and `.gitignore` are created.

Review the TODO items in the draft documents and fill them in.

---

## Step 2: Sprint planning

Start a Sprint. Set the Sprint scope and goal.

```
/hstl-oss:sprint:start
```

**Expected outcome:**
- `works/sprints/active//SPRINT.md` is created.
- The Sprint goal, period, and included Task list are recorded.
- `works/CURRENT-FOCUS.md` reflects the current Sprint.

---

## Step 3: Create and implement Tasks

Create Tasks within the Sprint and implement them.

```
/hstl-oss:task:start Implement user registration API endpoint
```

**Expected outcome:**
- `works/sprints/active//tasks/-user-registration-api.md` is created.
- The Task file records the goal, completion criteria, and estimated time.
- `works/CURRENT-FOCUS.md` is updated to point at the in-progress Task.

After finishing the implementation work, complete the Task.

```
/hstl-oss:task:complete
```

**Expected outcome:**
- The Task file's `status` becomes `done`.
- Completion timestamp and actual elapsed time are recorded.
- `works/CURRENT-FOCUS.md` is updated.

---

## Step 4: Code review

Review the quality of the implemented code. If you say "review my code", the `code-review` skill is invoked automatically.

```
Review my code — src/api/auth.py
```

**Expected outcome:**
- Findings are emitted across code-quality, security, and performance dimensions.
- Improvement suggestions are presented with priorities.
- Missing tests are flagged when applicable.

---

## Step 5: Complete the Sprint

After every Task in the Sprint is done, close out the Sprint.

```
/hstl-oss:sprint:complete
```

**Expected outcome:**
- The Sprint folder moves from `works/sprints/active/` to `works/sprints/completed/`.
- A Sprint review document is generated at `docs/06-reports/sprints/p1-s1-review.md`.
- Metrics like completion rate, carry-over Tasks, and velocity are recorded.

---

## Step 6: Retrospective

Record the lessons learned during the Sprint. Typing `/retro` invokes the `retro` skill automatically.

```
/retro
```

**Expected outcome:**
- The work history during the Sprint is analyzed and retrospective items are derived.
- Keep / Problem / Try are organized.
- Lessons are recorded under `docs/07-knowledge/`.
- Action items to apply in the next Sprint are presented.

---

## Full Flow Summary

```
Initialize → Sprint planning → Task creation/implementation → Code review → Sprint complete → Retrospective
      ↑                                                                                       │
      └─────────────────────── Next Sprint ──────────────────────────────────────────────────┘
```

Repeat this cycle Sprint by Sprint to build the project incrementally.
