#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-}"
if [ -z "$VERSION" ]; then
    echo "Error: VERSION argument required"
    echo "Usage: $0 <version>"
    exit 1
fi

REPO_ROOT="$(git -C "$(dirname "$0")" rev-parse --show-toplevel)"

echo "Validating version: v${VERSION}"
if ! echo "$VERSION" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$'; then
    echo "Error: VERSION is not in semver format (major.minor.patch)"
    exit 1
fi

if ! grep -q "## \[${VERSION}\]" "${REPO_ROOT}/CHANGELOG.md"; then
    echo "Error: No entry for [${VERSION}] found in CHANGELOG.md"
    exit 1
fi

git fetch origin --prune-tags --prune

if git log origin/main --oneline --grep="Release: ${VERSION}" -1 | grep -q .; then
    echo "WARNING: Release ${VERSION} already committed on remote"
    exit 0
fi

if git ls-remote origin "refs/tags/v${VERSION}" | grep -q .; then
    echo "WARNING: Release v${VERSION} already exists on remote"
    exit 0
fi

if [ -z "$(git -C "${REPO_ROOT}" status --porcelain)" ]; then
    echo "WARNING: No changes to commit for v${VERSION}"
else
    echo "Staging all changes"
    git -C "${REPO_ROOT}" add -A
    echo "Committing release ${VERSION}"
    git -C "${REPO_ROOT}" commit -m "Release: ${VERSION}"
fi

echo "Pushing to origin"
git -C "${REPO_ROOT}" push origin main
