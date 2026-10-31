# GitHub Project #2 synchronization

The workflow [`project-sync.yml`](../.github/workflows/project-sync.yml) synchronizes NestGo issues with the user-owned GitHub Project at https://github.com/users/kevenmiano/projects/2.

## One-time authorization

Create a **classic personal access token** for the `kevenmiano` account with the `project` scope. This is necessary because the default repository `GITHUB_TOKEN` does not provide the required user-project permissions. Use the narrowest possible lifetime and rotate it periodically.

1. Open GitHub → Settings → Developer settings → Personal access tokens → Tokens (classic).
2. Generate a token with the `project` scope.
3. In `kevenmiano/nestgo`, open Settings → Secrets and variables → Actions.
4. Create a repository secret named **`PROJECTS_TOKEN`** with that token.
5. Confirm the Project #2 has a single-select **Status** field and options named exactly **Backlog**, **In Review**, and **Done** (other columns may coexist).
6. Open Actions → **Sync NestGo issues to GitHub Project** → Run workflow, with **backfill** enabled.

Never commit a token, paste it in an issue, or send it through chat.

## Events and behavior

| Trigger | Effect |
| --- | --- |
| Issue opened/reopened | Add issue to project and set Backlog |
| Pull request opened/reopened/ready for review | Set linked issues to In Review |
| Pull request merged | Set linked issues to Done |
| Pull request closed without merge | Return linked issues to Backlog |
| Manual workflow dispatch | Import **all open issues** and set Backlog |

The PR body must reference local issues using `Closes #N`, `Fixes #N`, `Resolves #N`, or `Refs #N`. Only issues in the same repository are resolved.

**Caveats:** Manual backfill resets the status of all open issues to Backlog. Avoid rerunning it after the board is actively managed. The workflow does not yet synchronize Priority, Phase, Milestone or estimates; these remain manual project fields. Issues linked only in GitHub's sidebar but not in the PR body are not detected. No code from pull requests is checked out or executed.

## Troubleshooting

- **Missing secret:** GitHub GraphQL requests fail authentication; add `PROJECTS_TOKEN`.
- **Project not found:** Confirm the token owner has access to user Project #2 and the `project` scope.
- **Status missing:** Create the matching single-select Status option. The item is still added if a status option is missing.
- **Issue not updated by PR:** Add a reference such as `Closes #5` to the PR description and reopen or update the PR through a supported trigger.

## Security

The workflow uses `pull_request_target` for metadata-only processing and never checks out or executes untrusted PR code. Keep the workflow permissions minimal, protect the default branch and limit who can edit workflows. Review any workflow changes before merging.
