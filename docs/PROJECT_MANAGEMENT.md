# GSD Core and GitHub tracking

NestGo uses an issue-driven workflow based on [GSD Core](https://github.com/open-gsd/gsd-core). The repository includes an initial `.planning/` scaffold. **GSD Core is not installed by these files alone**; run `/gsd-onboard` in a supported agent environment to initialize and validate the workflow.

## Sources of truth

- GitHub Issues: task scope, ownership, priority, acceptance criteria and status history.
- GitHub Projects: visual board and work-in-progress management.
- `.planning/`: project context, requirements, phase roadmap, decisions and state.
- Pull requests: implementation, review and verification evidence.

## Board specification

Create a GitHub Project named **NestGo — Engineering Roadmap** with a single-select Status field:

`Backlog → Ready → In Progress → In Review → Blocked → Done`

Suggested additional fields: Priority (P0/P1/P2), Phase (0/1/2/3/Cross-cutting), Type (Feature/Bug/Docs/Chore/Test/CI), and Milestone (M001).

**Note:** The GitHub Project must be created separately. The connected GitHub integration currently exposes issue management but not GitHub Projects creation.

## Existing backlog

| Issue | Focus | Priority | Phase |
| --- | --- | --- | --- |
| [#2](https://github.com/kevenmiano/nestgo/issues/2) | GSD onboarding and board | P0 | 0 |
| [#3](https://github.com/kevenmiano/nestgo/issues/3) | CI and quality gates | P0 | 1 |
| [#4](https://github.com/kevenmiano/nestgo/issues/4) | Compiler tests | P0 | 1 |
| [#5](https://github.com/kevenmiano/nestgo/issues/5) | Runtime correctness | P0 | 2 |
| [#6](https://github.com/kevenmiano/nestgo/issues/6) | Module contracts | P1 | 3 |
| [#7](https://github.com/kevenmiano/nestgo/issues/7) | Documentation and ADRs | P1 | Cross-cutting |

## Working agreement

1. Capture every proposed change in an issue before implementation.
2. Link it to a requirement ID and a GSD phase.
3. Discuss and plan the work using GSD Core.
4. Create a focused branch and PR that references the issue.
5. Include test results, verification evidence, and documentation changes.
6. Merge only after review and required checks.
7. Close the issue and update board and `.planning/STATE.md`.

## Suggested branch and commit conventions

Branch: `feat/<issue-number>-short-slug`, `fix/<issue-number>-short-slug`, or `docs/<issue-number>-short-slug`.

Commit: `feat(compiler): validate route metadata`, `test(runtime): cover invalid handler signatures`, `docs: clarify GSD workflow`.

## Local GSD onboarding

In a GSD Core-enabled coding agent:

```text
/gsd-onboard
/gsd-progress
/gsd-discuss-phase 1
/gsd-plan-phase 1
```

Review any GSD-generated files before replacing this initial scaffold. GitHub issues remain authoritative for work status.
