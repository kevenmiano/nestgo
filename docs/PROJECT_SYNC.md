# GitHub Project #2 — automated metadata synchronization

[Project #2](https://github.com/users/kevenmiano/projects/2) is synchronized from NestGo repository issues by [project-sync.yml](../.github/workflows/project-sync.yml).

## Authentication

Configure the repository Actions secret `PROJECTS_TOKEN` using a GitHub classic PAT owned by a user with access to Project #2 and the `project` scope. Do not commit the token. The workflow uses metadata only and does not check out PR code.

## Source of truth

Issue descriptions contain these exact metadata lines:

```md
**Roadmap:** M002 — Core architecture
**Priority:** P0
**GSD status:** Backlog
```

The workflow extracts the milestone identifier `M001`–`M006` from `**Roadmap:**` or `**Milestone:**`, and priority `P0`–`P2` from `**Priority:**`. It creates missing **Milestone** and **Priority** single-select fields and options when possible. Existing fields must be single-select with the matching options; an incompatible field fails loudly instead of silently corrupting the board. The existing **Status** field must offer `Backlog`, `In Review` and `Done`.

| Event | Action |
| --- | --- |
| Issue opened | Add to project, set Backlog and sync metadata |
| Issue edited, labeled, unlabeled or milestone changed | Sync metadata without resetting status |
| Issue reopened | Sync metadata, preserve status |
| Issue closed | Set Done and sync metadata |
| PR opened/reopened/edited/ready for review | Set referenced issues In Review |
| PR merged | Set referenced issues Done |
| PR closed without merge | Set referenced issues Backlog |
| Manual dispatch with backfill enabled | Import all repository issues and reconcile metadata, preserving existing Status |

PR references in the description must use `Refs #N`, `Closes #N`, `Fixes #N`, or `Resolves #N`. References only in the GitHub sidebar are not detected.

## One-time setup

1. Ensure `PROJECTS_TOKEN` is configured and Project #2 is accessible to its owner.
2. Merge the workflow PR.
3. Open **Actions → Sync NestGo issues to GitHub Project → Run workflow**, with `backfill=true`.
4. Verify that issues #10–#25 have **Priority** and **Milestone** values. Check the workflow logs if not.
5. In Project #2, create a **Kanban** view grouped by Status and a **Roadmap** or table view grouped/filtered by Milestone, showing Priority. Project views themselves are not created by this workflow.

The milestone values are release groups, **not dates**. A true GitHub Projects Roadmap timeline requires start/target date fields and scheduling decisions. Do not invent dates.

## Troubleshooting and limitations

- Missing or insufficient token: GraphQL authentication/authorization fails; rotate or fix `PROJECTS_TOKEN`.
- Existing Priority/Milestone field has the wrong type or missing options: adjust it in Project settings.
- A manually edited Project field can be overwritten on the next issue edit or backfill; **issue metadata is authoritative**.
- The sync updates field values only when metadata is present; removing a metadata line does not clear the prior Project value.
- GitHub Actions execution and board population must be verified after merging. No passing run or board configuration is claimed in this document.
