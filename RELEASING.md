# Releasing Euxenion / Xenri

This document is the release runbook for human maintainers and coding agents.
**A PR merge is not permission to publish.** Agents may prepare release PRs,
propose version bumps, backports, and validation, but must not merge PRs,
create/push version tags, or publish releases without explicit human approval.

## Version contract

- The repository-root [`VERSION`](VERSION) is the source of truth for the
  version of the current branch: a single `MAJOR.MINOR.PATCH` line, with no
  leading `v` and no prerelease suffix (for example, `0.0.1`).
- `go build ./cmd/xenri` and `just build cli` always embed this base version
  and report `xenri vMAJOR.MINOR.PATCH-dev`. No git metadata or GoReleaser
  is needed to identify the source version.
- An official release must use exactly `v` followed by the `VERSION`
  contents. GoReleaser stamps official binaries with the version derived
  from the Git tag: `xenri vMAJOR.MINOR.PATCH`. A release workflow check
  rejects a tag that differs from the source-tree version.
- VERSION is **per branch**. After cutting a release, prepare the next
  planned version on `main` in a separate PR. A servicing branch advances
  PATCH in its own PR before each subsequent release.
- Version numbers and tags are never recycled. Published release tags and
  assets should be treated as immutable. A failed release is investigated,
  not repaired by deleting/repointing the tag.

The `-dev` suffix identifies *unofficial* builds, not uniqueness. Multiple
commits built from the same VERSION can report the same version; `git rev-parse
HEAD` and Go build metadata are available to distinguish source revisions.
This scheme does not guarantee reproducible binaries or attest provenance.

## Branch and tag model

- `main` carries ongoing development and normal new releases.
- A single servicing branch `release/vMAJOR.MINOR` (for example,
  `release/v0.1`) may be cut from the published release commit when that
  minor line needs independent fixes.
- Maintenance patches are introduced through **PRs targeted at their
  servicing branch**, not by direct pushes. Backport via cherry-pick onto a
  topic branch and open a PR; update `VERSION` in the same PR when preparing
  the next patch version.
- A tag `vMAJOR.MINOR.PATCH` is valid only when its commit is reachable
  from `origin/main`, or from the matching
  `origin/release/vMAJOR.MINOR` branch. The release workflow checks tag
  syntax, a matching VERSION, the exact tagged checkout, and ancestry.
  An unrelated branch, even if it contains a matching VERSION, cannot
  release.
- Do not routinely merge servicing branches back to main; forward-port
  individual fixes so their intent and compatibility can be reviewed.
- Each branch has its own GoReleaser configuration and release workflow.
  When changing the publishing mechanism, explicitly consider backporting
  its fixes to supported servicing branches.

### Branch cut example

After `v0.1.0` is published, if the 0.1 line needs servicing:

```bash
git fetch origin --tags
git switch --detach v0.1.0
git switch -c release/v0.1
git push -u origin release/v0.1
```

Creating a new release branch is an explicit maintainer operation. Do this
**before** advancing main's VERSION to 0.2.0. Protect the branch immediately
using a repository ruleset matching `release/v*` and requiring PRs, linear
history, no force pushes/deletion, and the same CI checks as main.

For example, to prepare `v0.1.1`, open a backport PR against
`release/v0.1`, change its VERSION to `0.1.1`, validate CI, and squash
merge with human approval. Then create the tag on that branch's merged tip.

## Repository settings prerequisites (maintainer)

Configure and periodically audit these in GitHub Settings:

1. **Branch rulesets:** protect `main` and `release/v*`: PRs only, squash
   merge, required successful Go/Just/release snapshot checks, linear
   history, disallow force pushes and deletion. Keep review count at zero
   while this is a single-maintainer repository.
2. **Tag rulesets:** target `v*`; prevent tag update/deletion without
   bypass, and separately restrict tag creation to the human release
   maintainer. Ensure this works for both main and servicing releases.
3. **GitHub Actions:** default read-only token; allow write only in the
   tag-triggered publishing job. Protect release workflow changes through
   the normal PR process.
4. **GitHub Releases:** enable release immutability before v0.0.1, where
   supported. Tag protection and immutable published assets are distinct.

GitHub settings are not automatically configured by files in this repository.
The publishing workflow is a **defense-in-depth check**, not a substitute
for protecting the branches and tag creation permissions.

## Preparing a release (human or agent)

1. Confirm the target branch (`main` or `release/vMAJOR.MINOR`).
   Ensure the branch is protected and current; for servicing releases
   verify that the branch name matches the intended version.
2. Open a **PR** containing the intended VERSION bump and release notes /
   fixes. Review and run CI. Never edit VERSION in place on a protected branch.
3. Before requesting a human merge, run in a local development shell:
   ```bash
   just check ci
   just check release
   ```
   GoReleaser v2.18.3 is required for `just check release`. The latter
   generates a local snapshot only, validates four Linux/macOS archives,
   checksums, and the host executable. It does not publish.
4. After human approval, squash merge the PR and confirm required CI checks
   passed on its final commit.
5. Fetch the protected branch, confirm that its VERSION is the desired
   `MAJOR.MINOR.PATCH`, and confirm the Git worktree is clean.

## Publishing (human-controlled)

Example for `v0.0.1` on main, **only after the release workflow PR merges**:

```bash
git switch main
git pull --ff-only origin main
test "$(cat VERSION)" = 0.0.1
git tag -s v0.0.1 -m "Euxenion v0.0.1"   # signed tag if Git signing configured
git push origin v0.0.1
```

Use `git tag -a` instead of `-s` if tag signing is not configured.
For servicing releases, switch to the protected `release/vMAJOR.MINOR`
branch and create the matching patch tag at its current approved tip.
Run `bash scripts/check-release-tag.sh vMAJOR.MINOR.PATCH` locally
after fetching `origin/main` and the corresponding service branch;
the action runs this check again before uploading anything.

The tag push triggers the workflow, which checks source-version equality,
branch ancestry, test results, absence of an existing release, and creates
a **pre-release** on GitHub. The published artifacts are four
`xenri_<version>_<os>_<arch>.tar.gz` archives and `checksums.txt`.

After the workflow completes, verify the release page, SHA-256 checksums,
executable version, and immutable-release indicator when enabled. Verify
published source/tag and release attestations independently as appropriate.
Checksums alone **do not** prove artifact authenticity.

**Never reuse or move a version tag to fix a release.** If publication fails,
investigate the workflow, document the partial state, and explicitly decide
on a recovery or next patch release before any further release action.
