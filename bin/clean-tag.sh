#!/usr/bin/env bash
set -euo pipefail

TAG_VERSION="${1:-}"
if [ -z "$TAG_VERSION" ]; then
    echo "Error: TAG_VERSION argument required"
    echo "Usage: $0 <version>"
    exit 1
fi

if ! echo "$TAG_VERSION" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$'; then
    echo "Error: TAG_VERSION is not in semver format (major.minor.patch)"
    exit 1
fi

TAG="v${TAG_VERSION}"

if git tag --list "$TAG" | grep -q .; then
    echo "Deleting local tag ${TAG}"
    git tag -d "$TAG"
else
    echo "Local tag ${TAG} not found, skipping"
fi

if git ls-remote --tags origin "$TAG" | grep -q .; then
    echo "Deleting remote tag ${TAG}"
    git push origin --delete "$TAG"
else
    echo "Remote tag ${TAG} not found, skipping"
fi
