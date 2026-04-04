# Release Process

## Steps

1. **Check the version** — look at the latest remote tag (`git tag --sort=-v:refname | head -1`)
   and confirm that `VERSION` is already bumped past it. If not, edit `VERSION` with the
   new semver (e.g. `0.0.11`).

2. **Run `make lint-all`** — runs all linters across the codebase. Fix any issues before
   writing the changelog.

3. **Run `make test-all`** — runs the full test suite (Go, bash, and e2e). Fix any issues before
   writing the changelog.

4. **Update the changelog** — only after lint and tests pass, add an entry to
   `CHANGELOG.md` documenting everything that changed:
   ```
   ## [0.0.11] - YYYY-MM-DD
   ```

5. **Run `make release`** — validates the version and changelog entry, checks that the
   release doesn't already exist on the remote, commits all changes (code, `VERSION`,
   and `CHANGELOG.md`), and pushes to `main`.

6. **Wait for CI to pass** — the release workflow requires a successful CI run for the
   tagged commit. Check the Actions tab or run:
   ```
   gh run list --workflow ci.yml --limit 5
   ```

7. **Run `make tag`** — once CI is green, manually create and push the `vX.Y.Z` tag,
   which triggers the release workflow to build and publish the GitHub release.
