#!/usr/bin/env bash
set -euo pipefail

TAG_VERSION="${1:-}"
if [ -z "$TAG_VERSION" ]; then
    echo "Error: TAG_VERSION argument required"
    echo "Usage: $0 <version>"
    exit 1
fi

echo "Validating version: v${TAG_VERSION}"
if ! echo "$TAG_VERSION" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$'; then
    echo "Error: TAG_VERSION is not in semver format (major.minor.patch)"
    exit 1
fi

BRANCH="$(git rev-parse --abbrev-ref HEAD)"
if [ "$BRANCH" != "main" ]; then
    echo "Error: Not on main branch (current: ${BRANCH})"
    exit 1
fi

git fetch origin --prune-tags --prune

if git ls-remote origin "refs/tags/v${TAG_VERSION}" | grep -q .; then
    echo "WARNING: Tag v${TAG_VERSION} already exists on remote"
    exit 0
fi

HEAD_COMMIT="$(git rev-parse HEAD)"
echo "Checking CI status for commit ${HEAD_COMMIT}"
if command -v gh &>/dev/null; then
    run_id="$(gh run list \
        --commit "${HEAD_COMMIT}" \
        --workflow ci.yml \
        --status success \
        --limit 1 \
        --json databaseId \
        --jq '.[0].databaseId' 2>/dev/null || true)"
    if [ -z "${run_id}" ] || [ "${run_id}" = "null" ]; then
        echo "Error: No successful CI run found for commit ${HEAD_COMMIT}"
        echo "Wait for CI to pass before tagging."
        exit 1
    fi
    echo "CI passed (run ${run_id})"
else
    echo "WARNING: gh not found, skipping CI check"
fi

echo "Creating and pushing tag v${TAG_VERSION}"
git tag -f "v${TAG_VERSION}"
git push origin "v${TAG_VERSION}"
