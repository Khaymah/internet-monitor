# Prerelease notes

Every release candidate has its own notes file here, named after its tag
(`v2.0.4-rc.1.md`, `v2.0.4-rc.2.md`, …). Each file covers **only what changed
since the previous rc**, so whoever reviews an rc knows exactly what to look at.

- CI puts the file at the top of the GitHub release notes for that tag.
- A pull request fails the `Web UI` job if `.release-version` names a prerelease
  and its notes file is missing.
- When the version is final, the full summary goes in `CHANGELOG.md` and in the
  in-app "What's new" list. The rc files stay as the review history.

Each file has three parts:

1. **What changed:** the user-visible changes in this rc.
2. **How to test:** a checklist a reviewer can follow on a real machine.
3. **Known limits:** what is not done or not verified yet.
