# Release Process

## Steps

1. **Update the version** — edit `VERSION` with the new semver (e.g. `0.0.7`)

2. **Update the changelog** — add an entry to `CHANGELOG.md`:
   ```
   ## [0.0.7] - YYYY-MM-DD
   ```

3. **Run `make release`** — validates the version and changelog entry, checks that the
   release doesn't already exist on the remote, then runs lint and tests, commits all
   changes, and pushes to `main`.

4. **Wait for CI to pass** — the release workflow requires a successful CI run for the
   tagged commit. Check the Actions tab or run:
   ```
   gh run list --workflow ci.yml --limit 5
   ```

5. **Run `make tag`** — creates and pushes the `vX.Y.Z` tag, which triggers the release
   workflow to build and publish the GitHub release.
