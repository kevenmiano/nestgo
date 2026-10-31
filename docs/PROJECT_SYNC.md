# GitHub Project #2 — issue metadata synchronization

[Project #2](https://github.com/users/kevenmiano/projects/2) is synchronized from issues by [project-sync.yml](../.github/workflows/project-sync.yml).

## Authentication and security

Configure `PROJECTS_TOKEN` as a repository Actions secret with the `project` scope and access to Project #2. Never commit the token. The workflow uses `pull_request_target`, but checks out **only the default branch**, never the PR head. The [policy tests](../.github/workflows/project-sync-tests.yml) require no credentials.

## Project fields

GitHub's native **Milestone** field is not necessarily a single-select. Do not overwrite or delete it.

| Project field | Type | Options |
| --- | --- | --- |
| Status | Existing single-select | The workflow supports common names including Todo, In Progress and Done; In Review is optional |
| Priority | Custom single-select | P0, P1, P2 |
| **Roadmap** | **Custom single-select** | M001, M002, M003, M004, M005, M006 |
| Milestone | GitHub native | Not modified by this workflow |

Missing Priority and Roadmap fields are created automatically. If an existing field has an incompatible type or lacks an expected option, the workflow stops with an actionable error.

## Source of truth

The workflow reads issue bodies such as:

```md
**Roadmap:** M002 — Core architecture
**Priority:** P0
**GSD status:** Backlog
```

For backward compatibility, `**Milestone:** M002` is also accepted as an **issue-body** source. The identifier is always written into the custom Project **Roadmap** field, never the native Milestone field.

## Status lifecycle

| Event | Project Status |
| --- | --- |
| Issue opened or reopened | Backlog |
| Issue closed | Done |
| Issue edited | Preserve existing Status |
| Backfill | Closed issues become Done; open issue statuses are preserved |
| PR opened / ready for review referencing an open issue | In Review if available; otherwise In Progress, or preserve the existing Status |
| Draft PR | In Progress, if available |
| Merged PR with `Refs #N` | In Progress, if available; **never Done** |
| Merged PR with `Closes #N`, `Fixes #N`, or `Resolves #N` | Done **only if the GitHub issue is actually closed**; otherwise In Progress |
| PR closed without merge | In Progress, if available |
| PR references an already closed issue | Done |

**Source of truth:** GitHub issue state, not PR text. A closing keyword alone cannot promote an open issue to Done. GitHub normally closes an issue via a qualifying merged PR targeting the default branch; if it remains open, investigate linkage/merge target or unmet acceptance criteria before manually closing it.

Status labels are matched to existing Project options rather than created or renamed: Backlog can map to Todo, In Review can map to In Progress, and Done can map to Completed. If no safe match exists, the workflow logs a warning and preserves the existing Status. It never maps review or in-progress work to Done or Backlog. Explicit closing keywords take precedence over a plain reference. Use `Refs #N` for partial work and closing keywords only when the complete issue is delivered.

## Reconciliation

1. Merge the fix after the **Project sync policy tests** check passes.
2. A push to main modifying the workflow or its logic **automatically triggers a backfill**. To retry, open **Actions → Sync NestGo issues to GitHub Project → Run workflow** with `backfill=true`.
3. Verify that issues #10–#25 show the expected Roadmap and Priority selections; closed issues are Done and open issue statuses remain preserved during backfill.
4. Configure Kanban grouped by Status and a Roadmap/table view that displays the custom Roadmap field. Project views are not created by the workflow.

The M001–M006 values represent **groups, not dates**. A timeline requires explicit dates; do not invent them.

## Troubleshooting

- `Milestone must be a single-select field`: old workflow incorrectly collided with the native GitHub Milestone field. This fix uses Roadmap.
- `Roadmap must be a single-select field`: inspect the existing Roadmap field in Project settings.
- Project not found or permission denied: check `PROJECTS_TOKEN` and Project access.
- Missing Priority or Roadmap option: add the expected option in Project settings.
- Missing `In Review` Status: automatically map to `In Progress` if available; inspect the log for the chosen option. No Project Status options are changed automatically.
- A successful CI build does not prove the Project was populated. Check the synchronization Action logs after merging.

The sync changes custom field values only when matching metadata is present; removing an issue-body line does not clear the old Project value.
